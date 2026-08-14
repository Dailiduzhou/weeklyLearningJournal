use std::{collections::HashMap, fmt, sync::Arc};

use async_trait::async_trait;
use jsonschema::Validator;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use thiserror::Error;

/// The side-effect class of a tool.
#[derive(Clone, Copy, Debug, Default, Eq, PartialEq, Serialize, Deserialize)]
pub enum ToolKind {
    #[default]
    #[serde(rename = "")]
    Unknown,
    #[serde(rename = "read")]
    Read,
    #[serde(rename = "write")]
    Write,
}

impl ToolKind {
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Unknown => "",
            Self::Read => "read",
            Self::Write => "write",
        }
    }
}

impl fmt::Display for ToolKind {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(self.as_str())
    }
}

#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
pub struct ToolDefinition {
    pub name: String,
    pub description: String,
    pub parameters: Value,
    pub kind: ToolKind,
}

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct ToolError {
    pub code: String,
    pub message: String,
    pub retryable: bool,
}

#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
pub struct ToolResult {
    pub ok: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<ToolError>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub summary: String,
    pub size_bytes: usize,
    pub truncated: bool,
}

impl ToolResult {
    pub fn success(data: impl Into<Value>, summary: impl Into<String>) -> Self {
        Self {
            ok: true,
            data: Some(data.into()),
            error: None,
            summary: summary.into(),
            size_bytes: 0,
            truncated: false,
        }
    }

    pub fn failure(code: impl Into<String>, message: impl Into<String>, retryable: bool) -> Self {
        Self {
            ok: false,
            data: None,
            error: Some(ToolError {
                code: code.into(),
                message: message.into(),
                retryable,
            }),
            summary: String::new(),
            size_bytes: 0,
            truncated: false,
        }
    }

    #[must_use]
    pub fn with_truncated(mut self, truncated: bool) -> Self {
        self.truncated = truncated;
        self
    }
}

#[async_trait]
pub trait Tool: Send + Sync {
    fn definition(&self) -> ToolDefinition;

    async fn execute(&self, arguments: Value) -> ToolResult;
}

struct RegisteredTool {
    tool: Arc<dyn Tool>,
    validator: Validator,
}

/// A validated, deterministically ordered collection of tools.
#[derive(Default)]
pub struct ToolRegistry {
    tools: HashMap<String, RegisteredTool>,
}

#[derive(Debug, Error)]
pub enum RegistryError {
    #[error("invalid tool definition {0:?}")]
    InvalidDefinition(String),
    #[error("duplicate tool {0:?}")]
    Duplicate(String),
    #[error("invalid JSON schema for tool {name:?}: {source}")]
    InvalidSchema {
        name: String,
        #[source]
        source: Box<jsonschema::ValidationError<'static>>,
    },
    #[error("unknown tool {0:?}")]
    UnknownTool(String),
    #[error("arguments for tool {name:?} do not match its schema: {message}")]
    InvalidArguments { name: String, message: String },
}

impl ToolRegistry {
    /// Builds a registry and validates every tool definition.
    ///
    /// # Errors
    ///
    /// Returns [`RegistryError`] when a definition is invalid, duplicated, or
    /// contains an invalid JSON schema.
    pub fn new(tools: impl IntoIterator<Item = Arc<dyn Tool>>) -> Result<Self, RegistryError> {
        let mut registry = Self::default();
        for tool in tools {
            registry.register(tool)?;
        }
        Ok(registry)
    }

    /// Validates and registers one tool.
    ///
    /// # Errors
    ///
    /// Returns [`RegistryError`] when the definition is invalid or duplicated,
    /// or its JSON schema cannot be compiled.
    pub fn register(&mut self, tool: Arc<dyn Tool>) -> Result<(), RegistryError> {
        let definition = tool.definition();
        if !valid_name(&definition.name)
            || definition.description.trim().is_empty()
            || !definition.parameters.is_object()
            || definition.kind == ToolKind::Unknown
        {
            return Err(RegistryError::InvalidDefinition(definition.name));
        }
        if self.tools.contains_key(&definition.name) {
            return Err(RegistryError::Duplicate(definition.name));
        }

        let validator = jsonschema::validator_for(&definition.parameters).map_err(|error| {
            RegistryError::InvalidSchema {
                name: definition.name.clone(),
                source: Box::new(error),
            }
        })?;
        self.tools
            .insert(definition.name, RegisteredTool { tool, validator });
        Ok(())
    }

