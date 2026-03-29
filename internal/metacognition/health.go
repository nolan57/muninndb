package metacognition

import (
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// HealthReport represents the overall knowledge base health assessment.
type HealthReport struct {
	Vault           string        `json:"vault"`
	Timestamp       time.Time     `json:"timestamp"`
	OverallHealth   float64       `json:"overall_health"` // 0-1: overall health score
	Metrics         HealthMetrics `json:"metrics"`
	Alerts          []HealthAlert `json:"alerts"`
	BlindSpots      []string      `json:"blind_spots"`
	Recommendations []string      `json:"recommendations"`
}

// HealthMetrics contains individual health metrics.
type HealthMetrics struct {
	Coverage           float64 `json:"coverage"`            // knowledge coverage (0-1)
	ConfidenceEntropy  float64 `json:"confidence_entropy"`  // uncertainty level (0-1, lower is better)
	Freshness          float64 `json:"freshness"`           // recently accessed memories (0-1)
	AssociationDensity float64 `json:"association_density"` // avg associations per engram
	PrototypeCoverage  float64 `json:"prototype_coverage"`  // engrams with prototypes (0-1)
}

// HealthAlert represents a health issue or warning.
type HealthAlert struct {
	Type           string `json:"type"`     // "low_freshness", "high_entropy", etc.
	Severity       string `json:"severity"` // "warning", "critical"
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

// HealthAnalyzer performs comprehensive health analysis.
type HealthAnalyzer struct {
	coverageAnalyzer *CoverageAnalyzer
	entropyAnalyzer  *EntropyAnalyzer
	engrams          []*storage.Engram
}

// NewHealthAnalyzer creates a new health analyzer.
func NewHealthAnalyzer(engrams []*storage.Engram) *HealthAnalyzer {
	return &HealthAnalyzer{
		coverageAnalyzer: NewCoverageAnalyzer(engrams),
		entropyAnalyzer:  NewEntropyAnalyzer(engrams),
		engrams:          engrams,
	}
}

// Analyze performs comprehensive health analysis.
func (ha *HealthAnalyzer) Analyze(vault string) *HealthReport {
	// Get coverage report (using empty query for overall coverage)
	coverageReport := ha.coverageAnalyzer.Analyze("")

	// Get entropy report
	entropyReport := ha.entropyAnalyzer.Analyze()

	// Calculate freshness
	freshness := ha.calculateFreshness()

	// Calculate association density
	assocDensity := ha.calculateAssociationDensity()

	// Calculate prototype coverage (TODO: implement when prototype module is available)
	// For now, use a placeholder value based on association density
	prototypeCoverage := 0.0 // TODO: implement prototype tracking
	_ = prototypeCoverage    // Suppress unused variable warning

	// Overall health: weighted combination
	// Coverage: 30%, (1-Entropy): 30%, Freshness: 20%, AssocDensity: 20%
	overallHealth := 0.3*coverageReport.OverallScore +
		0.3*(1.0-entropyReport.ConfidenceEntropy) +
		0.2*freshness +
		0.2*normalizeAssocDensity(assocDensity)

	// Generate alerts
	alerts := ha.generateAlerts(coverageReport, entropyReport, freshness, assocDensity)

	// Generate recommendations
	recommendations := ha.generateRecommendations(coverageReport, entropyReport, freshness)

	return &HealthReport{
		Vault:         vault,
		Timestamp:     time.Now(),
		OverallHealth: overallHealth,
		Metrics: HealthMetrics{
			Coverage:           coverageReport.OverallScore,
			ConfidenceEntropy:  entropyReport.ConfidenceEntropy,
			Freshness:          freshness,
			AssociationDensity: assocDensity,
			PrototypeCoverage:  prototypeCoverage,
		},
		Alerts:          alerts,
		BlindSpots:      coverageReport.BlindSpots,
		Recommendations: recommendations,
	}
}

// calculateFreshness computes the percentage of recently accessed memories.
func (ha *HealthAnalyzer) calculateFreshness() float64 {
	if len(ha.engrams) == 0 {
		return 0
	}

	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	recentCount := 0.0
	for _, engram := range ha.engrams {
		if engram.LastAccess.After(sevenDaysAgo) {
			recentCount++
		} else if engram.LastAccess.After(thirtyDaysAgo) {
			recentCount += 0.5 // Partial credit for 7-30 days
		}
	}

	return recentCount / float64(len(ha.engrams))
}

// calculateAssociationDensity computes average associations per engram.
func (ha *HealthAnalyzer) calculateAssociationDensity() float64 {
	if len(ha.engrams) == 0 {
		return 0
	}

	totalAssocs := 0
	for _, engram := range ha.engrams {
		totalAssocs += len(engram.Associations)
	}

	return float64(totalAssocs) / float64(len(ha.engrams))
}

func normalizeAssocDensity(density float64) float64 {
	// Normalize: 0 assocs = 0, 10+ assocs = 1.0
	if density >= 10 {
		return 1.0
	}
	return density / 10.0
}

// generateAlerts creates health alerts based on metrics.
func (ha *HealthAnalyzer) generateAlerts(
	coverage *CoverageReport,
	entropy *EntropyReport,
	freshness float64,
	assocDensity float64,
) []HealthAlert {
	var alerts []HealthAlert

	// Low freshness alert
	if freshness < 0.2 {
		alerts = append(alerts, HealthAlert{
			Type:           "low_freshness",
			Severity:       "warning",
			Message:        "Less than 20% of memories accessed in the last 30 days",
			Recommendation: "Review and prune outdated memories",
		})
	}

	// High entropy alert
	if entropy.ConfidenceEntropy > 0.7 {
		severity := "warning"
		if entropy.ConfidenceEntropy > 0.85 {
			severity = "critical"
		}
		alerts = append(alerts, HealthAlert{
			Type:           "high_entropy",
			Severity:       severity,
			Message:        "High uncertainty in knowledge base",
			Recommendation: "Resolve conflicting memories",
		})
	}

	// Low coverage alert
	if coverage.OverallScore < 0.3 {
		alerts = append(alerts, HealthAlert{
			Type:           "low_coverage",
			Severity:       "warning",
			Message:        "Knowledge coverage is below 30%",
			Recommendation: "Add more memories about key topics",
		})
	}

	// Low association density alert
	if assocDensity < 2 {
		alerts = append(alerts, HealthAlert{
			Type:           "low_associations",
			Severity:       "warning",
			Message:        "Average associations per memory is below 2",
			Recommendation: "Create more connections between memories",
		})
	}

	return alerts
}

// generateRecommendations creates improvement suggestions.
func (ha *HealthAnalyzer) generateRecommendations(
	coverage *CoverageReport,
	entropy *EntropyReport,
	freshness float64,
) []string {
	var recommendations []string

	// Add coverage recommendations
	recommendations = append(recommendations, coverage.Recommendations...)

	// Add entropy recommendations
	recommendations = append(recommendations, entropy.Recommendations...)

	// Freshness recommendations
	if freshness < 0.5 {
		recommendations = append(recommendations,
			"Consider archiving memories older than 90 days without access")
	}

	return recommendations
}
