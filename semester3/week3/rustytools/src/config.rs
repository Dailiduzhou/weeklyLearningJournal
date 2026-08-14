//! Strongly typed application configuration.
//!
//! Precedence is: built-in defaults, TOML, `.env`, then the process
//! environment. `AGENT_` keys use a double underscore between nested fields,
//! for example `AGENT_MODEL__MAX_RETRIES`.

use std::{
    collections::{HashMap, HashSet},
    env, fs,
    path::{Path, PathBuf},
    time::Duration,
};

use serde::{Deserialize, Serialize};
use thiserror::Error;
use tracing::{debug, warn};

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
pub struct Config {
    pub model: ModelConfig,
    pub agent: AgentConfig,
    pub documents: DocumentsConfig,
    pub database: DatabaseConfig,
    pub audit: AuditConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct ModelConfig {
    pub api_key: String,
    pub base_url: String,
    pub name: String,
    #[serde(with = "duration")]
    pub timeout: Duration,
    pub max_retries: usize,
}

impl Default for ModelConfig {
    fn default() -> Self {
        Self {
            api_key: String::new(),
            base_url: String::new(),
            name: "gpt-5.6-terra".into(),
            timeout: Duration::from_secs(30),
            max_retries: 2,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct AgentConfig {
    pub max_rounds: usize,
    #[serde(with = "duration")]
    pub task_timeout: Duration,
    #[serde(with = "duration")]
    pub tool_timeout: Duration,
    pub max_tool_result_bytes: usize,
    pub max_history_bytes: usize,
    pub max_repeated_failures: usize,
    pub max_unknown_tools: usize,
}

impl Default for AgentConfig {
    fn default() -> Self {
        Self {
            max_rounds: 8,
            task_timeout: Duration::from_mins(2),
            tool_timeout: Duration::from_secs(10),
            max_tool_result_bytes: 32_768,
            max_history_bytes: 262_144,
            max_repeated_failures: 3,
            max_unknown_tools: 3,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct DocumentsConfig {
    pub directory: PathBuf,
    pub chunk_runes: usize,
    pub max_results: usize,
}

impl Default for DocumentsConfig {
    fn default() -> Self {
        Self {
            directory: PathBuf::from("./docs"),
            chunk_runes: 1_200,
            max_results: 5,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct DatabaseConfig {
    pub enabled: bool,
    pub dsn: String,
    #[serde(with = "duration")]
    pub query_timeout: Duration,
    pub max_rows: usize,
    pub max_bytes: usize,
    pub queries: Vec<QueryConfig>,
}

impl Default for DatabaseConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            dsn: String::new(),
            query_timeout: Duration::from_secs(3),
            max_rows: 100,
            max_bytes: 65_536,
            queries: Vec::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
pub struct QueryConfig {
    pub name: String,
    pub description: String,
    pub sql: String,
    pub params: Vec<QueryParamConfig>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
pub struct QueryParamConfig {
    pub name: String,
    #[serde(rename = "type")]
    pub kind: String,
    pub required: bool,
    pub max_length: usize,
    pub minimum: Option<f64>,
    pub maximum: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(default)]
pub struct AuditConfig {
    pub level: String,
}

impl Default for AuditConfig {
    fn default() -> Self {
        Self {
            level: "info".into(),
        }
    }
}

#[derive(Debug, Error)]
pub enum ConfigError {
    #[error("read .env: {0}")]
    Dotenv(#[from] dotenvy::Error),
    #[error("read config {path}: {source}")]
    Read {
        path: PathBuf,
        #[source]
        source: std::io::Error,
    },
    #[error("decode config {path}: {source}")]
    Decode {
        path: PathBuf,
        #[source]
        source: toml::de::Error,
    },
    #[error("invalid environment value for {key}: expected {expected}, got {value:?}")]
    Environment {
        key: String,
        value: String,
        expected: &'static str,
    },
    #[error(transparent)]
    Validation(#[from] ValidationError),
}

#[derive(Debug, Error, PartialEq)]
pub enum ValidationError {
    #[error("model API key is required (AGENT_MODEL__API_KEY or OPENAI_API_KEY)")]
    MissingApiKey,
    #[error("invalid model configuration")]
    InvalidModel,
    #[error("invalid agent limits")]
    InvalidAgentLimits,
    #[error("invalid document search limits")]
    InvalidDocumentLimits,
    #[error("database DSN is required when database is enabled")]
    MissingDatabaseDsn,
    #[error("invalid database limits")]
    InvalidDatabaseLimits,
    #[error("database query name and SQL are required")]
    IncompleteDatabaseQuery,
    #[error("duplicate database query {0:?}")]
    DuplicateDatabaseQuery(String),
    #[error("query {query:?} has invalid parameter {parameter:?}")]
    InvalidQueryParameter { query: String, parameter: String },
    #[error("query {query:?} has duplicate parameter {parameter:?}")]
    DuplicateQueryParameter { query: String, parameter: String },
    #[error("query {query:?} has invalid limits for parameter {parameter:?}")]
    InvalidParameterLimits { query: String, parameter: String },
}

/// Loads configuration from an optional TOML file.
///
/// When `path` is `None`, `config.toml` in the current directory is used if it
/// exists. A missing implicit file is not an error; a missing explicit file is.
///
/// # Errors
///
/// Returns [`ConfigError`] when dotenv or TOML input cannot be read or parsed,
/// an environment override is invalid, or the resulting configuration fails
/// validation.
pub fn load(path: Option<&Path>) -> Result<Config, ConfigError> {
    let dotenv = read_dotenv()?;
    let selected = path.map_or_else(|| "config.toml".into(), Path::to_path_buf);

    let mut config = if selected.exists() {
        let text = fs::read_to_string(&selected).map_err(|source| ConfigError::Read {
            path: selected.clone(),
            source,
        })?;
        toml::from_str(&text).map_err(|source| ConfigError::Decode {
            path: selected.clone(),
            source,
        })?
    } else if path.is_some() {
        return Err(ConfigError::Read {
            path: selected.clone(),
            source: std::io::Error::new(std::io::ErrorKind::NotFound, "file not found"),
        });
    } else {
        Config::default()
    };

    if selected.exists() {
        debug!(path = %selected.display(), "loaded TOML configuration");
    } else {
        debug!("using built-in configuration defaults");
    }

    // `.env` behaves like a lower-precedence environment. Process values are
    // deliberately applied second so dotenv never mutates or wins over them.
    apply_agent_environment(
        &mut config,
        dotenv.iter().map(|(k, v)| (k.as_str(), v.as_str())),
    )?;
    let process: HashMap<String, String> = env::vars().collect();
    apply_agent_environment(
        &mut config,
        process.iter().map(|(k, v)| (k.as_str(), v.as_str())),
    )?;

    // Conventional OpenAI names are compatibility fallbacks, not overrides of
    // explicit application configuration.
    if config.model.api_key.is_empty() {
        config.model.api_key = compatibility_value("OPENAI_API_KEY", &process, &dotenv);
    }
    if config.model.base_url.is_empty() {
        config.model.base_url = compatibility_value("OPENAI_BASE_URL", &process, &dotenv);
    }

    config.validate()?;
    Ok(config)
}

impl Config {
    /// Loads and validates the application configuration.
    ///
    /// # Errors
    ///
    /// Returns [`ConfigError`] for unreadable or invalid configuration input.
    pub fn load(path: Option<&Path>) -> Result<Self, ConfigError> {
        load(path)
    }

    /// Validates all cross-field configuration invariants.
    ///
    /// # Errors
    ///
    /// Returns [`ValidationError`] when a required value is missing or a
    /// configured limit, query, or query parameter is invalid.
    pub fn validate(&self) -> Result<(), ValidationError> {
        if self.model.api_key.is_empty() {
            return Err(ValidationError::MissingApiKey);
        }
        if self.model.name.is_empty() || self.model.timeout.is_zero() {
            return Err(ValidationError::InvalidModel);
        }
        if self.agent.max_rounds == 0
            || self.agent.task_timeout.is_zero()
            || self.agent.tool_timeout.is_zero()
            || self.agent.max_tool_result_bytes < 256
            || self.agent.max_history_bytes < 1_024
            || self.agent.max_repeated_failures == 0
            || self.agent.max_unknown_tools == 0
        {
            return Err(ValidationError::InvalidAgentLimits);
        }
        if self.documents.chunk_runes == 0 || self.documents.max_results == 0 {
            return Err(ValidationError::InvalidDocumentLimits);
        }
        if !self.database.enabled {
            return Ok(());
        }
        if self.database.dsn.is_empty() {
            return Err(ValidationError::MissingDatabaseDsn);
        }
        if self.database.query_timeout.is_zero()
            || self.database.max_rows == 0
            || self.database.max_bytes < 256
        {
            return Err(ValidationError::InvalidDatabaseLimits);
        }

        let mut query_names = HashSet::with_capacity(self.database.queries.len());
        for query in &self.database.queries {
            if query.name.is_empty() || query.sql.is_empty() {
                return Err(ValidationError::IncompleteDatabaseQuery);
            }
            if !query_names.insert(&query.name) {
                return Err(ValidationError::DuplicateDatabaseQuery(query.name.clone()));
            }
            let mut parameter_names = HashSet::with_capacity(query.params.len());
            for parameter in &query.params {
                if parameter.name.is_empty()
                    || !matches!(
                        parameter.kind.as_str(),
                        "string" | "integer" | "number" | "boolean"
                    )
                {
                    return Err(ValidationError::InvalidQueryParameter {
                        query: query.name.clone(),
                        parameter: parameter.name.clone(),
                    });
                }
                if !parameter_names.insert(&parameter.name) {
                    return Err(ValidationError::DuplicateQueryParameter {
                        query: query.name.clone(),
                        parameter: parameter.name.clone(),
                    });
                }
                if parameter
                    .minimum
                    .zip(parameter.maximum)
                    .is_some_and(|(min, max)| min > max)
                {
                    return Err(ValidationError::InvalidParameterLimits {
                        query: query.name.clone(),
                        parameter: parameter.name.clone(),
                    });
                }
            }
        }
        Ok(())
    }
}

fn read_dotenv() -> Result<HashMap<String, String>, ConfigError> {
    match dotenvy::from_path_iter(".env") {
        Ok(entries) => entries
            .collect::<Result<HashMap<_, _>, _>>()
            .map_err(Into::into),
        Err(dotenvy::Error::Io(error)) if error.kind() == std::io::ErrorKind::NotFound => {
            Ok(HashMap::new())
        }
        Err(error) => Err(error.into()),
    }
}

fn compatibility_value(
    key: &str,
    process: &HashMap<String, String>,
    dotenv: &HashMap<String, String>,
) -> String {
    process
        .get(key)
        .filter(|value| !value.is_empty())
        .or_else(|| dotenv.get(key))
        .cloned()
        .unwrap_or_default()
}

fn apply_agent_environment<'a>(
    config: &mut Config,
    values: impl Iterator<Item = (&'a str, &'a str)>,
) -> Result<(), ConfigError> {
    for (key, value) in values.filter(|(key, value)| key.starts_with("AGENT_") && !value.is_empty())
    {
        let field = &key["AGENT_".len()..];
        match field {
            "MODEL__API_KEY" => config.model.api_key = value.into(),
            "MODEL__BASE_URL" => config.model.base_url = value.into(),
            "MODEL__NAME" => config.model.name = value.into(),
            "MODEL__TIMEOUT" => config.model.timeout = parse_duration(key, value)?,
            "MODEL__MAX_RETRIES" => {
                config.model.max_retries = parse_environment(key, value, "integer")?;
            }
            "AGENT__MAX_ROUNDS" => {
                config.agent.max_rounds = parse_environment(key, value, "positive integer")?;
            }
            "AGENT__TASK_TIMEOUT" => config.agent.task_timeout = parse_duration(key, value)?,
            "AGENT__TOOL_TIMEOUT" => config.agent.tool_timeout = parse_duration(key, value)?,
            "AGENT__MAX_TOOL_RESULT_BYTES" => {
                config.agent.max_tool_result_bytes =
                    parse_environment(key, value, "positive integer")?;
            }
            "AGENT__MAX_HISTORY_BYTES" => {
                config.agent.max_history_bytes = parse_environment(key, value, "positive integer")?;
            }
            "AGENT__MAX_REPEATED_FAILURES" => {
                config.agent.max_repeated_failures =
                    parse_environment(key, value, "positive integer")?;
            }
            "AGENT__MAX_UNKNOWN_TOOLS" => {
                config.agent.max_unknown_tools = parse_environment(key, value, "positive integer")?;
            }
            "DOCUMENTS__DIRECTORY" => config.documents.directory = value.into(),
            "DOCUMENTS__CHUNK_RUNES" => {
                config.documents.chunk_runes = parse_environment(key, value, "positive integer")?;
            }
            "DOCUMENTS__MAX_RESULTS" => {
                config.documents.max_results = parse_environment(key, value, "positive integer")?;
            }
            "DATABASE__ENABLED" => {
                config.database.enabled =
                    parse_bool(value).ok_or_else(|| environment_error(key, value, "boolean"))?;
            }
            "DATABASE__DSN" => config.database.dsn = value.into(),
            "DATABASE__QUERY_TIMEOUT" => {
                config.database.query_timeout = parse_duration(key, value)?;
            }
            "DATABASE__MAX_ROWS" => {
                config.database.max_rows = parse_environment(key, value, "positive integer")?;
            }
            "DATABASE__MAX_BYTES" => {
                config.database.max_bytes = parse_environment(key, value, "positive integer")?;
            }
            "AUDIT__LEVEL" => config.audit.level = value.into(),
            _ => warn!(
                environment_key = key,
                "ignoring unknown AGENT environment key"
            ),
        }
    }
    Ok(())
}

fn parse_environment<T>(key: &str, value: &str, expected: &'static str) -> Result<T, ConfigError>
where
    T: std::str::FromStr,
{
    value
        .parse()
        .map_err(|_| environment_error(key, value, expected))
}

fn environment_error(key: &str, value: &str, expected: &'static str) -> ConfigError {
    ConfigError::Environment {
        key: key.into(),
        value: value.into(),
        expected,
    }
}

fn parse_bool(value: &str) -> Option<bool> {
    match value.to_ascii_lowercase().as_str() {
        "true" | "1" | "yes" | "on" => Some(true),
        "false" | "0" | "no" | "off" => Some(false),
        _ => None,
    }
}

fn parse_duration(key: &str, value: &str) -> Result<Duration, ConfigError> {
    humantime::parse_duration(value).map_err(|_| ConfigError::Environment {
        key: key.into(),
        value: value.into(),
        expected: "duration such as 500ms, 10s, or 2m",
    })
}

mod duration {
    use std::time::Duration;

    use serde::{Deserialize, Deserializer, Serializer, de::Error as _};

    pub fn serialize<S>(value: &Duration, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&humantime::format_duration(*value).to_string())
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Duration, D::Error>
    where
        D: Deserializer<'de>,
    {
        let value = String::deserialize(deserializer)?;
        humantime::parse_duration(&value).map_err(D::Error::custom)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn toml_uses_defaults_and_parses_durations() {
        let config: Config = toml::from_str(
            r#"
                [model]
                api_key = "key"
                timeout = "750ms"

                [agent]
                max_rounds = 3
            "#,
        )
        .unwrap();

        assert_eq!(config.model.timeout, Duration::from_millis(750));
        assert_eq!(config.agent.max_rounds, 3);
        assert_eq!(config.agent.tool_timeout, Duration::from_secs(10));
        assert!(config.validate().is_ok());
    }

    #[test]
    fn enabled_database_validates_queries_and_parameters() {
        let mut config = Config::default();
        config.model.api_key = "key".into();
        config.database.enabled = true;
        config.database.dsn = "postgresql://localhost/example".into();
        config.database.queries = vec![QueryConfig {
            name: "lookup".into(),
            sql: "select $1".into(),
            params: vec![QueryParamConfig {
                name: "id".into(),
                kind: "integer".into(),
                minimum: Some(10.0),
                maximum: Some(1.0),
                ..Default::default()
            }],
            ..Default::default()
        }];

        assert!(matches!(
            config.validate(),
            Err(ValidationError::InvalidParameterLimits { .. })
        ));
    }

    #[test]
    fn empty_process_value_does_not_replace_dotenv_value() {
        let mut config = Config::default();
        apply_agent_environment(
            &mut config,
            [("AGENT_DATABASE__DSN", "postgresql://dotenv/example")].into_iter(),
        )
        .unwrap();
        apply_agent_environment(&mut config, [("AGENT_DATABASE__DSN", "")].into_iter()).unwrap();

        assert_eq!(config.database.dsn, "postgresql://dotenv/example");
    }

    #[test]
    fn toml_dotenv_and_process_environment_follow_precedence() {
        let mut config: Config = toml::from_str(
            r#"
                [model]
                api_key = "toml-key"
                name = "toml-model"

                [agent]
                max_rounds = 4
                tool_timeout = "7s"
            "#,
        )
        .unwrap();

        apply_agent_environment(
            &mut config,
            [
                ("AGENT_MODEL__API_KEY", "dotenv-key"),
                ("AGENT_MODEL__NAME", "dotenv-model"),
                ("AGENT_AGENT__TOOL_TIMEOUT", "5s"),
            ]
            .into_iter(),
        )
        .unwrap();
        apply_agent_environment(
            &mut config,
            [
                ("AGENT_MODEL__API_KEY", "process-key"),
                ("AGENT_MODEL__NAME", "process-model"),
                ("AGENT_AGENT__TOOL_TIMEOUT", "250ms"),
            ]
            .into_iter(),
        )
        .unwrap();

        assert_eq!(config.model.api_key, "process-key");
        assert_eq!(config.model.name, "process-model");
        assert_eq!(config.agent.max_rounds, 4);
        assert_eq!(config.agent.tool_timeout, Duration::from_millis(250));
        assert_eq!(config.agent.task_timeout, Duration::from_mins(2));
    }

    #[test]
    fn conventional_openai_values_prefer_process_over_dotenv() {
        let dotenv = HashMap::from([
            ("OPENAI_API_KEY".into(), "dotenv-key".into()),
            ("OPENAI_BASE_URL".into(), "https://dotenv.example/v1".into()),
        ]);
        let process = HashMap::from([
            ("OPENAI_API_KEY".into(), "process-key".into()),
            ("OPENAI_BASE_URL".into(), String::new()),
        ]);

        assert_eq!(
            compatibility_value("OPENAI_API_KEY", &process, &dotenv),
            "process-key"
        );
        assert_eq!(
            compatibility_value("OPENAI_BASE_URL", &process, &dotenv),
            "https://dotenv.example/v1"
        );
    }
}
