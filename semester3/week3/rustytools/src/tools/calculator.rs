use async_trait::async_trait;
use serde::Deserialize;
use serde_json::{Value, json};
use thiserror::Error;

use crate::tool::{Tool, ToolDefinition, ToolKind, ToolResult};

const MAX_EXPRESSION_LENGTH: usize = 256;
const MAX_PARSE_DEPTH: usize = 32;
const MAX_ABS_VALUE: f64 = 1e15;

#[derive(Debug, Default)]
pub struct Calculator;

impl Calculator {
    #[must_use]
    pub const fn new() -> Self {
        Self
    }
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct Arguments {
    expression: String,
}

#[async_trait]
impl Tool for Calculator {
    fn definition(&self) -> ToolDefinition {
        ToolDefinition {
            name: "calculator".into(),
            description: "Evaluate a safe arithmetic expression. Supports numbers, parentheses, +, -, *, / and ^ only.".into(),
            kind: ToolKind::Read,
            parameters: json!({
                "type": "object",
                "properties": {
                    "expression": {
                        "type": "string",
                        "minLength": 1,
                        "maxLength": MAX_EXPRESSION_LENGTH
                    }
                },
                "required": ["expression"],
                "additionalProperties": false
            }),
        }
    }

    async fn execute(&self, arguments: Value) -> ToolResult {
        // Make this bounded CPU-only operation cooperatively cancelable when
        // its runtime deadline has already elapsed.
        tokio::task::yield_now().await;
        let arguments: Arguments = match serde_json::from_value(arguments) {
            Ok(arguments) => arguments,
            Err(error) => {
                return ToolResult::failure("invalid_arguments", error.to_string(), false);
            }
        };

        match Parser::new(&arguments.expression).parse() {
            Ok(value) => ToolResult::success(json!({ "value": value }), format_number(value)),
            Err(error) => ToolResult::failure("invalid_expression", error.to_string(), false),
        }
    }
}

#[derive(Debug, Error, PartialEq)]
enum ParseError {
    #[error("expression length must be between 1 and {MAX_EXPRESSION_LENGTH}")]
    Length,
    #[error("unexpected character {character:?} at offset {offset}")]
    Unexpected { character: char, offset: usize },
    #[error("expected number at offset {0}")]
    ExpectedNumber(usize),
    #[error("invalid number")]
    InvalidNumber,
    #[error("division by zero")]
    DivisionByZero,
    #[error("missing closing parenthesis")]
    MissingClosingParenthesis,
    #[error("expression nesting exceeds {MAX_PARSE_DEPTH}")]
    TooDeep,
    #[error("numeric result is outside the allowed range")]
    OutOfRange,
}

struct Parser<'a> {
    input: &'a str,
    position: usize,
    depth: usize,
}

impl<'a> Parser<'a> {
    fn new(input: &'a str) -> Self {
        Self {
            input,
            position: 0,
            depth: 0,
        }
    }

    fn parse(mut self) -> Result<f64, ParseError> {
        if self.input.is_empty() || self.input.len() > MAX_EXPRESSION_LENGTH {
            return Err(ParseError::Length);
        }
        let value = self.expression()?;
        self.skip_whitespace();
        if self.position != self.input.len() {
            let character = self
                .remaining()
                .chars()
                .next()
                .expect("position is in range");
            return Err(ParseError::Unexpected {
                character,
                offset: self.position,
            });
        }
        checked(value)
    }

    fn expression(&mut self) -> Result<f64, ParseError> {
        let mut left = self.term()?;
        loop {
            match self.take(&['+', '-']) {
                Some('+') => left += self.term()?,
                Some('-') => left -= self.term()?,
                _ => return checked(left),
            }
            left = checked(left)?;
        }
    }

    fn term(&mut self) -> Result<f64, ParseError> {
        let mut left = self.power()?;
        loop {
            match self.take(&['*', '/']) {
                Some('*') => left *= self.power()?,
                Some('/') => {
                    let right = self.power()?;
                    if right == 0.0 {
                        return Err(ParseError::DivisionByZero);
                    }
                    left /= right;
                }
                _ => return checked(left),
            }
            left = checked(left)?;
        }
    }

