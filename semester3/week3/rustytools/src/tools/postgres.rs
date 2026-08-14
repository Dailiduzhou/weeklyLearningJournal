use std::{
    collections::{BTreeMap, HashMap, HashSet},
    sync::Arc,
    time::Duration,
};

use async_trait::async_trait;
use futures_util::TryStreamExt;
use serde::Deserialize;
use serde_json::{Map, Value, json};
use thiserror::Error;
use tokio::sync::Mutex;
use tokio_postgres::{
    CancelToken, Client, IsolationLevel, NoTls,
    types::{Json, ToSql},
};
use tokio_postgres_rustls::MakeRustlsConnect;
use tracing::{debug, instrument, warn};

use crate::{
    config::{QueryConfig, QueryParamConfig},
    tool::{Tool, ToolDefinition, ToolKind, ToolResult},
};

#[derive(Debug, Clone)]
pub struct ParamDefinition {
    pub name: String,
    pub kind: String,
    pub required: bool,
    pub max_length: usize,
    pub minimum: Option<f64>,
    pub maximum: Option<f64>,
}

impl From<&QueryParamConfig> for ParamDefinition {
    fn from(value: &QueryParamConfig) -> Self {
        Self {
            name: value.name.clone(),
            kind: value.kind.clone(),
            required: value.required,
            max_length: value.max_length,
            minimum: value.minimum,
            maximum: value.maximum,
        }
    }
}

#[derive(Debug, Clone)]
pub struct QueryDefinition {
    pub name: String,
    pub description: String,
    pub sql: String,
    pub params: Vec<ParamDefinition>,
}

impl From<&QueryConfig> for QueryDefinition {
    fn from(value: &QueryConfig) -> Self {
        Self {
            name: value.name.clone(),
            description: value.description.clone(),
            sql: value.sql.clone(),
            params: value.params.iter().map(Into::into).collect(),
        }
    }
}

pub struct PostgresQuery {
    client: Option<ClientHandle>,
    queries: HashMap<String, QueryDefinition>,
    query_timeout: Duration,
    max_rows: usize,
    max_bytes: usize,
    parameters: Value,
}

struct ClientHandle {
    client: Arc<Mutex<Client>>,
    tls: PostgresTls,
}

/// TLS strategy shared by the main `PostgreSQL` connection and cancellation
/// connections. A server that requires TLS also requires it for cancel
/// requests.
#[derive(Clone, Default)]
pub enum PostgresTls {
    #[default]
    Disabled,
    Rustls(MakeRustlsConnect),
}

impl PostgresTls {
    async fn cancel(&self, token: CancelToken) -> Result<(), tokio_postgres::Error> {
        match self {
            Self::Disabled => token.cancel_query(NoTls).await,
            Self::Rustls(connector) => token.cancel_query(connector.clone()).await,
        }
    }
}

/// Cancels the backend operation if the future owning this guard is dropped.
/// This covers outer runtime/task/caller deadlines, not only this tool's own
/// query timeout.
struct CancellationGuard {
    token: Option<CancelToken>,
    tls: PostgresTls,
}

impl CancellationGuard {
    fn new(token: CancelToken, tls: PostgresTls) -> Self {
        Self {
            token: Some(token),
            tls,
        }
    }

    fn disarm(&mut self) {
        self.token = None;
    }
}

impl Drop for CancellationGuard {
    fn drop(&mut self) {
        let Some(token) = self.token.take() else {
            return;
        };
        let tls = self.tls.clone();
        let Ok(runtime) = tokio::runtime::Handle::try_current() else {
            warn!("could not cancel dropped database query: Tokio runtime unavailable");
            return;
        };
        runtime.spawn(async move {
            if let Err(error) = tls.cancel(token).await {
                warn!(error = %error, "could not cancel dropped database query on server");
            } else {
                debug!("canceled dropped database query on server");
            }
        });
    }
}

