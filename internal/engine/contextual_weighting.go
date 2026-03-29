package engine

import (
	"math"
	"strings"

	"github.com/scrypster/muninndb/internal/storage"
)

// EmbeddingProvider defines the interface for embedding computation.
// This allows injection of any embedding provider (ONNX, external API, etc.)
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
}

// semanticSimilarityHybrid computes similarity using embeddings if available,
// otherwise falls back to token overlap. Returns weighted average:
// 0.7 * embedSim + 0.3 * simpleSim
func semanticSimilarityHybrid(a, b string, embProvider EmbeddingProvider) float64 {
	// Try embedding-based similarity first
	if embProvider != nil {
		embedA, errA := embProvider.Embed(a)
		embedB, errB := embProvider.Embed(b)

		if errA == nil && errB == nil && len(embedA) > 0 && len(embedB) > 0 {
			cosSim := cosineSimilarity(embedA, embedB)
			// Blend with simple similarity for robustness
			simpleSim := semanticSimilaritySimple(a, "", b)
			return 0.7*cosSim + 0.3*simpleSim
		}
	}

	// Fallback to simple token overlap
	return semanticSimilaritySimple(a, "", b)
}

// cosineSimilarity computes cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ContextualScoreWithEmbedding calculates context-aware score with hybrid similarity.
// Uses embedding-based similarity when provider is available.
func ContextualScoreWithEmbedding(
	engram *storage.Engram,
	taskContext string,
	goals []string,
	embProvider EmbeddingProvider,
) float64 {
	base := 1.0

	// Task relevance with hybrid similarity
	textA := engram.Content + " " + engram.Concept
	taskRelevance := semanticSimilarityHybrid(textA, taskContext, embProvider)

	// Goal alignment with hybrid similarity
	goalAlignment := maxGoalAlignmentHybrid(engram, goals, embProvider)

	score := base + 0.3*taskRelevance + 0.2*goalAlignment
	return score
}

// maxGoalAlignmentHybrid returns max alignment using hybrid similarity.
func maxGoalAlignmentHybrid(engram *storage.Engram, goals []string, embProvider EmbeddingProvider) float64 {
	if len(goals) == 0 {
		return 0
	}

	maxAlign := 0.0
	textA := engram.Content + " " + engram.Concept

	for _, goal := range goals {
		align := semanticSimilarityHybrid(textA, goal, embProvider)
		if align > maxAlign {
			maxAlign = align
		}
	}

	return maxAlign
}

// adjustForCollaboratorRelevance boosts score if engram involves current collaborators.
func adjustForCollaboratorRelevance(score float64, engram *storage.Engram, collaborators []string) float64 {
	if len(collaborators) == 0 {
		return score
	}

	// Check if any collaborator appears in engram content or tags
	for _, collab := range collaborators {
		if strings.Contains(strings.ToLower(engram.Content), strings.ToLower(collab)) ||
			strings.Contains(strings.ToLower(engram.Concept), strings.ToLower(collab)) {
			for _, tag := range engram.Tags {
				if strings.Contains(strings.ToLower(tag), strings.ToLower(collab)) {
					return score + 0.3 // Bonus for collaborator match
				}
			}
			return score + 0.15 // Smaller bonus for content match only
		}
	}

	return score
}

// FinalContextualScore computes the final context-aware score with all enhancements.
func FinalContextualScore(
	engram *storage.Engram,
	taskContext string,
	goals []string,
	cognitiveLoad float64,
	timePressure float64,
	emotionalState map[string]float64,
	currentLocation string,
	collaborators []string,
	embProvider EmbeddingProvider,
) float64 {
	// Use embedding-aware scoring if provider available
	var score float64
	if embProvider != nil {
		score = ContextualScoreWithEmbedding(engram, taskContext, goals, embProvider)
	} else {
		score = ContextualScore(engram, taskContext, goals)
	}

	// Apply cognitive load adjustment
	score = adjustForCognitiveLoad(score, engram, cognitiveLoad)

	// Apply time pressure adjustment
	score = adjustForTimePressure(score, engram, timePressure)

	// Apply emotional congruence
	score += emotionalCongruence(engram, emotionalState) * 0.2

	// Apply collaborator relevance
	score = adjustForCollaboratorRelevance(score, engram, collaborators)

	_ = currentLocation // Reserved for future location-based modulation

	return score
}