    // Recursive parsing makes exponentiation right-associative: 2^3^2 = 512.
    fn power(&mut self) -> Result<f64, ParseError> {
        let left = self.unary()?;
        if self.take(&['^']).is_some() {
            return checked(left.powf(self.power()?));
        }
        Ok(left)
    }

    fn unary(&mut self) -> Result<f64, ParseError> {
        match self.take(&['+', '-']) {
            Some('+') => self.unary(),
            Some('-') => checked(-self.unary()?),
            _ => self.primary(),
        }
    }

    fn primary(&mut self) -> Result<f64, ParseError> {
        self.skip_whitespace();
        if self.remaining().starts_with('(') {
            self.position += 1;
            self.depth += 1;
            if self.depth > MAX_PARSE_DEPTH {
                return Err(ParseError::TooDeep);
            }
            let value = self.expression();
            self.depth -= 1;
            let value = value?;
            self.skip_whitespace();
            if !self.remaining().starts_with(')') {
                return Err(ParseError::MissingClosingParenthesis);
            }
            self.position += 1;
            return Ok(value);
        }

        let start = self.position;
        let mut seen_digit = false;
        let mut seen_dot = false;
        for byte in self.input.as_bytes().iter().copied().skip(self.position) {
            if byte.is_ascii_digit() {
                seen_digit = true;
                self.position += 1;
            } else if byte == b'.' && !seen_dot {
                seen_dot = true;
                self.position += 1;
            } else {
                break;
            }
        }
        if !seen_digit {
            return Err(ParseError::ExpectedNumber(start));
        }
        let value = self.input[start..self.position]
            .parse()
            .map_err(|_| ParseError::InvalidNumber)?;
        checked(value)
    }

    fn take(&mut self, expected: &[char]) -> Option<char> {
        self.skip_whitespace();
        let character = self.remaining().chars().next()?;
        if expected.contains(&character) {
            self.position += character.len_utf8();
            Some(character)
        } else {
            None
        }
    }

    fn skip_whitespace(&mut self) {
        while let Some(character) = self.remaining().chars().next() {
            if !character.is_whitespace() {
                break;
            }
            self.position += character.len_utf8();
        }
    }

    fn remaining(&self) -> &'a str {
        &self.input[self.position..]
    }
}

fn checked(value: f64) -> Result<f64, ParseError> {
    if !value.is_finite() || value.abs() > MAX_ABS_VALUE {
        Err(ParseError::OutOfRange)
    } else {
        Ok(value)
    }
}

fn format_number(value: f64) -> String {
    if value == 0.0 {
        return "0".into();
    }
    value.to_string()
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use super::*;
    use crate::tool::ToolRegistry;

    #[test]
    fn respects_precedence_and_right_associative_power() {
        assert_eq!(Parser::new("2 + 3 * 4").parse(), Ok(14.0));
        assert_eq!(Parser::new("2^3^2").parse(), Ok(512.0));
        assert_eq!(Parser::new("-(2 + 3)").parse(), Ok(-5.0));
    }

    #[test]
    fn rejects_unsafe_or_invalid_results() {
        assert_eq!(
            Parser::new("1 / 0").parse(),
            Err(ParseError::DivisionByZero)
        );
        assert_eq!(
            Parser::new("2e3").parse(),
            Err(ParseError::Unexpected {
                character: 'e',
                offset: 1
            })
        );
        assert_eq!(
            Parser::new("1e16").parse(),
            Err(ParseError::Unexpected {
                character: 'e',
                offset: 1
            })
        );
    }

    #[tokio::test]
    async fn works_through_registry_validation_and_dispatch() {
        let registry = ToolRegistry::new([Arc::new(Calculator::new()) as Arc<dyn Tool>]).unwrap();
        let arguments = json!({ "expression": "(6 + 2) / 4" });
        registry.validate("calculator", &arguments).unwrap();
        assert!(
            registry
                .validate(
                    "calculator",
                    &json!({ "expression": "1", "sql": "select 1" })
                )
                .is_err()
        );

        let result = registry
            .lookup("calculator")
            .unwrap()
            .execute(arguments)
            .await;
        assert!(result.ok);
        assert_eq!(result.data.unwrap()["value"], 2.0);
    }
}
