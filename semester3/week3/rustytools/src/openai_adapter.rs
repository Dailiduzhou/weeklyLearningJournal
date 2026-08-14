//! `OpenAI` Responses API implementation of the provider-neutral LLM client.

use async_trait::async_trait;
use openai_oxide::{
    ClientConfig, OpenAI, OpenAIError,
    types::responses::{
        EasyInputContent, EasyInputMessage, FunctionCallOutput, FunctionCallOutputItemParam,
        FunctionToolCall, InputItem, Item, MessageType, Response as OpenAiResponse,
        ResponseCreateRequest, ResponseInput, ResponseTool, Role as OpenAiRole,
    },
};
use tracing::{debug, warn};

use crate::llm::{
    Client, ErrorKind, LlmError, Message, Request, Response, Role, ToolCall, ToolSpec,
};

/// A model client backed by `openai-oxide`'s Responses API.
#[derive(Debug, Clone)]
pub struct OpenAiClient {
    client: OpenAI,
    model: String,
}

impl OpenAiClient {
    /// Construct a client. An empty `base_url` selects `OpenAI`'s default API URL.
    ///
    /// SDK retries are disabled because the runtime owns the retry policy.
    pub fn new(
        api_key: impl Into<String>,
        base_url: impl Into<String>,
        model: impl Into<String>,
    ) -> Self {
        let base_url = base_url.into();
        let mut config = ClientConfig::new(api_key).max_retries(0);
        if !base_url.is_empty() {
            // openai-oxide concatenates endpoint paths directly, so normalize
            // the same user-friendly trailing slash accepted by official SDKs.
            config = config.base_url(base_url.trim_end_matches('/'));
        }
        Self {
            client: OpenAI::with_config(config),
            model: model.into(),
        }
    }

    /// Construct from an existing SDK client, primarily for custom transports.
    pub fn with_client(client: OpenAI, model: impl Into<String>) -> Self {
        Self {
            client,
            model: model.into(),
        }
    }

    fn create_request(&self, request: &Request) -> Result<ResponseCreateRequest, LlmError> {
        let mut params = ResponseCreateRequest::new(self.model.clone());
        params.input = Some(convert_messages(&request.messages)?);
        params.tools = Some(request.tools.iter().map(convert_tool).collect());
        params.parallel_tool_calls = Some(false);
        Ok(params)
    }
}

#[async_trait]
impl Client for OpenAiClient {
    async fn complete(&self, request: Request) -> Result<Response, LlmError> {
        let params = self.create_request(&request)?;

        debug!(
            model = %self.model,
            messages = request.messages.len(),
            tools = request.tools.len(),
            "requesting OpenAI response"
        );
        let response = self
            .client
            .responses()
            .create(params)
            .await
            .map_err(classify)?;

        convert_response(&response)
    }
}

fn convert_tool(spec: &ToolSpec) -> ResponseTool {
    ResponseTool::Function {
        name: spec.name.clone(),
        description: Some(spec.description.clone()),
        parameters: Some(spec.schema.clone()),
        strict: Some(false),
    }
}

fn convert_messages(messages: &[Message]) -> Result<ResponseInput, LlmError> {
    let mut items = Vec::with_capacity(messages.len());
    for message in messages {
        match message.role {
            Role::System | Role::User => {
                items.push(serialize_input(easy_message(message))?);
            }
            Role::Tool => {
                let item = InputItem::Item(Item::FunctionCallOutput(FunctionCallOutputItemParam {
                    call_id: message.tool_call_id.clone(),
                    output: FunctionCallOutput::Text(message.content.clone()),
                    id: None,
                    status: None,
                }));
                items.push(serialize_input(item)?);
            }
            Role::Assistant => {
                if !message.content.is_empty() || message.tool_calls.is_empty() {
                    items.push(serialize_input(easy_message(message))?);
                }
                for call in &message.tool_calls {
                    let item = InputItem::Item(Item::FunctionCall(FunctionToolCall {
                        arguments: call.arguments.clone(),
                        call_id: call.id.clone(),
                        name: call.name.clone(),
                        id: None,
                        status: None,
                    }));
                    items.push(serialize_input(item)?);
                }
            }
        }
    }
    Ok(ResponseInput::Items(items))
}

fn easy_message(message: &Message) -> InputItem {
    InputItem::EasyMessage(EasyInputMessage {
        r#type: MessageType::Message,
        role: match message.role {
            Role::System => OpenAiRole::System,
            Role::User => OpenAiRole::User,
            Role::Assistant => OpenAiRole::Assistant,
            Role::Tool => unreachable!("tool messages use function_call_output"),
        },
        content: EasyInputContent::Text(message.content.clone()),
    })
}

