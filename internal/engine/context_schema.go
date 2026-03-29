package engine

import (
	"encoding/json"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// ActivationContext represents the current cognitive context for memory retrieval.
// Used to modulate recall based on task, goals, emotional state, and environment.
type ActivationContext struct {
	// Task context
	Task      string   `json:"task,omitempty"`        // Current task description
	Goals     []string `json:"goals,omitempty"`       // Active goals
	SubGoalOf *string  `json:"sub_goal_of,omitempty"` // Parent goal reference

	// Cognitive state
	CognitiveLoad float64 `json:"cognitive_load,omitempty"` // 0-1: cognitive load
	TimePressure  float64 `json:"time_pressure,omitempty"`  // 0-1: time pressure

	// Emotional state (optional)
	EmotionalState map[string]float64 `json:"emotional_state,omitempty"` // {"anxiety": 0.7}

	// Spatial context (NEW)
	CurrentLocation string `json:"current_location,omitempty"` // e.g., "office", "production_env"

	// Social context (NEW)
	Collaborators []string `json:"collaborators,omitempty"` // e.g., ["alice", "dev-team"]

	// Environment
	Environment map[string]string `json:"environment,omitempty"` // {"location": "office"}

	// Temporal context
	TimeOfDay    time.Time `json:"time_of_day,omitempty"`
	SessionStart time.Time `json:"session_start,omitempty"`

	// History
	RecentActivations []storage.ULID `json:"recent_activations,omitempty"`
}

// NewActivationContext creates a new activation context with defaults.
func NewActivationContext() *ActivationContext {
	return &ActivationContext{
		CognitiveLoad:     0.5,
		TimePressure:      0.5,
		EmotionalState:    make(map[string]float64),
		Environment:       make(map[string]string),
		Collaborators:     make([]string, 0),
		RecentActivations: make([]storage.ULID, 0),
	}
}

// Marshal serializes the context to JSON.
func (ctx *ActivationContext) Marshal() ([]byte, error) {
	return json.Marshal(ctx)
}

// Unmarshal deserializes the context from JSON.
func (ctx *ActivationContext) Unmarshal(data []byte) error {
	return json.Unmarshal(data, ctx)
}

// WithTask sets the task description.
func (ctx *ActivationContext) WithTask(task string) *ActivationContext {
	ctx.Task = task
	return ctx
}

// WithGoals sets the active goals.
func (ctx *ActivationContext) WithGoals(goals ...string) *ActivationContext {
	ctx.Goals = goals
	return ctx
}

// WithCognitiveLoad sets the cognitive load (0-1).
func (ctx *ActivationContext) WithCognitiveLoad(load float64) *ActivationContext {
	if load < 0 {
		load = 0
	}
	if load > 1 {
		load = 1
	}
	ctx.CognitiveLoad = load
	return ctx
}

// WithTimePressure sets the time pressure (0-1).
func (ctx *ActivationContext) WithTimePressure(pressure float64) *ActivationContext {
	if pressure < 0 {
		pressure = 0
	}
	if pressure > 1 {
		pressure = 1
	}
	ctx.TimePressure = pressure
	return ctx
}

// WithEmotion adds an emotional state.
func (ctx *ActivationContext) WithEmotion(emotion string, intensity float64) *ActivationContext {
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1 {
		intensity = 1
	}
	ctx.EmotionalState[emotion] = intensity
	return ctx
}

// WithEnvironment sets an environment variable.
func (ctx *ActivationContext) WithEnvironment(key, value string) *ActivationContext {
	ctx.Environment[key] = value
	return ctx
}

// WithRecentActivations sets the recently activated engrams.
func (ctx *ActivationContext) WithRecentActivations(ids []storage.ULID) *ActivationContext {
	ctx.RecentActivations = ids
	return ctx
}

// WithCurrentLocation sets the current physical/virtual location.
func (ctx *ActivationContext) WithCurrentLocation(location string) *ActivationContext {
	ctx.CurrentLocation = location
	return ctx
}

// WithCollaborators sets the list of current collaborators.
func (ctx *ActivationContext) WithCollaborators(collabs []string) *ActivationContext {
	ctx.Collaborators = collabs
	return ctx
}

// AddCollaborator adds a single collaborator to the context.
func (ctx *ActivationContext) AddCollaborator(name string) *ActivationContext {
	for _, c := range ctx.Collaborators {
		if c == name {
			return ctx // Already exists
		}
	}
	ctx.Collaborators = append(ctx.Collaborators, name)
	return ctx
}

// IsHighLoad returns true if cognitive load is high (>0.7).
func (ctx *ActivationContext) IsHighLoad() bool {
	return ctx.CognitiveLoad > 0.7
}

// IsHighPressure returns true if time pressure is high (>0.7).
func (ctx *ActivationContext) IsHighPressure() bool {
	return ctx.TimePressure > 0.7
}

// Clone creates a deep copy of the context.
func (ctx *ActivationContext) Clone() *ActivationContext {
	clone := &ActivationContext{
		Task:              ctx.Task,
		Goals:             make([]string, len(ctx.Goals)),
		CognitiveLoad:     ctx.CognitiveLoad,
		TimePressure:      ctx.TimePressure,
		EmotionalState:    make(map[string]float64),
		CurrentLocation:   ctx.CurrentLocation,
		Collaborators:     make([]string, len(ctx.Collaborators)),
		Environment:       make(map[string]string),
		TimeOfDay:         ctx.TimeOfDay,
		SessionStart:      ctx.SessionStart,
		RecentActivations: make([]storage.ULID, len(ctx.RecentActivations)),
	}

	copy(clone.Goals, ctx.Goals)
	copy(clone.Collaborators, ctx.Collaborators)
	copy(clone.RecentActivations, ctx.RecentActivations)

	for k, v := range ctx.EmotionalState {
		clone.EmotionalState[k] = v
	}
	for k, v := range ctx.Environment {
		clone.Environment[k] = v
	}

	if ctx.SubGoalOf != nil {
		subGoal := *ctx.SubGoalOf
		clone.SubGoalOf = &subGoal
	}

	return clone
}
