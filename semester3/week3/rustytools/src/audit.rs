use std::time::Duration;

use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

use crate::tool::ToolKind;

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct AuditEntry {
    pub task_id: String,
    pub round: usize,
    pub tool_name: String,
    #[serde(rename = "tool_type")]
    pub tool_kind: ToolKind,
    pub arguments_summary: Value,
    pub status: String,
    #[serde(with = "duration_nanos")]
    pub duration: Duration,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error_type: String,
    pub result_bytes: usize,
    pub truncated: bool,
}

pub trait AuditLogger: Send + Sync {
    fn log(&self, entry: &AuditEntry);
}

#[derive(Clone, Copy, Debug, Default)]
pub struct DiscardAuditLogger;

impl AuditLogger for DiscardAuditLogger {
    fn log(&self, _entry: &AuditEntry) {}
}

/// Emits structured audit records through the application's tracing subscriber.
#[derive(Clone, Copy, Debug, Default)]
pub struct TracingAuditLogger;

impl AuditLogger for TracingAuditLogger {
    fn log(&self, entry: &AuditEntry) {
        tracing::info!(
            target: "rustytools::audit",
            event = "tool_call",
            task_id = %entry.task_id,
            round = entry.round,
            tool_name = %entry.tool_name,
            tool_type = %entry.tool_kind,
            arguments_summary = %entry.arguments_summary,
            status = %entry.status,
            duration = entry.duration.as_nanos(),
            error_type = %entry.error_type,
            result_bytes = entry.result_bytes,
            truncated = entry.truncated,
        );
    }
}

#[must_use]
pub fn redact_arguments(arguments: &Value) -> Value {
    redact(arguments)
}

/// Parses raw model arguments for audit use without ever echoing malformed input.
pub fn redact_raw_arguments(arguments: &str) -> Value {
    serde_json::from_str(arguments).as_ref().map_or_else(
        |_| json!({ "invalid_json": true, "size_bytes": arguments.len() }),
        redact,
    )
}

fn redact(value: &Value) -> Value {
    match value {
        Value::Object(object) => Value::Object(
            object
                .iter()
                .map(|(key, child)| {
                    let value = if sensitive(key) {
                        Value::String("[REDACTED]".to_owned())
                    } else {
                        redact(child)
                    };
                    (key.clone(), value)
                })
                .collect(),
        ),
        Value::Array(items) => Value::Array(items.iter().map(redact).collect()),
        Value::String(text) if text.chars().count() > 200 => {
            Value::String(format!("{}…", text.chars().take(200).collect::<String>()))
        }
        _ => value.clone(),
    }
}

fn sensitive(key: &str) -> bool {
    let key = key.to_ascii_lowercase();
    [
        "password",
        "passwd",
        "secret",
        "token",
        "api_key",
        "apikey",
        "dsn",
        "connection_string",
    ]
    .iter()
    .any(|marker| key.contains(marker))
}

mod duration_nanos {
    use std::time::Duration;

    use serde::{Deserialize, Deserializer, Serializer, de::Error};

    pub fn serialize<S>(duration: &Duration, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u64(duration.as_nanos().try_into().unwrap_or(u64::MAX))
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Duration, D::Error>
    where
        D: Deserializer<'de>,
    {
        let nanos = u64::deserialize(deserializer).map_err(D::Error::custom)?;
        Ok(Duration::from_nanos(nanos))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn recursively_redacts_secrets_and_bounds_strings() {
        let arguments = json!({
            "query": "x".repeat(201),
            "nested": [{"api_token": "do not log"}],
            "password_hint": "also secret"
        });
        let redacted = redact_arguments(&arguments);

        assert_eq!(redacted["nested"][0]["api_token"], "[REDACTED]");
        assert_eq!(redacted["password_hint"], "[REDACTED]");
        assert_eq!(redacted["query"].as_str().unwrap().chars().count(), 201);
    }

    #[test]
    fn malformed_json_is_not_echoed() {
        assert_eq!(
            redact_raw_arguments("{secret"),
            json!({"invalid_json": true, "size_bytes": 7})
        );
    }

    #[test]
    fn audit_serialization_uses_lowercase_tool_type() {
        let entry = AuditEntry {
            task_id: "task".to_owned(),
            round: 1,
            tool_name: "reader".to_owned(),
            tool_kind: ToolKind::Read,
            arguments_summary: json!({}),
            status: "success".to_owned(),
            duration: Duration::ZERO,
            error_type: String::new(),
            result_bytes: 2,
            truncated: false,
        };

        let value = serde_json::to_value(entry).unwrap();
        assert_eq!(value["tool_type"], "read");
        assert!(value.get("tool_kind").is_none());
    }
}
