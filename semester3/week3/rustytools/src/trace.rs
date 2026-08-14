use std::time::Duration;

use chrono::{SecondsFormat, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;

#[derive(Clone, Copy, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum EventType {
    RoundStarted,
    ToolCalled,
    ToolFinished,
    Stopped,
}

/// A deliberately safe public event. It contains no prompts, hidden reasoning,
/// raw tool output, system instructions, or unsanitized errors.
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct TraceEvent {
    pub time: String,
    #[serde(default, skip_serializing_if = "is_zero")]
    pub round: usize,
    #[serde(rename = "type")]
    pub event_type: EventType,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub arguments_summary: Option<Value>,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub status: String,
    #[serde(
        default,
        skip_serializing_if = "duration_is_zero",
        with = "duration_nanos"
    )]
    pub duration: Duration,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub summary: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub stop_reason: String,
}

impl TraceEvent {
    #[must_use]
    pub fn new(round: usize, event_type: EventType) -> Self {
        Self {
            time: String::new(),
            round,
            event_type,
            tool: String::new(),
            arguments_summary: None,
            status: String::new(),
            duration: Duration::ZERO,
            summary: String::new(),
            stop_reason: String::new(),
        }
    }

    #[must_use]
    pub fn duration(mut self, duration: Duration) -> Self {
        self.duration = duration;
        self
    }
}

#[derive(Default)]
pub struct TraceRecorder {
    events: Vec<TraceEvent>,
}

impl TraceRecorder {
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    pub fn add(&mut self, mut event: TraceEvent) {
        event.time = Utc::now().to_rfc3339_opts(SecondsFormat::Nanos, true);
        self.events.push(event);
    }

    #[must_use]
    pub fn events(&self) -> Vec<TraceEvent> {
        self.events.clone()
    }

    #[must_use]
    pub fn into_events(self) -> Vec<TraceEvent> {
        self.events
    }
}

// Serde requires skip predicates to accept a reference, even for `Copy` types.
#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero(value: &usize) -> bool {
    *value == 0
}

fn duration_is_zero(value: &Duration) -> bool {
    value.is_zero()
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
    fn trace_serialization_has_no_private_fields() {
        let mut recorder = TraceRecorder::new();
        recorder.add(TraceEvent::new(1, EventType::RoundStarted));
        let value = serde_json::to_value(recorder.events()).unwrap();

        assert_eq!(value[0]["type"], "round_started");
        assert!(value[0]["time"].as_str().unwrap().ends_with('Z'));
        assert!(value[0].get("prompt").is_none());
        assert!(value[0].get("reasoning").is_none());
        assert!(value[0].get("system_prompt").is_none());
    }
}
