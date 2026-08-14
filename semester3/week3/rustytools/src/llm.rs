//! Provider-neutral language-model request and response types.

use std::{error::Error as StdError, fmt};

use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use thiserror::Error;

/// The author of a message in the model conversation.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Role {
    System,
    User,
    #[default]
    Assistant,
    Tool,
}

/// A conversation message.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct Message {
    pub role: Role,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub content: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool_call_id: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub tool_calls: Vec<ToolCall>,
}

impl Message {
    pub fn new(role: Role, content: impl Into<String>) -> Self {
        Self {
            role,
            content: content.into(),
            ..Self::default()
        }
    }

    pub fn system(content: impl Into<String>) -> Self {
        Self::new(Role::System, content)
    }

    pub fn user(content: impl Into<String>) -> Self {
        Self::new(Role::User, content)
    }

    pub fn assistant(content: impl Into<String>) -> Self {
        Self::new(Role::Assistant, content)
    }

    pub fn tool_result(call_id: impl Into<String>, content: impl Into<String>) -> Self {
        Self {
            role: Role::Tool,
            content: content.into(),
            tool_call_id: call_id.into(),
            tool_calls: Vec::new(),
        }
    }
}

/// A function call requested by the model.
///
/// `arguments` deliberately remains raw JSON text. The tool layer, rather than
/// the model adapter, owns argument validation; this also preserves malformed
/// model output for a useful `invalid_arguments` tool result.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ToolCall {
    pub id: String,
    pub name: String,
    pub arguments: String,
}

impl ToolCall {
    pub fn new(
        id: impl Into<String>,
        name: impl Into<String>,
        arguments: impl Into<String>,
    ) -> Self {
        Self {
            id: id.into(),
            name: name.into(),
            arguments: arguments.into(),
        }
    }
}

/// A function exposed to the model.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ToolSpec {
    pub name: String,
    pub description: String,
    #[serde(rename = "parameters")]
    pub schema: Value,
}

impl ToolSpec {
    pub fn new(name: impl Into<String>, description: impl Into<String>, schema: Value) -> Self {
        Self {
            name: name.into(),
            description: description.into(),
            schema,
        }
    }
}

#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
pub struct Request {
    pub messages: Vec<Message>,
    pub tools: Vec<ToolSpec>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Response {
    pub message: Message,
}

/// A model client suitable for runtime dependency injection and test doubles.
#[async_trait]
pub trait Client: Send + Sync {
    async fn complete(&self, request: Request) -> Result<Response, LlmError>;
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ErrorKind {
    Temporary,
    Permanent,
    InvalidResponse,
}

impl fmt::Display for ErrorKind {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(match self {
            Self::Temporary => "temporary",
            Self::Permanent => "permanent",
            Self::InvalidResponse => "invalid_response",
        })
    }
}

/// A classified model error used by the runtime's retry policy.
#[derive(Debug, Error)]
#[error("llm {kind} error: {source}")]
pub struct LlmError {
    pub kind: ErrorKind,
    pub retryable: bool,
    #[source]
    pub source: Box<dyn StdError + Send + Sync>,
}

impl LlmError {
    pub fn new<E>(kind: ErrorKind, retryable: bool, source: E) -> Self
    where
        E: StdError + Send + Sync + 'static,
    {
        Self {
            kind,
            retryable,
            source: Box::new(source),
        }
    }

    pub fn message(kind: ErrorKind, retryable: bool, message: impl Into<String>) -> Self {
        Self::new(kind, retryable, ErrorMessage(message.into()))
    }

    pub fn temporary<E>(source: E) -> Self
    where
        E: StdError + Send + Sync + 'static,
    {
        Self::new(ErrorKind::Temporary, true, source)
    }

    pub fn permanent<E>(source: E) -> Self
    where
        E: StdError + Send + Sync + 'static,
    {
        Self::new(ErrorKind::Permanent, false, source)
    }

    pub fn invalid_response(message: impl Into<String>) -> Self {
        Self::message(ErrorKind::InvalidResponse, false, message)
    }

    #[must_use]
    pub fn is_retryable(&self) -> bool {
        self.retryable
    }
}

#[derive(Debug, Error)]
#[error("{0}")]
struct ErrorMessage(String);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn messages_omit_empty_conversation_fields() {
        let encoded = serde_json::to_value(Message::user("hello")).expect("serialize message");
        assert_eq!(encoded["role"], "user");
        assert_eq!(encoded["content"], "hello");
        assert!(encoded.get("tool_call_id").is_none());
        assert!(encoded.get("tool_calls").is_none());
    }

    #[test]
    fn classified_error_exposes_retry_policy() {
        let error = LlmError::message(ErrorKind::Temporary, true, "busy");
        assert_eq!(error.kind, ErrorKind::Temporary);
        assert!(error.is_retryable());
        assert_eq!(error.to_string(), "llm temporary error: busy");
    }
}