    #[must_use]
    pub fn lookup(&self, name: &str) -> Option<Arc<dyn Tool>> {
        self.tools
            .get(name)
            .map(|registered| Arc::clone(&registered.tool))
    }

    /// Validates arguments against a registered tool's JSON schema.
    ///
    /// # Errors
    ///
    /// Returns [`RegistryError`] if the tool is unknown or the arguments do not
    /// satisfy its schema.
    pub fn validate(&self, name: &str, arguments: &Value) -> Result<(), RegistryError> {
        let registered = self
            .tools
            .get(name)
            .ok_or_else(|| RegistryError::UnknownTool(name.to_owned()))?;
        registered
            .validator
            .validate(arguments)
            .map_err(|error| RegistryError::InvalidArguments {
                name: name.to_owned(),
                message: error.to_string(),
            })
    }

    #[must_use]
    pub fn definitions(&self) -> Vec<ToolDefinition> {
        let mut definitions = self
            .tools
            .values()
            .map(|registered| registered.tool.definition())
            .collect::<Vec<_>>();
        definitions.sort_unstable_by(|left, right| left.name.cmp(&right.name));
        definitions
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.tools.is_empty()
    }
}

fn valid_name(name: &str) -> bool {
    !name.is_empty()
        && name.len() <= 64
        && name
            .bytes()
            .all(|byte| byte.is_ascii_alphanumeric() || matches!(byte, b'_' | b'-'))
}

/// Encodes a tool result while enforcing the runtime's byte budget.
///
/// The reported size is the full, pre-truncation result size. A bounded UTF-8
/// preview is returned when it fits; otherwise the data is omitted entirely.
///
/// # Panics
///
/// Panics only if serializing a [`ToolResult`] backed by [`serde_json::Value`]
/// fails, which would violate `serde_json`'s in-memory value invariant.
#[must_use]
pub fn encode_result(mut result: ToolResult, max_bytes: usize) -> (Vec<u8>, ToolResult) {
    let initial = serde_json::to_vec(&result).unwrap_or_else(|_| {
        serde_json::to_vec(&ToolResult::failure(
            "result_encoding",
            "tool result could not be encoded",
            false,
        ))
        .expect("the static result-encoding failure is serializable")
    });
    result.size_bytes = initial.len();

    let encoded = serde_json::to_vec(&result).expect("ToolResult is always serializable");
    if initial.len() <= max_bytes {
        return (encoded, result);
    }

    let preview = utf8_prefix(&initial, max_bytes / 3);
    result.data = Some(json!({ "preview": preview }));
    result.truncated = true;
    "tool result exceeded the runtime size limit; preview returned".clone_into(&mut result.summary);
    let mut encoded = serde_json::to_vec(&result).expect("ToolResult is always serializable");
    if encoded.len() > max_bytes {
        result.data = None;
        encoded = serde_json::to_vec(&result).expect("ToolResult is always serializable");
    }
    (encoded, result)
}

fn utf8_prefix(bytes: &[u8], max_bytes: usize) -> &str {
    let text = std::str::from_utf8(bytes).expect("serde_json emits UTF-8");
    if text.len() <= max_bytes {
        return text;
    }
    let mut boundary = max_bytes;
    while boundary > 0 && !text.is_char_boundary(boundary) {
        boundary -= 1;
    }
    &text[..boundary]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn truncation_preserves_utf8_and_original_size() {
        let result = ToolResult::success(json!({ "text": "界".repeat(300) }), "large");
        let full_size = serde_json::to_vec(&result).unwrap().len();
        let (encoded, final_result) = encode_result(result, 256);

        assert!(encoded.len() <= 256);
        assert!(final_result.truncated);
        assert_eq!(final_result.size_bytes, full_size);
        assert!(serde_json::from_slice::<Value>(&encoded).is_ok());
    }

    #[test]
    fn tool_names_are_strict() {
        assert!(valid_name("document_search-2"));
        assert!(!valid_name(""));
        assert!(!valid_name("has spaces"));
        assert!(!valid_name(&"a".repeat(65)));
    }
}
