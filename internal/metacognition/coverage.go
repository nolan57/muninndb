package metacognition

import (
	"encoding/json"
	"math"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// CoverageReport represents the knowledge coverage analysis result.
type CoverageReport struct {
	Query           string    `json:"query"`
	EntityCoverage  float64   `json:"entity_coverage"` // 0-1: known entities / total entities
	TopicDensity    float64   `json:"topic_density"`   // 0-1: vector space density
	OverallScore    float64   `json:"overall_score"`   // weighted combination
	BlindSpots      []string  `json:"blind_spots"`     // topics with low coverage
	Recommendations []string  `json:"recommendations"` // suggestions for improvement
	Timestamp       time.Time `json:"timestamp"`
}

// CoverageAnalyzer analyzes knowledge coverage for a given query.
type CoverageAnalyzer struct {
	engrams []*storage.Engram
}

// NewCoverageAnalyzer creates a new coverage analyzer.
func NewCoverageAnalyzer(engrams []*storage.Engram) *CoverageAnalyzer {
	return &CoverageAnalyzer{
		engrams: engrams,
	}
}

// Analyze computes coverage for a given query.
func (ca *CoverageAnalyzer) Analyze(query string) *CoverageReport {
	queryEntities := ca.extractEntities(query)

	// Calculate entity coverage
	knownEntities := ca.countKnownEntities(queryEntities)
	entityCoverage := 0.0
	if len(queryEntities) > 0 {
		entityCoverage = float64(knownEntities) / float64(len(queryEntities))
	}

	// Calculate topic density
	topicDensity := ca.calculateTopicDensity(query)

	// Overall score: 60% entity + 40% topic
	overallScore := 0.6*entityCoverage + 0.4*topicDensity

	// Identify blind spots
	blindSpots := ca.identifyBlindSpots(queryEntities, knownEntities)

	// Generate recommendations
	recommendations := ca.generateRecommendations(blindSpots)

	return &CoverageReport{
		Query:           query,
		EntityCoverage:  entityCoverage,
		TopicDensity:    topicDensity,
		OverallScore:    overallScore,
		BlindSpots:      blindSpots,
		Recommendations: recommendations,
		Timestamp:       time.Now(),
	}
}

// extractEntities extracts entity names from a query (simple implementation).
func (ca *CoverageAnalyzer) extractEntities(query string) []string {
	// Simple tokenization - in production, use NER
	words := tokenize(query)
	return words
}

// countKnownEntities counts how many query entities exist in memory.
func (ca *CoverageAnalyzer) countKnownEntities(entities []string) int {
	known := 0
	for _, entity := range entities {
		if ca.entityExists(entity) {
			known++
		}
	}
	return known
}

// entityExists checks if an entity exists in memory.
func (ca *CoverageAnalyzer) entityExists(entity string) bool {
	for _, engram := range ca.engrams {
		if containsIgnoreCase(engram.Concept, entity) ||
			containsIgnoreCase(engram.Content, entity) {
			return true
		}
	}
	return false
}

// calculateTopicDensity computes vector space density for the query topic.
func (ca *CoverageAnalyzer) calculateTopicDensity(query string) float64 {
	if len(ca.engrams) == 0 {
		return 0
	}

	// Count engrams relevant to query
	relevantCount := 0
	for _, engram := range ca.engrams {
		if containsIgnoreCase(engram.Content, query) ||
			containsIgnoreCase(engram.Concept, query) {
			relevantCount++
		}
	}

	// Density = relevant / total (normalized)
	density := float64(relevantCount) / float64(len(ca.engrams))

	// Apply logarithmic scaling to avoid extreme values
	if density > 0 {
		density = math.Log10(density*10+1) / math.Log10(11)
	}

	return density
}

// identifyBlindSpots finds topics with low coverage.
func (ca *CoverageAnalyzer) identifyBlindSpots(entities []string, knownCount int) []string {
	var blindSpots []string

	if knownCount < len(entities) {
		// Some entities are not covered
		for _, entity := range entities {
			if !ca.entityExists(entity) {
				blindSpots = append(blindSpots, entity)
			}
		}
	}

	return blindSpots
}

// generateRecommendations creates suggestions for improving coverage.
func (ca *CoverageAnalyzer) generateRecommendations(blindSpots []string) []string {
	var recommendations []string

	for _, spot := range blindSpots {
		recommendations = append(recommendations,
			"Add memories about "+spot)
	}

	if len(blindSpots) == 0 {
		recommendations = append(recommendations,
			"Coverage is good - continue maintaining knowledge base")
	}

	return recommendations
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(containsSlow(s, substr)))
}

func containsSlow(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

func tokenize(text string) []string {
	var tokens []string
	current := ""
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			current += string(r)
		} else {
			if current != "" {
				tokens = append(tokens, current)
				current = ""
			}
		}
	}
	if current != "" {
		tokens = append(tokens, current)
	}
	return tokens
}

// Marshal serializes the report to JSON.
func (r *CoverageReport) Marshal() ([]byte, error) {
	return json.Marshal(r)
}
