package engine

import (
	"encoding/json"
	"testing"
)

func TestActivationContext_Basic(t *testing.T) {
	ctx := NewActivationContext()

	if ctx.CognitiveLoad != 0.5 {
		t.Errorf("Default cognitive load should be 0.5, got %f", ctx.CognitiveLoad)
	}

	if ctx.TimePressure != 0.5 {
		t.Errorf("Default time pressure should be 0.5, got %f", ctx.TimePressure)
	}
}

func TestActivationContext_Fluent(t *testing.T) {
	ctx := NewActivationContext().
		WithTask("debug payment").
		WithGoals("fix bug", "prevent recurrence").
		WithCognitiveLoad(0.8).
		WithTimePressure(0.6).
		WithEmotion("frustration", 0.7).
		WithEnvironment("location", "office")

	if ctx.Task != "debug payment" {
		t.Errorf("Task not set correctly")
	}

	if len(ctx.Goals) != 2 {
		t.Errorf("Expected 2 goals, got %d", len(ctx.Goals))
	}

	if ctx.CognitiveLoad != 0.8 {
		t.Errorf("Cognitive load not set correctly")
	}

	if ctx.EmotionalState["frustration"] != 0.7 {
		t.Errorf("Emotion not set correctly")
	}
}

func TestActivationContext_Clamping(t *testing.T) {
	ctx := NewActivationContext().
		WithCognitiveLoad(1.5). // Should clamp to 1.0
		WithTimePressure(-0.5)  // Should clamp to 0.0

	if ctx.CognitiveLoad != 1.0 {
		t.Errorf("Cognitive load should be clamped to 1.0, got %f", ctx.CognitiveLoad)
	}

	if ctx.TimePressure != 0.0 {
		t.Errorf("Time pressure should be clamped to 0.0, got %f", ctx.TimePressure)
	}
}

func TestActivationContext_Marshal(t *testing.T) {
	ctx := NewActivationContext().
		WithTask("test task").
		WithGoals("goal1").
		WithCognitiveLoad(0.7)

	data, err := ctx.Marshal()
	if err != nil {
		t.Errorf("Marshal failed: %v", err)
	}

	// Unmarshal to verify
	var ctx2 ActivationContext
	err = ctx2.Unmarshal(data)
	if err != nil {
		t.Errorf("Unmarshal failed: %v", err)
	}

	if ctx2.Task != ctx.Task {
		t.Errorf("Task not preserved after marshal/unmarshal")
	}
}

func TestActivationContext_Clone(t *testing.T) {
	ctx := NewActivationContext().
		WithTask("original").
		WithGoals("goal1", "goal2").
		WithEmotion("happy", 0.8)

	clone := ctx.Clone()

	// Verify deep copy
	clone.Task = "modified"
	if ctx.Task == clone.Task {
		t.Error("Clone should be independent")
	}

	clone.Goals[0] = "modified"
	if ctx.Goals[0] == clone.Goals[0] {
		t.Error("Goals should be deep copied")
	}

	clone.EmotionalState["happy"] = 0.3
	if ctx.EmotionalState["happy"] == clone.EmotionalState["happy"] {
		t.Error("EmotionalState should be deep copied")
	}
}

func TestActivationContext_Predicates(t *testing.T) {
	ctx := NewActivationContext().WithCognitiveLoad(0.9)
	if !ctx.IsHighLoad() {
		t.Error("Should detect high cognitive load")
	}

	ctx = ctx.WithTimePressure(0.8)
	if !ctx.IsHighPressure() {
		t.Error("Should detect high time pressure")
	}

	ctx = ctx.WithCognitiveLoad(0.3).WithTimePressure(0.2)
	if ctx.IsHighLoad() || ctx.IsHighPressure() {
		t.Error("Should not detect high load/pressure")
	}
}

func TestActivationContext_JSON(t *testing.T) {
	ctx := NewActivationContext().
		WithTask("debug").
		WithGoals("fix").
		WithEmotion("curious", 0.6).
		WithEnvironment("time", "morning")

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("JSON output:\n%s", string(data))

	// Verify it can be unmarshaled
	var ctx2 ActivationContext
	err = json.Unmarshal(data, &ctx2)
	if err != nil {
		t.Errorf("Unmarshal failed: %v", err)
	}
}
