package metacognition

import (
	"math"

	"github.com/scrypster/muninndb/internal/storage"
)

// EntropyReport represents the confidence entropy analysis.
type EntropyReport struct {
	ConfidenceEntropy float64  `json:"confidence_entropy"` // 0-1: higher = more uncertainty
	Contradictions    int      `json:"contradictions"`     // number of contradictions detected
	RiskLevel         string   `json:"risk_level"`         // "low", "medium", "high"
	Recommendations   []string `json:"recommendations"`
}

// EntropyAnalyzer analyzes confidence entropy in the knowledge base.
type EntropyAnalyzer struct {
	engrams []*storage.Engram
}

// NewEntropyAnalyzer creates a new entropy analyzer.
func NewEntropyAnalyzer(engrams []*storage.Engram) *EntropyAnalyzer {
	return &EntropyAnalyzer{
		engrams: engrams,
	}
}

// Analyze computes confidence entropy for the knowledge base.
func (ea *EntropyAnalyzer) Analyze() *EntropyReport {
	// Calculate Shannon entropy of confidence distribution
	entropy := ea.calculateShannonEntropy()

	// Normalize to 0-1 range
	normalizedEntropy := ea.normalizeEntropy(entropy)

	// Detect contradictions
	contradictions := ea.detectContradictions()

	// Determine risk level
	riskLevel := ea.determineRiskLevel(normalizedEntropy, contradictions)

	// Generate recommendations
	recommendations := ea.generateRecommendations(normalizedEntropy, contradictions)

	return &EntropyReport{
		ConfidenceEntropy: normalizedEntropy,
		Contradictions:    contradictions,
		RiskLevel:         riskLevel,
		Recommendations:   recommendations,
	}
}

// calculateShannonEntropy computes Shannon entropy of confidence distribution.
// H = -Σ p(x) × log2(p(x))
// where x represents confidence bins (0-0.2, 0.2-0.4, ..., 0.8-1.0)
func (ea *EntropyAnalyzer) calculateShannonEntropy() float64 {
	if len(ea.engrams) == 0 {
		return 0
	}

	// Bin engrams by confidence (5 bins: 0-0.2, 0.2-0.4, ..., 0.8-1.0)
	bins := make([]int, 5)
	for _, engram := range ea.engrams {
		binIndex := int(engram.Confidence * 5)
		if binIndex >= 5 {
			binIndex = 4
		}
		if binIndex < 0 {
			binIndex = 0
		}
		bins[binIndex]++
	}

	// Calculate entropy
	entropy := 0.0
	total := float64(len(ea.engrams))
	for _, count := range bins {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// normalizeEntropy normalizes entropy to 0-1 range.
// Max entropy for 5 bins is log2(5) ≈ 2.32
func (ea *EntropyAnalyzer) normalizeEntropy(entropy float64) float64 {
	maxEntropy := math.Log2(5) // ≈ 2.32 for 5 bins
	if maxEntropy == 0 {
		return 0
	}
	normalized := entropy / maxEntropy
	if normalized > 1.0 {
		normalized = 1.0
	}
	return normalized
}

// detectContradictions counts contradictions in the knowledge base.
func (ea *EntropyAnalyzer) detectContradictions() int {
	contradictions := 0

	// Check for engrams with low confidence and contradictory relationships
	for _, engram := range ea.engrams {
		if engram.Confidence < 0.5 {
			// Check for contradictory relationships
			for _, assoc := range engram.Associations {
				if assoc.RelType == storage.RelContradicts {
					contradictions++
				}
			}
		}
	}

	return contradictions
}

// determineRiskLevel determines the overall risk level based on entropy and contradictions.
func (ea *EntropyAnalyzer) determineRiskLevel(entropy float64, contradictions int) string {
	// High risk: high entropy (>0.7) OR many contradictions (>5)
	if entropy > 0.7 || contradictions > 5 {
		return "high"
	}

	// Medium risk: medium entropy (0.4-0.7) OR some contradictions (1-5)
	if entropy > 0.4 || contradictions > 0 {
		return "medium"
	}

	// Low risk: low entropy (<0.4) AND no contradictions
	return "low"
}

// generateRecommendations creates suggestions for reducing entropy.
func (ea *EntropyAnalyzer) generateRecommendations(entropy float64, contradictions int) []string {
	var recommendations []string

	if entropy > 0.7 {
		recommendations = append(recommendations,
			"High entropy detected - review and consolidate conflicting memories")
	}

	if entropy > 0.4 && entropy <= 0.7 {
		recommendations = append(recommendations,
			"Moderate entropy - consider updating low-confidence memories")
	}

	if contradictions > 5 {
		recommendations = append(recommendations,
			"Multiple contradictions detected - resolve conflicting information")
	} else if contradictions > 0 {
		recommendations = append(recommendations,
			"Review contradictory memories and update confidence scores")
	}

	if entropy < 0.3 && contradictions == 0 {
		recommendations = append(recommendations,
			"Knowledge base shows good consistency - maintain current practices")
	}

	return recommendations
}

// GetConfidenceDistribution returns the confidence distribution across bins.
func (ea *EntropyAnalyzer) GetConfidenceDistribution() map[string]int {
	bins := map[string]int{
		"0.0-0.2": 0,
		"0.2-0.4": 0,
		"0.4-0.6": 0,
		"0.6-0.8": 0,
		"0.8-1.0": 0,
	}

	for _, engram := range ea.engrams {
		confidence := float64(engram.Confidence)
		switch {
		case confidence < 0.2:
			bins["0.0-0.2"]++
		case confidence < 0.4:
			bins["0.2-0.4"]++
		case confidence < 0.6:
			bins["0.4-0.6"]++
		case confidence < 0.8:
			bins["0.6-0.8"]++
		default:
			bins["0.8-1.0"]++
		}
	}

	return bins
}
