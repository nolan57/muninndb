package engine

import (
	"math"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// SpacingEffectConfig holds configuration for spacing effect calculation.
// Allows tuning of cognitive parameters without code changes.
type SpacingEffectConfig struct {
	// MinBonus is the minimum bonus floor (default: 0.7)
	MinBonus float64
	// MaxBonus is the maximum bonus ceiling (default: 1.3)
	MaxBonus float64
	// StabilityPen0 is bonus for 0 accesses (default: 0.7)
	StabilityPen0 float64
	// StabilityPen1 is bonus for 1 access (default: 0.9)
	StabilityPen1 float64
	// CVCoefficient is the coefficient for CV bonus (default: 0.2)
	CVCoefficient float64
	// TrendCoefficient is the coefficient for trend bonus (default: 0.3)
	TrendCoefficient float64
}

// DefaultSpacingEffectConfig returns default spacing effect configuration.
func DefaultSpacingEffectConfig() SpacingEffectConfig {
	return SpacingEffectConfig{
		MinBonus:         0.7,
		MaxBonus:         1.3,
		StabilityPen0:    0.7,
		StabilityPen1:    0.9,
		CVCoefficient:    0.2,
		TrendCoefficient: 0.3,
	}
}

// DynamicForgetting calculates the activation score for an engram using
// an enhanced Ebbinghaus model with multiple modulating factors:
//   - Base-level activation (ACT-R formula)
//   - Emotional weight modulation
//   - Spacing effect bonus
//   - Memory type decay characteristics
//
// This implements the Phase 1 cognitive enhancement for dynamic forgetting.
// Higher scores indicate more cognitively available memories.
func DynamicForgetting(engram *storage.Engram, accessHistory []AccessEvent) float64 {
	return DynamicForgettingWithConfig(engram, accessHistory, DefaultSpacingEffectConfig())
}

// DynamicForgettingWithConfig calculates activation with custom spacing effect configuration.
func DynamicForgettingWithConfig(engram *storage.Engram, accessHistory []AccessEvent, config SpacingEffectConfig) float64 {
	// Base-level activation (ACT-R formula)
	base := BaseLevelActivation(engram.AccessCount, engram.LastAccess)

	// Memory type decay modulation
	typeMod := MemoryTypeDecayModulator(engram.MemoryType, engram.LastAccess)

	// Emotional weight modulation (if available)
	emotionalMod := EmotionalWeightModulator(engram.Relevance)

	// Spacing effect bonus
	spacingBonus := SpacingEffectBonusWithConfig(accessHistory, config)

	// Combined score
	score := base * typeMod * emotionalMod * spacingBonus

	// Ensure non-negative
	if score < 0 {
		return 0
	}
	return score
}

// AccessEvent represents a single access of an engram.
type AccessEvent struct {
	Timestamp time.Time
	Context   string // optional context identifier
}

// BaseLevelActivation implements the ACT-R base-level activation formula:
// B = ln(n + 1) - d × ln(age / (n + 1))
//
// Where:
//   - n = number of accesses
//   - age = time since last access (in days)
//   - d = decay parameter (0.5 by default)
//
// This captures both frequency and recency of access.
func BaseLevelActivation(accessCount uint32, lastAccess time.Time) float64 {
	n := float64(accessCount)

	// Calculate age in days
	ageDays := time.Since(lastAccess).Hours() / 24.0
	if ageDays < 0.001 {
		ageDays = 0.001 // prevent log(0)
	}

	// ACT-R formula with standard decay parameter d=0.5
	d := 0.5
	base := math.Log(n+1) - d*math.Log(ageDays/(n+1))

	return base
}

// MemoryTypeDecayModulator adjusts decay rate based on memory type.
// Different memory types have different natural decay characteristics.
func MemoryTypeDecayModulator(mt storage.MemoryType, lastAccess time.Time) float64 {
	// Get default decay time for this memory type
	defaultDecay := mt.DefaultDecayTime() // in seconds

	// Calculate time since last access
	ageSeconds := time.Since(lastAccess).Seconds()

	// Modulator: 1.0 at age=0, decays to 0.5 at defaultDecay time
	// Uses exponential decay: mod = exp(-ln(2) × age / half_life)
	halfLife := defaultDecay * 0.5 // half of default decay time
	mod := math.Exp(-math.Ln2 * ageSeconds / halfLife)

	// Clamp to [0.5, 1.0] range to ensure some activation remains
	if mod < 0.5 {
		mod = 0.5
	}
	if mod > 1.0 {
		mod = 1.0
	}

	return mod
}

// EmotionalWeightModulator adjusts activation based on emotional significance.
// In MuninnDB, the 'Relevance' field serves as a proxy for emotional weight.
// Higher emotional weight → slower decay → higher modulation.
func EmotionalWeightModulator(emotionalWeight float32) float64 {
	// emotionalWeight is 0.0-1.0
	// Modulation: 1.0 (no emotion) to 0.7 (high emotion = slower decay)
	// High emotional weight memories decay 30% slower
	return 1.0 - (float64(emotionalWeight) * 0.3)
}

// SpacingEffectBonusWithConfig calculates bonus with custom configuration.
func SpacingEffectBonusWithConfig(history []AccessEvent, config SpacingEffectConfig) float64 {
	historyLen := len(history)

	// Short history handling: apply stability penalty for new memories
	if historyLen < 2 {
		// 0 accesses = StabilityPen0, 1 access = StabilityPen1
		if historyLen == 0 {
			return config.StabilityPen0
		}
		return config.StabilityPen1
	}

	// Calculate intervals between accesses
	intervals := make([]float64, 0, historyLen-1)
	for i := 1; i < historyLen; i++ {
		interval := history[i].Timestamp.Sub(history[i-1].Timestamp).Hours()
		intervals = append(intervals, interval)
	}

	if len(intervals) == 0 {
		return 1.0
	}

	// Calculate spacing metric: variance of intervals
	// Higher variance with increasing trend → higher bonus
	meanInterval := 0.0
	for _, interval := range intervals {
		meanInterval += interval
	}
	meanInterval /= float64(len(intervals))

	// Calculate coefficient of variation (CV = std/mean)
	variance := 0.0
	for _, interval := range intervals {
		diff := interval - meanInterval
		variance += diff * diff
	}
	variance /= float64(len(intervals))
	stdDev := math.Sqrt(variance)

	cv := 0.0
	if meanInterval > 0 {
		cv = stdDev / meanInterval
	}

	// Check for increasing interval trend (optimal spacing pattern)
	increasingTrend := calculateIncreasingTrend(intervals)

	// Bonus formula:
	// - Base bonus from CV (optimal around 0.5-1.0)
	// - Additional bonus for increasing trend
	cvBonus := math.Min(cv, 1.5) * config.CVCoefficient
	trendBonus := increasingTrend * config.TrendCoefficient

	bonus := 1.0 + cvBonus + trendBonus

	// Clamp to [MinBonus, MaxBonus] range to avoid extreme values
	if bonus < config.MinBonus {
		return config.MinBonus
	}
	if bonus > config.MaxBonus {
		return config.MaxBonus
	}
	return bonus
}

// SpacingEffectBonus uses default configuration for backward compatibility.
func SpacingEffectBonus(history []AccessEvent) float64 {
	return SpacingEffectBonusWithConfig(history, DefaultSpacingEffectConfig())
}

// calculateIncreasingTrend returns 0.0-1.0 indicating how much intervals are increasing.
// 1.0 = perfectly increasing, 0.0 = decreasing or random.
func calculateIncreasingTrend(intervals []float64) float64 {
	if len(intervals) < 2 {
		return 0.0
	}

	increasing := 0
	total := len(intervals) - 1

	for i := 1; i < len(intervals); i++ {
		if intervals[i] > intervals[i-1] {
			increasing++
		}
	}

	return float64(increasing) / float64(total)
}

// MemoryStrength represents the consolidation level of a memory.
// Higher strength = more resistant to forgetting.
type MemoryStrength uint8

const (
	StrengthFragile   MemoryStrength = 0 // newly encoded, not consolidated
	StrengthLabile    MemoryStrength = 1 // unstable, requires rehearsal
	StrengthStable    MemoryStrength = 2 // stable, normal decay curve
	StrengthRobust    MemoryStrength = 3 // robust, resistant to interference
	StrengthPermanent MemoryStrength = 4 // permanent, almost no forgetting
)

// CalculateMemoryStrength determines the strength level based on
// access patterns, consolidation state, and association connectivity.
func CalculateMemoryStrength(engram *storage.Engram, history []AccessEvent) MemoryStrength {
	// Factor 1: Access count
	accessScore := 0.0
	if engram.AccessCount > 20 {
		accessScore = 1.0
	} else if engram.AccessCount > 10 {
		accessScore = 0.8
	} else if engram.AccessCount > 5 {
		accessScore = 0.6
	} else if engram.AccessCount > 2 {
		accessScore = 0.4
	} else {
		accessScore = 0.2
	}

	// Factor 2: Spacing effect bonus
	spacingBonus := SpacingEffectBonus(history)
	spacingScore := math.Min(spacingBonus-1.0, 1.0) // normalize to 0-1

	// Factor 3: Association count (connectivity)
	assocScore := 0.0
	assocCount := len(engram.Associations)
	if assocCount > 20 {
		assocScore = 1.0
	} else if assocCount > 10 {
		assocScore = 0.8
	} else if assocCount > 5 {
		assocScore = 0.6
	} else if assocCount > 2 {
		assocScore = 0.4
	}

	// Weighted combination
	totalScore := 0.4*accessScore + 0.3*spacingScore + 0.3*assocScore

	// Map to strength levels
	if totalScore >= 0.9 {
		return StrengthPermanent
	} else if totalScore >= 0.7 {
		return StrengthRobust
	} else if totalScore >= 0.5 {
		return StrengthStable
	} else if totalScore >= 0.3 {
		return StrengthLabile
	}
	return StrengthFragile
}

// PredictRetention calculates the expected retention probability after
// a given time period, based on current strength and decay patterns.
func PredictRetention(engram *storage.Engram, strength MemoryStrength, futureDays float64) float64 {
	// Get base decay rate for memory type
	baseDecay := engram.MemoryType.DefaultDecayTime() / 86400.0 // convert to days

	// Strength modifier (higher strength = slower decay)
	strengthModifier := map[MemoryStrength]float64{
		StrengthFragile:   1.0,
		StrengthLabile:    0.7,
		StrengthStable:    0.5,
		StrengthRobust:    0.3,
		StrengthPermanent: 0.1,
	}[strength]

	// Effective decay rate
	effectiveDecay := baseDecay * strengthModifier

	// Ebbinghaus retention formula: R = exp(-t / effective_decay)
	retention := math.Exp(-futureDays / effectiveDecay)

	return retention
}