// semanticSimilaritySimple computes a simple semantic similarity score.
func semanticSimilaritySimple(content, concept, query string) float64 {
	if query == "" {
		return 0
	}

	contentWords := tokenize(strings.ToLower(content + " " + concept))
	queryWords := tokenize(strings.ToLower(query))

	if len(contentWords) == 0 || len(queryWords) == 0 {
		return 0
	}

	overlap := 0
	queryWordSet := make(map[string]struct{})
	for _, word := range queryWords {
		queryWordSet[word] = struct{}{}
	}

	for _, word := range contentWords {
		if _, found := queryWordSet[word]; found {
			overlap++
		}
	}

	similarity := float64(overlap) / float64(len(queryWordSet))
	return math.Min(similarity, 1.0)
}

// maxGoalAlignment returns the maximum alignment score across all goals.
func maxGoalAlignment(engram *storage.Engram, goals []string) float64 {
	if len(goals) == 0 {
		return 0
	}

	maxAlign := 0.0
	for _, goal := range goals {
		align := semanticSimilaritySimple(engram.Content, engram.Concept, goal)
		if align > maxAlign {
			maxAlign = align
		}
	}

	return maxAlign
}

// tokenize splits text into words.
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

// adjustForCognitiveLoad modulates score based on cognitive load.
func adjustForCognitiveLoad(score float64, engram *storage.Engram, cognitiveLoad float64) float64 {
	if cognitiveLoad <= 0.5 {
		return score
	}

	complexity := estimateComplexity(engram)
	penalty := (cognitiveLoad - 0.5) * complexity * 0.5

	return score - penalty
}

// adjustForTimePressure modulates score based on time pressure.
func adjustForTimePressure(score float64, engram *storage.Engram, timePressure float64) float64 {
	if timePressure <= 0.5 {
		return score
	}

	confidenceBonus := (timePressure - 0.5) * float64(engram.Confidence) * 0.5

	return score + confidenceBonus
}

// estimateComplexity estimates the complexity of an engram.
func estimateComplexity(engram *storage.Engram) float64 {
	contentLen := len(engram.Content)
	tagCount := len(engram.Tags)

	lenScore := math.Min(float64(contentLen)/1000.0, 1.0)
	tagScore := math.Min(float64(tagCount)/10.0, 1.0)

	return 0.7*lenScore + 0.3*tagScore
}

// emotionalCongruence calculates emotional match.
func emotionalCongruence(engram *storage.Engram, emotionalState map[string]float64) float64 {
	if len(emotionalState) == 0 {
		return 0
	}

	totalMatch := 0.0
	for emotion, intensity := range emotionalState {
		if strings.Contains(strings.ToLower(engram.Content), strings.ToLower(emotion)) {
			totalMatch += intensity * 0.5
		}
	}

	return math.Min(totalMatch, 1.0)
}

// ContextualScore calculates a context-aware activation score for an engram.
// Backward-compatible version without embedding support.
func ContextualScore(engram *storage.Engram, taskContext string, goals []string) float64 {
	base := 1.0
	taskRelevance := semanticSimilaritySimple(engram.Content, engram.Concept, taskContext)
	goalAlignment := maxGoalAlignment(engram, goals)
	score := base + 0.3*taskRelevance + 0.2*goalAlignment
	return score
}

// adjustForLocationMatch boosts score if engram metadata matches current location.
func adjustForLocationMatch(score float64, engram *storage.Engram, currentLocation string) float64 {
	if currentLocation == "" {
		return score
	}

	// Check if location appears in engram tags
	for _, tag := range engram.Tags {
		if containsIgnoreCase(tag, currentLocation) {
			return score + 0.2
		}
	}

	// Smaller bonus for content match
	if containsIgnoreCase(engram.Content, currentLocation) ||
		containsIgnoreCase(engram.Concept, currentLocation) {
		return score + 0.1
	}

	return score
}

// adjustForCollaboratorRelevance boosts score if engram involves current collaborators.


func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
