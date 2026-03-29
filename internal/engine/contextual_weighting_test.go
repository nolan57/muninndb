package engine

import (
	"testing"

	"github.com/scrypster/muninndb/internal/storage"
)

func TestContextualScore_Basic(t *testing.T) {
	engram := &storage.Engram{
		Concept: "payment system",
		Content: "The payment system handles transactions and idempotency",
	}

	score := ContextualScore(engram, "payment", []string{})

	if score < 1.0 {
		t.Errorf("Base score should be at least 1.0, got %f", score)
	}

	t.Logf("Contextual score: %f", score)
}

func TestContextualScore_TaskRelevance(t *testing.T) {
	engram1 := &storage.Engram{
		Concept: "payment system",
		Content: "Handles payment transactions",
	}

	engram2 := &storage.Engram{
		Concept: "authentication",
		Content: "User login and JWT tokens",
	}

	score1 := ContextualScore(engram1, "payment", []string{})
	score2 := ContextualScore(engram2, "payment", []string{})

	if score1 <= score2 {
		t.Errorf("Payment engram should score higher for payment task")
	}

	t.Logf("Payment score: %f, Auth score: %f", score1, score2)
}

func TestContextualScore_GoalAlignment(t *testing.T) {
	engram := &storage.Engram{
		Concept: "idempotency",
		Content: "Prevents duplicate payments with idempotency keys",
	}

	goals := []string{"prevent duplicate charge", "ensure reliability"}
	score := ContextualScore(engram, "payment", goals)

	if score < 1.0 {
		t.Errorf("Score should be at least 1.0, got %f", score)
	}

	t.Logf("Score with goals: %f", score)
}

func TestSemanticSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		content string
		query   string
		wantMin float64
	}{
		{"exact match", "payment system", "payment", 0.5},
		{"partial match", "payment system handles transactions", "payment", 0.2},
		{"no match", "authentication module", "payment", 0.0},
		{"empty query", "payment", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := semanticSimilaritySimple(tt.content, "", tt.query)
			if got < tt.wantMin {
				t.Errorf("Similarity too low: got %f, want >= %f", got, tt.wantMin)
			}
		})
	}
}

func TestCognitiveLoadAdjustment(t *testing.T) {
	engram := &storage.Engram{
		Concept:    "complex system",
		Content:    "This is a very long and complex description with many details about the system architecture and implementation...",
		Confidence: 0.8,
		Tags:       []string{"arch", "complex", "detailed"},
	}

	baseScore := 1.5

	// Low cognitive load: no penalty
	score1 := adjustForCognitiveLoad(baseScore, engram, 0.3)
	if score1 != baseScore {
		t.Errorf("Low load should not penalize: got %f, want %f", score1, baseScore)
	}

	// High cognitive load: penalty applied
	score2 := adjustForCognitiveLoad(baseScore, engram, 0.9)
	if score2 >= score1 {
		t.Errorf("High load should reduce score: got %f, want < %f", score2, score1)
	}

	t.Logf("Low load score: %f, High load score: %f", score1, score2)
}

func TestTimePressureAdjustment(t *testing.T) {
	engram := &storage.Engram{
		Concept:    "simple fix",
		Content:    "Quick fix",
		Confidence: 0.95,
	}

	baseScore := 1.5

	// Low time pressure: no bonus
	score1 := adjustForTimePressure(baseScore, engram, 0.3)
	if score1 != baseScore {
		t.Errorf("Low pressure should not boost: got %f, want %f", score1, baseScore)
	}

	// High time pressure: boost high-confidence memories
	score2 := adjustForTimePressure(baseScore, engram, 0.9)
	if score2 <= score1 {
		t.Errorf("High pressure should boost score: got %f, want > %f", score2, score1)
	}

	t.Logf("Low pressure score: %f, High pressure score: %f", score1, score2)
}

func TestEmotionalCongruence(t *testing.T) {
	engram := &storage.Engram{
		Concept: "frustrating bug",
		Content: "This bug is causing frustration and anxiety",
	}

	emotionalState := map[string]float64{
		"frustration": 0.8,
		"anxiety":     0.6,
	}

	score := emotionalCongruence(engram, emotionalState)

	if score <= 0 {
		t.Errorf("Emotional congruence should be positive, got %f", score)
	}

	t.Logf("Emotional congruence: %f", score)
}

func TestFinalContextualScore(t *testing.T) {
	engram := &storage.Engram{
		Concept:    "payment bug fix",
		Content:    "Fix for payment idempotency issue",
		Confidence: 0.9,
		Tags:       []string{"bug", "payment"},
	}

	score := FinalContextualScore(
		engram,
		"debug payment",
		[]string{"fix bug", "prevent recurrence"},
		0.7, // cognitive load
		0.5, // time pressure
		map[string]float64{"frustration": 0.6},
		"office",          // current location
		[]string{"alice"}, // collaborators
		nil,               // no embedding provider
	)

	if score < 1.0 {
		t.Errorf("Final score should be at least 1.0, got %f", score)
	}

	t.Logf("Final contextual score: %f", score)
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		count int
	}{
		{"hello world", 2},
		{"hello, world!", 2},
		{"", 0},
		{"single", 1},
	}

	for _, tt := range tests {
		tokens := tokenize(tt.input)
		if len(tokens) != tt.count {
			t.Errorf("tokenize(%q) = %d tokens, want %d", tt.input, len(tokens), tt.count)
		}
	}
}
