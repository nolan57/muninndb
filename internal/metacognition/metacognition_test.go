package metacognition

import (
	"testing"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

func TestCoverageAnalyzer_Basic(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "payment system", Content: "Handles transactions"},
		{Concept: "authentication", Content: "User login"},
	}

	analyzer := NewCoverageAnalyzer(engrams)
	report := analyzer.Analyze("payment")

	if report.EntityCoverage < 0.5 {
		t.Errorf("Expected coverage >= 0.5, got %f", report.EntityCoverage)
	}

	if report.OverallScore < 0.3 {
		t.Errorf("Expected overall score >= 0.3, got %f", report.OverallScore)
	}

	t.Logf("Coverage report: %+v", report)
}

func TestCoverageAnalyzer_BlindSpots(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "payment", Content: "Payment processing"},
	}

	analyzer := NewCoverageAnalyzer(engrams)
	report := analyzer.Analyze("payment deployment kubernetes")

	if len(report.BlindSpots) == 0 {
		t.Error("Expected blind spots for unknown topics")
	}

	t.Logf("Blind spots: %v", report.BlindSpots)
}

func TestEntropyAnalyzer_Basic(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "test1", Confidence: 0.9},
		{Concept: "test2", Confidence: 0.8},
		{Concept: "test3", Confidence: 0.95},
	}

	analyzer := NewEntropyAnalyzer(engrams)
	report := analyzer.Analyze()

	if report.ConfidenceEntropy > 0.5 {
		t.Errorf("Expected low entropy for high-confidence memories, got %f", report.ConfidenceEntropy)
	}

	if report.RiskLevel != "low" {
		t.Errorf("Expected low risk level, got %s", report.RiskLevel)
	}

	t.Logf("Entropy report: %+v", report)
}

func TestEntropyAnalyzer_HighEntropy(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "test1", Confidence: 0.1},
		{Concept: "test2", Confidence: 0.9},
		{Concept: "test3", Confidence: 0.3},
		{Concept: "test4", Confidence: 0.7},
	}

	analyzer := NewEntropyAnalyzer(engrams)
	report := analyzer.Analyze()

	// Mixed confidence should produce higher entropy
	if report.ConfidenceEntropy < 0.3 {
		t.Errorf("Expected higher entropy for mixed confidence, got %f", report.ConfidenceEntropy)
	}

	t.Logf("High entropy report: %+v", report)
}

func TestEntropyAnalyzer_Distribution(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "test1", Confidence: 0.1},
		{Concept: "test2", Confidence: 0.3},
		{Concept: "test3", Confidence: 0.5},
		{Concept: "test4", Confidence: 0.7},
		{Concept: "test5", Confidence: 0.9},
	}

	analyzer := NewEntropyAnalyzer(engrams)
	dist := analyzer.GetConfidenceDistribution()

	expectedBins := 5
	if len(dist) != expectedBins {
		t.Errorf("Expected %d bins, got %d", expectedBins, len(dist))
	}

	total := 0
	for _, count := range dist {
		total += count
	}

	if total != len(engrams) {
		t.Errorf("Expected %d total, got %d", len(engrams), total)
	}

	t.Logf("Distribution: %+v", dist)
}

func TestHealthAnalyzer_Basic(t *testing.T) {
	now := time.Now()
	engrams := []*storage.Engram{
		{
			Concept:      "payment",
			Confidence:   0.9,
			LastAccess:   now,
			Associations: make([]storage.Association, 3),
		},
		{
			Concept:      "auth",
			Confidence:   0.8,
			LastAccess:   now.AddDate(0, 0, -1),
			Associations: make([]storage.Association, 2),
		},
	}

	analyzer := NewHealthAnalyzer(engrams)
	report := analyzer.Analyze("default")

	if report.OverallHealth < 0.5 {
		t.Errorf("Expected health > 0.5, got %f", report.OverallHealth)
	}

	if len(report.Alerts) > 2 {
		t.Errorf("Expected few alerts for healthy DB, got %d", len(report.Alerts))
	}

	t.Logf("Health report: Overall=%.2f, Alerts=%d", report.OverallHealth, len(report.Alerts))
}

func TestHealthAnalyzer_LowFreshness(t *testing.T) {
	oldTime := time.Now().AddDate(0, 0, -60)
	engrams := []*storage.Engram{
		{
			Concept:    "old1",
			Confidence: 0.5,
			LastAccess: oldTime,
		},
		{
			Concept:    "old2",
			Confidence: 0.5,
			LastAccess: oldTime,
		},
	}

	analyzer := NewHealthAnalyzer(engrams)
	report := analyzer.Analyze("default")

	// Should trigger low freshness alert
	hasFreshnessAlert := false
	for _, alert := range report.Alerts {
		if alert.Type == "low_freshness" {
			hasFreshnessAlert = true
			break
		}
	}

	if !hasFreshnessAlert {
		t.Error("Expected low freshness alert for old memories")
	}

	t.Logf("Freshness: %f, Alerts: %d", report.Metrics.Freshness, len(report.Alerts))
}

func TestHealthMetrics_Ranges(t *testing.T) {
	engrams := []*storage.Engram{
		{Concept: "test", Confidence: 0.8, LastAccess: time.Now()},
	}

	analyzer := NewHealthAnalyzer(engrams)
	report := analyzer.Analyze("test")

	// Verify all metrics are in valid range [0, 1]
	metrics := report.Metrics

	if metrics.Coverage < 0 || metrics.Coverage > 1 {
		t.Errorf("Coverage out of range: %f", metrics.Coverage)
	}
	if metrics.ConfidenceEntropy < 0 || metrics.ConfidenceEntropy > 1 {
		t.Errorf("Entropy out of range: %f", metrics.ConfidenceEntropy)
	}
	if metrics.Freshness < 0 || metrics.Freshness > 1 {
		t.Errorf("Freshness out of range: %f", metrics.Freshness)
	}
	if metrics.AssociationDensity < 0 {
		t.Errorf("Association density negative: %f", metrics.AssociationDensity)
	}

	t.Logf("All metrics in valid range")
}