impl std::fmt::Debug for PostgresQuery {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        formatter
            .debug_struct("PostgresQuery")
            .field("enabled", &self.client.is_some())
            .field("queries", &self.queries.keys())
            .field("query_timeout", &self.query_timeout)
            .field("max_rows", &self.max_rows)
            .field("max_bytes", &self.max_bytes)
            .finish_non_exhaustive()
    }
}

impl PostgresQuery {
    #[must_use]
    pub fn new(
        client: Option<Client>,
        queries: impl IntoIterator<Item = QueryDefinition>,
        query_timeout: Duration,
        max_rows: usize,
        max_bytes: usize,
    ) -> Self {
        Self::new_with_tls(
            client,
            PostgresTls::Disabled,
            queries,
            query_timeout,
            max_rows,
            max_bytes,
        )
    }

    #[must_use]
    pub fn new_with_tls(
        client: Option<Client>,
        tls: PostgresTls,
        queries: impl IntoIterator<Item = QueryDefinition>,
        query_timeout: Duration,
        max_rows: usize,
        max_bytes: usize,
    ) -> Self {
        let queries: Vec<QueryDefinition> = queries.into_iter().collect();
        let parameters = build_schema(&queries);
        Self {
            client: client.map(|client| ClientHandle {
                client: Arc::new(Mutex::new(client)),
                tls,
            }),
            queries: queries
                .into_iter()
                .map(|query| (query.name.clone(), query))
                .collect(),
            query_timeout,
            max_rows,
            max_bytes,
            parameters,
        }
    }

    #[must_use]
    pub fn from_config(
        client: Option<Client>,
        queries: &[QueryConfig],
        query_timeout: Duration,
        max_rows: usize,
        max_bytes: usize,
    ) -> Self {
        Self::new(
            client,
            queries.iter().map(QueryDefinition::from),
            query_timeout,
            max_rows,
            max_bytes,
        )
    }

    #[must_use]
    pub fn from_config_with_tls(
        client: Option<Client>,
        tls: PostgresTls,
        queries: &[QueryConfig],
        query_timeout: Duration,
        max_rows: usize,
        max_bytes: usize,
    ) -> Self {
        Self::new_with_tls(
            client,
            tls,
            queries.iter().map(QueryDefinition::from),
            query_timeout,
            max_rows,
            max_bytes,
        )
    }

    fn description(&self) -> String {
        let mut queries: Vec<_> = self.queries.values().collect();
        queries.sort_by(|left, right| left.name.cmp(&right.name));
        let available = if queries.is_empty() {
            "none configured".into()
        } else {
            queries
                .iter()
                .map(|query| format!("{} ({})", query.name, query.description))
                .collect::<Vec<_>>()
                .join("; ")
        };
        format!(
            "Run one predefined read-only PostgreSQL query. SQL text is never accepted. Available queries: {available}"
        )
    }
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Arguments {
    query: String,
    #[serde(default)]
    params: Map<String, Value>,
}

#[async_trait]
impl Tool for PostgresQuery {
    fn definition(&self) -> ToolDefinition {
        ToolDefinition {
            name: "database_query".into(),
            description: self.description(),
            parameters: self.parameters.clone(),
            kind: ToolKind::Read,
        }
    }

