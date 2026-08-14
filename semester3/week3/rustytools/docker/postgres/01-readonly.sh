#!/bin/sh
set -eu

psql --set=ON_ERROR_STOP=1 --set=readonly_password="$AGENT_DB_PASSWORD" --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
CREATE ROLE agent_readonly LOGIN PASSWORD :'readonly_password';
ALTER ROLE agent_readonly SET default_transaction_read_only = on;
GRANT CONNECT ON DATABASE agent TO agent_readonly;
GRANT USAGE ON SCHEMA public TO agent_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO agent_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO agent_readonly;
SQL

