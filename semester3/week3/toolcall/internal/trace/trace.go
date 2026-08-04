package trace

import "time"

type EventType string

const (
	EventRoundStarted EventType = "round_started"
	EventToolCalled   EventType = "tool_called"
	EventToolFinished EventType = "tool_finished"
	EventStopped      EventType = "stopped"
)

type Event struct {
	Time       time.Time     `json:"time"`
	Round      int           `json:"round,omitempty"`
	Type       EventType     `json:"type"`
	Tool       string        `json:"tool,omitempty"`
	Arguments  any           `json:"arguments_summary,omitempty"`
	Status     string        `json:"status,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
	Summary    string        `json:"summary,omitempty"`
	StopReason string        `json:"stop_reason,omitempty"`
}

type Recorder struct {
	events []Event
	now    func() time.Time
}

func NewRecorder() *Recorder { return &Recorder{now: time.Now} }

func (r *Recorder) Add(event Event) {
	event.Time = r.now().UTC()
	r.events = append(r.events, event)
}

func (r *Recorder) Events() []Event {
	result := make([]Event, len(r.events))
	copy(result, r.events)
	return result
}
