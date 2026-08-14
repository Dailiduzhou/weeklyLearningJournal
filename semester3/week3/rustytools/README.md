# Rustytools

An idiomatic Rust port of the minimal tool-calling agent runtime. It uses
`openai-oxide`, whose generated types track the official OpenAI Python SDK, and
the OpenAI Responses API. The runtime provides compiled JSON Schema validation,
derived deadlines, explicit retry classification, duplicate-failure detection,
read-only PostgreSQL query whitelisting, structured `tracing` audit events, and
a safe public trace.

## Run

```sh
cp config.example.toml config.toml
cp .env.example .env
# Set OPENAI_API_KEY and, if needed, OPENAI_BASE_URL in .env.
cargo run -- "What is (17 + 5) * 3?"
```

Pass `--config path/to/config.toml` to select a configuration file. Without a
positional prompt the program starts an interactive loop; enter `exit` or
`quit` to stop. Results are pretty-printed JSON on stdout and structured audit
records are emitted on stderr.

Configuration precedence is defaults, then TOML, then `AGENT_*` environment
variables. Nested environment keys use double underscores, such as
`AGENT_MODEL__NAME` and `AGENT_DATABASE__DSN`. Conventional `OPENAI_API_KEY`
and `OPENAI_BASE_URL` fill otherwise-empty model fields. Process environment
values take precedence over `.env` values.

## PostgreSQL

The model receives only predefined query names and parameter rules; it can
never supply SQL. Enable the database in `config.toml`, provide
`AGENT_DATABASE__DSN`, and use a genuinely read-only account. Every query runs
in a read-only transaction with timeout, row, and byte limits.
TLS-required PostgreSQL endpoints are supported through Rustls using the
platform's native root certificates; cancellation requests use the same TLS
policy as the main connection.

For a local database:

```sh
export POSTGRES_ADMIN_PASSWORD='local-admin-password'
export AGENT_DB_PASSWORD='local-readonly-password'
docker compose up -d
export AGENT_DATABASE__DSN='postgres://agent_readonly:local-readonly-password@localhost:5432/agent'
```

## Verify

```sh
cargo fmt --check
cargo test
cargo clippy --all-targets --all-features -- -D warnings
```

The agent-loop tests use a scripted model and never contact OpenAI.