fn serialize_input(item: InputItem) -> Result<serde_json::Value, LlmError> {
    serde_json::to_value(item)
        .map_err(|error| LlmError::new(ErrorKind::InvalidResponse, false, error))
}

fn convert_response(response: &OpenAiResponse) -> Result<Response, LlmError> {
    if let Some(status) = response.status.as_deref()
        && status != "completed"
    {
        let detail = response
            .error
            .as_ref()
            .map(|error| error.message.as_str())
            .or_else(|| {
                response
                    .incomplete_details
                    .as_ref()
                    .and_then(|details| details.reason.as_deref())
            });
        let message = match detail {
            Some(detail) => format!("response was not completed: {status}: {detail}"),
            None => format!("response was not completed: {status}"),
        };
        return Err(LlmError::invalid_response(message));
    }

    let mut result = Message::assistant(response.output_text());
    for item in &response.output {
        match item.type_.as_str() {
            "message" => {
                if item
                    .content
                    .as_deref()
                    .unwrap_or_default()
                    .iter()
                    .any(|content| content.type_ == "refusal")
                {
                    warn!(response_id = %response.id, "model refused the request");
                    return Err(LlmError::message(
                        ErrorKind::Permanent,
                        false,
                        "model refused the request",
                    ));
                }
            }
            "function_call" => result.tool_calls.push(ToolCall::new(
                item.call_id.clone().unwrap_or_default(),
                item.name.clone().unwrap_or_default(),
                item.arguments.clone().unwrap_or_default(),
            )),
            "reasoning" => {}
            unsupported => {
                return Err(LlmError::invalid_response(format!(
                    "unsupported response output type {unsupported:?}"
                )));
            }
        }
    }

    debug!(
        response_id = %response.id,
        tool_calls = result.tool_calls.len(),
        "OpenAI response completed"
    );
    Ok(Response { message: result })
}

fn classify(error: OpenAIError) -> LlmError {
    match error {
        error @ OpenAIError::ApiError { status, .. } => {
            let retryable = matches!(status, 408 | 409 | 429) || status >= 500;
            let kind = if retryable {
                ErrorKind::Temporary
            } else {
                ErrorKind::Permanent
            };
            LlmError::new(kind, retryable, error)
        }
        OpenAIError::RequestError(source) => {
            // Only failures incurred while executing an HTTP request correspond
            // to Go's net.Error. Invalid URL/build and response-decode failures
            // are deterministic and must not consume the runtime's retry budget.
            let retryable = source.is_timeout() || source.is_connect() || source.is_request();
            let kind = if retryable {
                ErrorKind::Temporary
            } else {
                ErrorKind::Permanent
            };
            LlmError::new(kind, retryable, OpenAIError::RequestError(source))
        }
        error => LlmError::new(ErrorKind::Permanent, false, error),
    }
}

#[cfg(test)]
mod tests {
    use serde_json::json;

    use super::*;