    #[instrument(skip(self, arguments))]
    async fn execute(&self, arguments: Value) -> ToolResult {
        let arguments: Arguments = match serde_json::from_value(arguments) {
            Ok(arguments) => arguments,
            Err(error) => {
                return ToolResult::failure("invalid_arguments", error.to_string(), false);
            }
        };
        let Some(query) = self.queries.get(&arguments.query) else {
            return ToolResult::failure(
                "query_not_allowed",
                "query is not in the configured whitelist",
                false,
            );
        };
        let values = match validate_params(&query.params, &arguments.params) {
            Ok(values) => values,
            Err(error) => {
                return ToolResult::failure("invalid_arguments", error.to_string(), false);
            }
        };
        let Some(client) = &self.client else {
            return ToolResult::failure("database_unavailable", "database is not enabled", true);
        };

        match tokio::time::timeout(self.query_timeout, self.run(client, query, &values)).await {
            Err(_) => {
                warn!(query = query.name, "database query timed out");
                ToolResult::failure("timeout", "database query timed out", true)
            }
            Ok(Err(error)) => {
                warn!(query = query.name, error = %error, "database query failed");
                ToolResult::failure("database_error", error.to_string(), true)
            }
            Ok(Ok((columns, rows, truncated))) => {
                debug!(
                    query = query.name,
                    rows = rows.len(),
                    truncated,
                    "database query completed"
                );
                let row_count = rows.len();
                let mut result = ToolResult::success(
                    json!({ "columns": columns, "rows": rows, "row_count": row_count }),
                    format!("returned {row_count} rows"),
                );
                result.truncated = truncated;
                result
            }
        }
    }
}

impl PostgresQuery {
    async fn run(
        &self,
        client: &ClientHandle,
        query: &QueryDefinition,
        values: &[ParamValue],
    ) -> Result<(Vec<String>, Vec<Value>, bool), tokio_postgres::Error> {
        let mut database = client.client.lock().await;
        let mut cancellation = CancellationGuard::new(database.cancel_token(), client.tls.clone());
        let result = async {
            let transaction = database
                .build_transaction()
                .isolation_level(IsolationLevel::ReadCommitted)
                .read_only(true)
                .start()
                .await?;

            // PostgreSQL itself performs the type-to-JSON conversion. This
            // covers timestamps, UUIDs, numeric values, bytea and extensions
            // without lossy client-side guesses, while SQL remains entirely
            // configuration-owned.
            let configured_sql = query.sql.trim().trim_end_matches(';');
            let statement = transaction.prepare(configured_sql).await?;
            let columns = statement
                .columns()
                .iter()
                .map(|column| column.name().to_owned())
                .collect();
            let wrapped = format!(
                "SELECT to_jsonb(__rustytools_row) AS __rustytools_row_json FROM ({configured_sql}) AS __rustytools_row"
            );
            let parameters = values.iter().map(ParamValue::as_sql);
            let mut stream = Box::pin(transaction.query_raw(&wrapped, parameters).await?);

            let mut rows = Vec::with_capacity(self.max_rows.min(16));
            let mut bytes: usize = 0;
            let mut truncated = false;
            while let Some(row) = stream.try_next().await? {
                if rows.len() >= self.max_rows {
                    truncated = true;
                    break;
                }
                let Json(value): Json<Value> = row.try_get(0)?;
                let encoded = serde_json::to_vec(&value)
                    .expect("serde_json::Value serialization is infallible");
                if bytes.saturating_add(encoded.len()) > self.max_bytes {
                    truncated = true;
                    break;
                }
                bytes += encoded.len();
                rows.push(value);
            }
            drop(stream);
            transaction.rollback().await?;
            Ok((columns, rows, truncated))
        }
        .await;
        cancellation.disarm();
        result
    }
}

#[derive(Debug)]
enum ParamValue {
    String(Option<String>),
    Integer(Option<i64>),
    Number(Option<f64>),
    Boolean(Option<bool>),
}

impl ParamValue {
    fn as_sql(&self) -> &(dyn ToSql + Sync) {
        match self {
            Self::String(value) => value,
            Self::Integer(value) => value,
            Self::Number(value) => value,
            Self::Boolean(value) => value,
        }
    }
}

#[derive(Debug, Error, PartialEq)]
enum ParameterError {
    #[error("missing required parameter {0:?}")]
    Missing(String),
    #[error("unknown parameter {0:?}")]
    Unknown(String),
    #[error("parameter {name:?}: must be a {kind}")]
    Type { name: String, kind: &'static str },
    #[error("parameter {name:?}: exceeds maximum length {maximum}")]
    TooLong { name: String, maximum: usize },
    #[error("parameter {name:?}: must be at least {minimum}")]
    TooSmall { name: String, minimum: f64 },
    #[error("parameter {name:?}: must be at most {maximum}")]
    TooLarge { name: String, maximum: f64 },
    #[error("parameter {0:?}: unsupported configured type")]
    Unsupported(String),
}

fn validate_params(
    definitions: &[ParamDefinition],
    raw: &Map<String, Value>,
) -> Result<Vec<ParamValue>, ParameterError> {
    let known: HashSet<_> = definitions
        .iter()
        .map(|definition| definition.name.as_str())
        .collect();
    if let Some(unknown) = raw.keys().find(|name| !known.contains(name.as_str())) {
        return Err(ParameterError::Unknown(unknown.clone()));
    }

    definitions
        .iter()
        .map(|definition| {
            let Some(value) = raw.get(&definition.name) else {
                if definition.required {
                    return Err(ParameterError::Missing(definition.name.clone()));
                }
                return match definition.kind.as_str() {
                    "string" => Ok(ParamValue::String(None)),
                    "integer" => Ok(ParamValue::Integer(None)),
                    "number" => Ok(ParamValue::Number(None)),
                    "boolean" => Ok(ParamValue::Boolean(None)),
                    _ => Err(ParameterError::Unsupported(definition.name.clone())),
                };
            };
            decode_param(definition, value)
        })
        .collect()
}

fn decode_param(definition: &ParamDefinition, value: &Value) -> Result<ParamValue, ParameterError> {
    match definition.kind.as_str() {
        "string" => {
            let value = value.as_str().ok_or_else(|| ParameterError::Type {
                name: definition.name.clone(),
                kind: "string",
            })?;
            if definition.max_length > 0 && value.chars().count() > definition.max_length {
                return Err(ParameterError::TooLong {
                    name: definition.name.clone(),
                    maximum: definition.max_length,
                });
            }
            Ok(ParamValue::String(Some(value.into())))
        }
        "integer" => {
            let value = value.as_i64().ok_or_else(|| ParameterError::Type {
                name: definition.name.clone(),
                kind: "integer",
            })?;
            // Numeric limits are configured as JSON numbers (`f64`), so integer
            // parameters must use the same comparison domain.
            #[allow(clippy::cast_precision_loss)]
            numeric_range(value as f64, definition)?;
            Ok(ParamValue::Integer(Some(value)))
        }
        "number" => {
            let value = value
                .as_f64()
                .filter(|value| value.is_finite())
                .ok_or_else(|| ParameterError::Type {
                    name: definition.name.clone(),
                    kind: "number",
                })?;
            numeric_range(value, definition)?;
            Ok(ParamValue::Number(Some(value)))
        }
        "boolean" => value
            .as_bool()
            .map(|value| ParamValue::Boolean(Some(value)))
            .ok_or_else(|| ParameterError::Type {
                name: definition.name.clone(),
                kind: "boolean",
            }),
        _ => Err(ParameterError::Unsupported(definition.name.clone())),
    }
}

fn numeric_range(value: f64, definition: &ParamDefinition) -> Result<(), ParameterError> {
    if definition.minimum.is_some_and(|minimum| value < minimum) {
        return Err(ParameterError::TooSmall {
            name: definition.name.clone(),
            minimum: definition.minimum.expect("checked above"),
        });
    }
    if definition.maximum.is_some_and(|maximum| value > maximum) {
        return Err(ParameterError::TooLarge {
            name: definition.name.clone(),
            maximum: definition.maximum.expect("checked above"),
        });
    }
    Ok(())
}

fn build_schema(queries: &[QueryDefinition]) -> Value {
    if queries.is_empty() {
        return json!({
            "type": "object",
            "properties": {
                "query": { "type": "string", "minLength": 1, "maxLength": 64 },
                "params": { "type": "object" }
            },
            "required": ["query", "params"],
            "additionalProperties": false
        });
    }

    let variants: Vec<Value> = queries
        .iter()
        .map(|query| {
            let mut properties = BTreeMap::new();
            let mut required = Vec::new();
            for parameter in &query.params {
                let mut schema = Map::new();
                schema.insert("type".into(), Value::String(parameter.kind.clone()));
                if parameter.max_length > 0 {
                    schema.insert("maxLength".into(), parameter.max_length.into());
                }
                if let Some(minimum) = parameter.minimum {
                    schema.insert("minimum".into(), json!(minimum));
                }
                if let Some(maximum) = parameter.maximum {
                    schema.insert("maximum".into(), json!(maximum));
                }
                properties.insert(parameter.name.clone(), Value::Object(schema));
                if parameter.required {
                    required.push(parameter.name.clone());
                }
            }
            let mut param_schema = json!({
                "type": "object",
                "properties": properties,
                "additionalProperties": false
            });
            if !required.is_empty() {
                param_schema["required"] = json!(required);
            }
            json!({
                "type": "object",
                "properties": {
                    "query": { "const": query.name },
                    "params": param_schema
                },
                "required": ["query", "params"],
                "additionalProperties": false
            })
        })
        .collect();
    json!({ "type": "object", "oneOf": variants })
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use super::*;
    use crate::tool::ToolRegistry;

    fn parameter(name: &str, kind: &str) -> ParamDefinition {
        ParamDefinition {
            name: name.into(),
            kind: kind.into(),
            required: true,
            max_length: 0,
            minimum: None,
            maximum: None,
        }
    }

    #[test]
    fn validates_parameter_order_types_and_unknowns() {
        let definitions = vec![parameter("id", "integer"), parameter("active", "boolean")];
        let raw = json!({ "active": true, "id": 42 })
            .as_object()
            .unwrap()
            .clone();
        let values = validate_params(&definitions, &raw).unwrap();
        assert!(matches!(values[0], ParamValue::Integer(Some(42))));
        assert!(matches!(values[1], ParamValue::Boolean(Some(true))));

        let unknown = json!({ "id": 42, "active": true, "sql": "drop table users" })
            .as_object()
            .unwrap()
            .clone();
        assert!(matches!(
            validate_params(&definitions, &unknown),
            Err(ParameterError::Unknown(name)) if name == "sql"
        ));
    }

    #[test]
    fn schema_contains_only_whitelisted_query_names() {
        let schema = build_schema(&[QueryDefinition {
            name: "user_by_id".into(),
            description: "Find a user".into(),
            sql: "select * from users where id = $1".into(),
            params: vec![parameter("id", "integer")],
        }]);
        assert_eq!(
            schema["oneOf"][0]["properties"]["query"]["const"],
            "user_by_id"
        );
        assert_eq!(
            schema["oneOf"][0]["properties"]["params"]["additionalProperties"],
            false
        );
    }

    #[tokio::test]
    async fn whitelist_is_checked_before_database_availability() {
        let tool = PostgresQuery::new(
            None,
            [QueryDefinition {
                name: "user_by_id".into(),
                description: "Find a user".into(),
                sql: "select name from users where id = $1".into(),
                params: vec![parameter("id", "integer")],
            }],
            Duration::from_secs(1),
            10,
            1_024,
        );

        let denied = tool
            .execute(json!({ "query": "raw_sql", "params": {} }))
            .await;
        assert_eq!(denied.error.unwrap().code, "query_not_allowed");

        let unavailable = tool
            .execute(json!({ "query": "user_by_id", "params": { "id": 7 } }))
            .await;
        let error = unavailable.error.unwrap();
        assert_eq!(error.code, "database_unavailable");
        assert!(error.retryable);
    }

    #[test]
    fn registry_rejects_non_whitelisted_query_and_raw_sql_fields() {
        let tool = PostgresQuery::new(
            None,
            [QueryDefinition {
                name: "health".into(),
                description: "Health check".into(),
                sql: "select true as healthy".into(),
                params: Vec::new(),
            }],
            Duration::from_secs(1),
            10,
            1_024,
        );
        let registry = ToolRegistry::new([Arc::new(tool) as Arc<dyn Tool>]).unwrap();

        registry
            .validate(
                "database_query",
                &json!({ "query": "health", "params": {} }),
            )
            .unwrap();
        assert!(
            registry
                .validate("database_query", &json!({ "query": "other", "params": {} }))
                .is_err()
        );
        assert!(
            registry
                .validate(
                    "database_query",
                    &json!({ "query": "health", "params": {}, "sql": "drop table users" }),
                )
                .is_err()
        );
    }
}
