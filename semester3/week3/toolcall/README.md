# Minimal Agent Runtime

A framework-free Go implementation of a sequential Responses API tool loop. It includes compiled JSON Schema validation, derived context deadlines, retry classification, duplicate-failure detection, read-only PostgreSQL query whitelisting, structured audit logs, and a safe public trace.

## Run

Requires Go 1.25 or newer.

```sh
cp config.example.yaml config.yaml
export OPENAI_API_KEY='...'
go run . "What is (17 + 5) * 3?"
```

With a Responses-compatible OpenAI endpoint, also set `OPENAI_BASE_URL` and select a compatible model in `config.yaml`. Environment variables use the `AGENT_` prefix and double underscores for nesting, for example `AGENT_MODEL__NAME` and `AGENT_DATABASE__DSN`. Environment values override the YAML file, which overrides defaults.

Without a positional prompt the program starts an interactive loop. Enter `exit` or `quit` to stop. Results are JSON and include the safe execution trace; audit records are JSON lines on stderr.

## PostgreSQL

The model only receives predefined query names and parameter rules—it can never supply SQL. Enable the database in `config.yaml`, provide `AGENT_DATABASE__DSN`, and use a genuinely read-only account. The runtime also opens every query in a read-only transaction and enforces query timeout, row, and byte limits.

For a local database:

```sh
export POSTGRES_ADMIN_PASSWORD='local-admin-password'
export AGENT_DB_PASSWORD='local-readonly-password'
docker compose up -d
export AGENT_DATABASE__DSN='postgres://agent_readonly:local-readonly-password@localhost:5432/agent'
```

The initialization script creates `agent_readonly` and grants only schema usage and `SELECT`. Add the application schema as the admin account, then restart the agent with database support enabled.

## Test

```sh
go test ./...
go vet ./...
```

Agent-loop tests use a fake model client and never call a real model service.