    #[test]
    fn converts_responses_history_without_losing_raw_arguments() {
        let input = convert_messages(&[
            Message::system("be concise"),
            Message::user("calculate"),
            Message {
                role: Role::Assistant,
                content: "working".into(),
                tool_call_id: String::new(),
                tool_calls: vec![ToolCall::new(
                    "call-1",
                    "calculator",
                    r#"{"expression":"1+1"}"#,
                )],
            },
            Message::tool_result("call-1", r#"{"result":2}"#),
        ])
        .expect("convert messages");

        let encoded = serde_json::to_value(input).expect("serialize input");
        let items = encoded.as_array().expect("items input");
        assert_eq!(items[0]["role"], "system");
        assert_eq!(items[1]["role"], "user");
        assert_eq!(items[2]["role"], "assistant");
        assert_eq!(items[3]["type"], "function_call");
        assert_eq!(items[3]["call_id"], "call-1");
        assert_eq!(items[3]["arguments"], r#"{"expression":"1+1"}"#);
        assert_eq!(items[4]["type"], "function_call_output");
        assert_eq!(items[4]["output"], r#"{"result":2}"#);
    }

    #[test]
    fn assistant_with_only_calls_does_not_emit_empty_message() {
        let input = convert_messages(&[Message {
            role: Role::Assistant,
            content: String::new(),
            tool_call_id: String::new(),
            tool_calls: vec![ToolCall::new("call-1", "calculator", "not-json")],
        }])
        .expect("convert messages");
        let encoded = serde_json::to_value(input).expect("serialize input");
        let items = encoded.as_array().expect("items input");
        assert_eq!(items.len(), 1);
        assert_eq!(items[0]["arguments"], "not-json");
    }

    #[test]
    fn builds_expected_responses_request_shape() {
        let client = OpenAiClient::new("test-key", "", "test-model");
        let params = client
            .create_request(&Request {
                messages: vec![Message::user("calculate")],
                tools: vec![ToolSpec::new(
                    "calculator",
                    "calculate",
                    json!({
                        "type": "object",
                        "properties": {"expression": {"type": "string"}}
                    }),
                )],
            })
            .expect("build request");
        let body = serde_json::to_value(params).expect("serialize request");

        assert_eq!(body["model"], "test-model");
        assert_eq!(body["parallel_tool_calls"], false);
        assert_eq!(body["input"][0]["type"], "message");
        assert_eq!(body["input"][0]["role"], "user");
        assert_eq!(body["input"][0]["content"], "calculate");
        assert_eq!(body["tools"][0]["type"], "function");
        assert_eq!(body["tools"][0]["name"], "calculator");
        assert_eq!(body["tools"][0]["description"], "calculate");
        assert_eq!(body["tools"][0]["strict"], false);
        assert_eq!(
            body["tools"][0]["parameters"]["properties"]["expression"]["type"],
            "string"
        );
    }

    #[tokio::test]
    async fn url_builder_errors_are_permanent() {
        let client = OpenAiClient::new("test-key", "://not-a-url", "test-model");
        let error = client
            .complete(Request::default())
            .await
            .expect_err("invalid URL must fail before network I/O");

        assert_eq!(error.kind, ErrorKind::Permanent);
        assert!(!error.retryable);
    }

    #[test]
    fn converts_text_and_function_call_response() {
        let response: OpenAiResponse = serde_json::from_value(json!({
            "id": "resp-test",
            "object": "response",
            "created_at": 1.0,
            "model": "test-model",
            "status": "completed",
            "output": [
                {
                    "id": "msg-1",
                    "type": "message",
                    "role": "assistant",
                    "status": "completed",
                    "content": [{
                        "type": "output_text",
                        "text": "I'll calculate.",
                        "annotations": []
                    }]
                },
                {
                    "id": "fc-1",
                    "type": "function_call",
                    "call_id": "call-1",
                    "name": "calculator",
                    "arguments": r#"{"expression":"1+1"}"#,
                    "status": "completed"
                }
            ]
        }))
        .expect("parse SDK response");

        let response = convert_response(&response).expect("convert response");
        assert_eq!(response.message.content, "I'll calculate.");
        assert_eq!(
            response.message.tool_calls,
            vec![ToolCall::new(
                "call-1",
                "calculator",
                r#"{"expression":"1+1"}"#
            )]
        );
    }

    #[test]
    fn incomplete_and_refusal_responses_are_rejected() {
        let incomplete: OpenAiResponse = serde_json::from_value(json!({
            "id": "resp-incomplete",
            "object": "response",
            "created_at": 1.0,
            "model": "test-model",
            "status": "incomplete",
            "incomplete_details": {"reason": "max_output_tokens"},
            "output": []
        }))
        .expect("parse incomplete response");
        let error = convert_response(&incomplete).expect_err("reject incomplete response");
        assert_eq!(error.kind, ErrorKind::InvalidResponse);
        assert!(error.to_string().contains("max_output_tokens"));

        let refusal: OpenAiResponse = serde_json::from_value(json!({
            "id": "resp-refusal",
            "object": "response",
            "created_at": 1.0,
            "model": "test-model",
            "status": "completed",
            "output": [{
                "id": "msg-1",
                "type": "message",
                "role": "assistant",
                "status": "completed",
                "content": [{"type": "refusal"}]
            }]
        }))
        .expect("parse refusal response");
        let error = convert_response(&refusal).expect_err("reject refusal");
        assert_eq!(error.kind, ErrorKind::Permanent);
    }

    #[test]
    fn classifies_retryable_http_statuses() {
        let error = classify(OpenAIError::ApiError {
            status: 429,
            message: "slow down".into(),
            type_: None,
            code: None,
            request_id: None,
        });
        assert_eq!(error.kind, ErrorKind::Temporary);
        assert!(error.retryable);

        let error = classify(OpenAIError::ApiError {
            status: 400,
            message: "bad request".into(),
            type_: None,
            code: None,
            request_id: None,
        });
        assert_eq!(error.kind, ErrorKind::Permanent);
        assert!(!error.retryable);
    }
}
