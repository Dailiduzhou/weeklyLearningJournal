use std::{
    io::{self, BufRead, Write},
    path::PathBuf,
    process::ExitCode,
    sync::Arc,
};

use clap::Parser;
use rustytools::{
    agent::{AgentRuntime, CancellationToken, RuntimeBuildError, RuntimeConfig},
    audit::TracingAuditLogger,
    config::{Config, ConfigError},
    openai_adapter::OpenAiClient,
    tool::{RegistryError, Tool, ToolRegistry},
    tools::{
        Calculator, DocumentSearch, PostgresQuery, docsearch::DocumentSearchError,
        postgres::PostgresTls,
    },
};
use thiserror::Error;
use tokio_postgres_rustls::MakeRustlsConnect;
use tracing::Level;

#[derive(Debug, Parser)]
#[command(version, about = "A minimal OpenAI Responses API tool-calling runtime")]
struct Cli {
    /// Path to TOML configuration.
    #[arg(long)]
    config: Option<PathBuf>,

    /// Prompt to run. Omit it to start an interactive session.
    prompt: Vec<String>,
}

#[derive(Debug, Error)]
enum AppError {
    #[error(transparent)]
    Config(#[from] ConfigError),
    #[error(transparent)]
    Documents(#[from] DocumentSearchError),
    #[error("connect database: {0}")]
    Database(#[from] tokio_postgres::Error),
    #[error(transparent)]
    Registry(#[from] RegistryError),
    #[error(transparent)]
    Runtime(#[from] RuntimeBuildError),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
    #[error(transparent)]
    Io(#[from] io::Error),
}

#[tokio::main]
async fn main() -> ExitCode {
    match run(Cli::parse()).await {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("{error}");
            ExitCode::FAILURE
        }
    }
}

async fn run(cli: Cli) -> Result<(), AppError> {
    let config = Config::load(cli.config.as_deref())?;
    init_tracing(&config);
    tracing::info!("configuration loaded");

    let documents = DocumentSearch::new(
        &config.documents.directory,
        config.documents.chunk_runes,
        config.documents.max_results,
    )?;

    let (database, database_tls) = if config.database.enabled {
        let tls = postgres_tls();
        let (client, connection) =
            tokio_postgres::connect(&config.database.dsn, tls.clone()).await?;
        tokio::spawn(async move {
            if let Err(error) = connection.await {
                tracing::error!(error = %error, "database connection stopped");
            }
        });
        (Some(client), PostgresTls::Rustls(tls))
    } else {
        (None, PostgresTls::Disabled)
    };
    let database_tool = PostgresQuery::from_config_with_tls(
        database,
        database_tls,
        &config.database.queries,
        config.database.query_timeout,
        config.database.max_rows,
        config.database.max_bytes,
    );

    let tools: Vec<Arc<dyn Tool>> = vec![
        Arc::new(Calculator::new()),
        Arc::new(documents),
        Arc::new(database_tool),
    ];
    let registry = Arc::new(ToolRegistry::new(tools)?);
    let model = Arc::new(OpenAiClient::new(
        &config.model.api_key,
        &config.model.base_url,
        &config.model.name,
    ));
    let runtime = AgentRuntime::new(
        RuntimeConfig {
            max_rounds: config.agent.max_rounds,
            task_timeout: config.agent.task_timeout,
            model_timeout: config.model.timeout,
            tool_timeout: config.agent.tool_timeout,
            max_tool_result_bytes: config.agent.max_tool_result_bytes,
            max_history_bytes: config.agent.max_history_bytes,
            max_repeated_failures: config.agent.max_repeated_failures,
            max_unknown_tools: config.agent.max_unknown_tools,
            model_retries: config.model.max_retries,
        },
        model,
        registry,
        Some(Arc::new(TracingAuditLogger)),
    )?;

    let cancellation = CancellationToken::new();
    watch_for_shutdown(cancellation.clone());
    let prompt = cli.prompt.join(" ").trim().to_owned();
    if prompt.is_empty() {
        interactive(&runtime, cancellation).await
    } else {
        write_result(&runtime.run_with_cancellation(prompt, cancellation).await)
    }
}

fn postgres_tls() -> MakeRustlsConnect {
    let native = rustls_native_certs::load_native_certs();
    for error in &native.errors {
        tracing::warn!(%error, "could not load a native root certificate");
    }
    let mut roots = rustls::RootCertStore::empty();
    let (_, invalid) = roots.add_parsable_certificates(native.certs);
    if invalid > 0 {
        tracing::warn!(invalid, "ignored invalid native root certificates");
    }
    let config = rustls::ClientConfig::builder()
        .with_root_certificates(roots)
        .with_no_client_auth();
    MakeRustlsConnect::new(config)
}

fn init_tracing(config: &Config) {
    let level = if config.audit.level.eq_ignore_ascii_case("debug") {
        Level::DEBUG
    } else {
        Level::INFO
    };
    tracing_subscriber::fmt()
        .json()
        .with_max_level(level)
        .with_writer(io::stderr)
        .init();
}

async fn interactive(
    runtime: &AgentRuntime,
    cancellation: CancellationToken,
) -> Result<(), AppError> {
    let stdin = io::stdin();
    let mut lines = stdin.lock().lines();
    loop {
        eprint!("> ");
        io::stderr().flush()?;
        let Some(line) = lines.next() else {
            return Ok(());
        };
        let prompt = line?.trim().to_owned();
        if prompt.is_empty() {
            continue;
        }
        if prompt.eq_ignore_ascii_case("exit") || prompt.eq_ignore_ascii_case("quit") {
            return Ok(());
        }
        let result = runtime
            .run_with_cancellation(prompt, cancellation.clone())
            .await;
        write_result(&result)?;
        if cancellation.is_cancelled() {
            return Ok(());
        }
    }
}

fn write_result(result: &rustytools::agent::RunResult) -> Result<(), AppError> {
    let stdout = io::stdout();
    let mut output = stdout.lock();
    serde_json::to_writer_pretty(&mut output, result)?;
    writeln!(output)?;
    Ok(())
}

fn watch_for_shutdown(cancellation: CancellationToken) {
    tokio::spawn(async move {
        #[cfg(unix)]
        {
            let mut terminate =
                tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
                    .expect("install SIGTERM handler");
            tokio::select! {
                result = tokio::signal::ctrl_c() => {
                    if let Err(error) = result {
                        tracing::error!(%error, "Ctrl-C handler failed");
                    }
                }
                _ = terminate.recv() => {}
            }
        }
        #[cfg(not(unix))]
        if let Err(error) = tokio::signal::ctrl_c().await {
            tracing::error!(%error, "Ctrl-C handler failed");
        }
        cancellation.cancel();
    });
}
