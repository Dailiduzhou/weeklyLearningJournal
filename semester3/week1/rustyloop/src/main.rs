use std::env;
use std::future::Future;
use std::io::{self, BufRead, Write};
use std::process::ExitCode;

use openai_oxide::types::chat::{ChatCompletionMessageParam, ChatCompletionRequest, UserContent};
use openai_oxide::{ClientConfig, OpenAI, OpenAIError};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use thiserror::Error;

const MAX_RETRIES: u32 = 3;
const MODEL: &str = "deepseek-v4-flash";

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Response {
    title: String,
    summary: String,
    priority: String,
    tags: Vec<String>,
}

#[derive(Debug, Error)]
enum AppError {
    #[error("model output does not match Response")]
    InvalidResponse,
    #[error("OPENAI_API_KEY is required")]
    MissingApiKey,
    #[error("create chat completion: {0}")]
    Request(#[source] OpenAIError),
    #[error(transparent)]
    Io(#[from] io::Error),
    #[error(transparent)]
    Json(#[from] serde_json::Error),
}

#[tokio::main]
async fn main() -> ExitCode {
    if let Err(error) = run().await {
        eprintln!("{error}");
        return ExitCode::FAILURE;
    }
    ExitCode::SUCCESS
}

async fn run() -> Result<(), AppError> {
    let client = OpenAI::with_config(client_config()?);

    interactive_loop(
        io::stdin().lock(),
        io::stdout().lock(),
        io::stderr().lock(),
        move |prompt| {
            let client = client.clone();
            async move { create_response(&client, prompt).await }
        },
    )
    .await
}

async fn interactive_loop<R, W, D, H, F>(
    mut input: R,
    mut output: W,
    mut diagnostics: D,
    mut handle: H,
) -> Result<(), AppError>
where
    R: BufRead,
    W: Write,
    D: Write,
    H: FnMut(String) -> F,
    F: Future<Output = Result<Response, AppError>>,
{
    let mut line = String::new();

    loop {
        write!(diagnostics, "> ")?;
        diagnostics.flush()?;

        line.clear();
        if input.read_line(&mut line)? == 0 {
            break;
        }

        let prompt = line.trim();
        if prompt.is_empty() {
            continue;
        }
        if prompt.eq_ignore_ascii_case("quit") || prompt.eq_ignore_ascii_case("exit") {
            break;
        }

        match handle(prompt.to_owned()).await {
            Ok(response) => {
                if let Err(error) = write_response(&mut output, &response) {
                    writeln!(diagnostics, "{error}")?;
                }
            }
            Err(AppError::InvalidResponse) => {} // 忽略解析失败，继续循环
            Err(error) => {
                writeln!(diagnostics, "{error}")?;
            }
        }
    }

    Ok(())
}

async fn create_response(client: &OpenAI, user_prompt: String) -> Result<Response, AppError> {
    let system_prompt = format!(
        "Return only one JSON object that exactly matches the following JSON Schema. \
         Do not use Markdown or add explanatory text.\n\
         JSON Schema:\n{}",
        response_schema()
    );

    let request = ChatCompletionRequest::new(
        MODEL,
        vec![
            ChatCompletionMessageParam::System {
                content: system_prompt,
                name: None,
            },
            ChatCompletionMessageParam::User {
                content: UserContent::Text(user_prompt),
                name: None,
            },
        ],
    );

    let completion = client
        .chat()
        .completions()
        .create(request)
        .await
        .map_err(AppError::Request)?;

    let message = completion
        .choices
        .first()
        .map(|choice| &choice.message)
        .ok_or(AppError::InvalidResponse)?;

    if message.refusal.as_deref().is_some_and(|r| !r.is_empty()) {
        return Err(AppError::InvalidResponse);
    }

    let content = message
        .content
        .as_deref()
        .ok_or(AppError::InvalidResponse)?;

    decode_response(content).ok_or(AppError::InvalidResponse)
}

fn client_config() -> Result<ClientConfig, AppError> {
    client_config_with(|name| env::var(name).ok())
}

fn client_config_with(
    mut get_env: impl FnMut(&str) -> Option<String>,
) -> Result<ClientConfig, AppError> {
    let api_key = ["OPENAI_API_KEY", "OPENAI_APIKEY"]
        .into_iter()
        .find_map(|key| get_env(key).filter(|value| !value.is_empty()))
        .ok_or(AppError::MissingApiKey)?;

    let mut config = ClientConfig::new(api_key).max_retries(MAX_RETRIES);

    // Legacy BASEURL 具有更高优先级
    if let Some(base_url) = ["OPENAI_BASEURL", "OPENAI_BASE_URL"]
        .into_iter()
        .find_map(|key| get_env(key).filter(|value| !value.is_empty()))
    {
        config = config.base_url(base_url.trim_end_matches('/'));
    }

    Ok(config)
}

fn response_schema() -> Value {
    json!({
        "type": "object",
        "properties": {
            "title": { "type": "string" },
            "summary": { "type": "string" },
            "priority": { "type": "string" },
            "tags": {
                "type": "array",
                "items": { "type": "string" }
            }
        },
        "required": ["title", "summary", "priority", "tags"],
        "additionalProperties": false
    })
}

fn decode_response(raw: &str) -> Option<Response> {
    serde_json::from_str(raw).ok()
}

fn write_response(mut output: impl Write, response: &Response) -> Result<(), AppError> {
    serde_json::to_writer(&mut output, response)?;
    writeln!(output)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use std::collections::HashMap;
    use std::io::Cursor;

    use mockito::Matcher;
    use serde_json::json;

    use super::*;

    fn response(title: &str) -> Response {
        Response {
            title: title.to_owned(),
            summary: "summary".to_owned(),
            priority: "high".to_owned(),
            tags: vec!["rust".to_owned()],
        }
    }

    #[tokio::test]
    async fn create_response_uses_chat_completions() {
        let mut server = mockito::Server::new_async().await;
        let system_prompt = format!(
            "Return only one JSON object that exactly matches the following JSON Schema. \
             Do not use Markdown or add explanatory text.\n\
             JSON Schema:\n{}",
            response_schema()
        );

        let endpoint = server
            .mock("POST", "/chat/completions")
            .match_header("authorization", "Bearer test-key")
            .match_body(Matcher::Json(json!({
                "model": MODEL,
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": "test prompt"}
                ]
            })))
            .with_status(200)
            .with_header("content-type", "application/json")
            .with_body(
                r#"{
                    "id":"chatcmpl-test",
                    "object":"chat.completion",
                    "created":0,
                    "model":"deepseek-v4-flash",
                    "choices":[{
                        "index":0,
                        "message":{
                            "role":"assistant",
                            "content":"{\"title\":\"Chat Completion\",\"summary\":\"Uses the chat endpoint.\",\"priority\":\"high\",\"tags\":[\"chat\"]}",
                            "refusal":null
                        },
                        "finish_reason":"stop",
                        "logprobs":null
                    }]
                }"#,
            )
            .create_async()
            .await;

        let client = OpenAI::with_config(
            ClientConfig::new("test-key")
                .base_url(server.url())
                .max_retries(0),
        );
        let got = create_response(&client, "test prompt".to_owned())
            .await
            .expect("chat completion should succeed");

        assert_eq!(
            got,
            Response {
                title: "Chat Completion".to_owned(),
                summary: "Uses the chat endpoint.".to_owned(),
                priority: "high".to_owned(),
                tags: vec!["chat".to_owned()],
            }
        );
        endpoint.assert_async().await;
    }

    #[tokio::test]
    async fn interactive_loop_keeps_running_after_response_and_request_errors() {
        let input = Cursor::new(
            "\nfirst prompt\ninvalid response\nrequest error\nlast prompt\nQUIT\nignored prompt\n",
        );
        let mut output = Vec::new();
        let mut diagnostics = Vec::new();
        let mut prompts = Vec::new();

        interactive_loop(input, &mut output, &mut diagnostics, |prompt| {
            prompts.push(prompt.clone());
            std::future::ready(match prompt.as_str() {
                "invalid response" => Err(AppError::InvalidResponse),
                "request error" => Err(AppError::Request(OpenAIError::InvalidArgument(
                    "request failed".to_owned(),
                ))),
                _ => Ok(response(&format!("response for {prompt}"))),
            })
        })
        .await
        .expect("interactive loop should exit cleanly");

        assert_eq!(
            prompts,
            [
                "first prompt",
                "invalid response",
                "request error",
                "last prompt"
            ]
        );
        assert_eq!(
            String::from_utf8(output).unwrap(),
            "{\"title\":\"response for first prompt\",\"summary\":\"summary\",\"priority\":\"high\",\"tags\":[\"rust\"]}\n\
             {\"title\":\"response for last prompt\",\"summary\":\"summary\",\"priority\":\"high\",\"tags\":[\"rust\"]}\n"
        );
        let diagnostics = String::from_utf8(diagnostics).unwrap();
        assert!(!diagnostics.contains("model output does not match Response"));
        assert!(diagnostics.contains("request failed"));
    }

    #[test]
    fn decode_response_is_strict() {
        let want = Response {
            title: "SDK migration".to_owned(),
            summary: "Use structured output.".to_owned(),
            priority: "high".to_owned(),
            tags: vec!["go".to_owned(), "sdk".to_owned()],
        };
        let cases = [
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":["go","sdk"]}"#,
                Some(&want),
            ),
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":"high"}"#,
                None,
            ),
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[],"owner":"team"}"#,
                None,
            ),
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":1,"tags":[]}"#,
                None,
            ),
            (
                r#"{"title":null,"summary":"Use structured output.","priority":"high","tags":[]}"#,
                None,
            ),
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[null]}"#,
                None,
            ),
            (
                r#"{"title":"SDK migration","summary":"Use structured output.","priority":"high","tags":[]} {}"#,
                None,
            ),
        ];

        for (raw, expected) in cases {
            let got = decode_response(raw);
            assert_eq!(got.as_ref(), expected, "raw response: {raw}");
        }
    }

    #[test]
    fn response_schema_is_strict() {
        let schema = response_schema();
        assert_eq!(schema["additionalProperties"], false);
        assert_eq!(
            schema["required"],
            json!(["title", "summary", "priority", "tags"])
        );
    }

    #[test]
    fn client_config_supports_standard_and_legacy_environment_names() {
        let values = HashMap::from([
            ("OPENAI_APIKEY", "legacy-key"),
            ("OPENAI_BASE_URL", "https://standard.example/v1/"),
        ]);
        let config = client_config_with(|name| values.get(name).map(ToString::to_string)).unwrap();

        assert_eq!(config.api_key, "legacy-key");
        assert_eq!(config.base_url, "https://standard.example/v1");
        assert_eq!(config.max_retries, MAX_RETRIES);
    }

    #[test]
    fn legacy_base_url_takes_precedence() {
        let values = HashMap::from([
            ("OPENAI_API_KEY", "standard-key"),
            ("OPENAI_BASEURL", "https://legacy.example/v1"),
            ("OPENAI_BASE_URL", "https://standard.example/v1"),
        ]);
        let config = client_config_with(|name| values.get(name).map(ToString::to_string)).unwrap();

        assert_eq!(config.api_key, "standard-key");
        assert_eq!(config.base_url, "https://legacy.example/v1");
    }
}
