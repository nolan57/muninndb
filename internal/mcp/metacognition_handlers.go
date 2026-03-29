package mcp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/scrypster/muninndb/internal/metacognition"
	"github.com/scrypster/muninndb/internal/storage"
	"github.com/scrypster/muninndb/internal/transport/mbp"
)

// handleMetacognitionHealth handles muninn_metacognition_health MCP tool calls.
func (s *MCPServer) handleMetacognitionHealth(ctx context.Context, w http.ResponseWriter, id json.RawMessage, vault string, args map[string]any) {
	// Get all engrams for analysis
	engrams := s.getAllEngramsForAnalysis(ctx, vault)

	analyzer := metacognition.NewHealthAnalyzer(engrams)
	report := analyzer.Analyze(vault)

	sendResult(w, id, map[string]any{
		"overall_health":  report.OverallHealth,
		"metrics":         report.Metrics,
		"alerts":          report.Alerts,
		"blind_spots":     report.BlindSpots,
		"recommendations": report.Recommendations,
		"timestamp":       report.Timestamp.Format("2006-01-02T15:04:05Z"),
	})
}

// handleMetacognitionCoverage handles muninn_metacognition_coverage MCP tool calls.
func (s *MCPServer) handleMetacognitionCoverage(ctx context.Context, w http.ResponseWriter, id json.RawMessage, vault string, args map[string]any) {
	query, _ := args["query"].(string)
	if query == "" {
		sendError(w, id, -32000, "missing required parameter: query")
		return
	}

	engrams := s.getAllEngramsForAnalysis(ctx, vault)
	analyzer := metacognition.NewCoverageAnalyzer(engrams)
	report := analyzer.Analyze(query)

	sendResult(w, id, map[string]any{
		"query":           report.Query,
		"entity_coverage": report.EntityCoverage,
		"topic_density":   report.TopicDensity,
		"overall_score":   report.OverallScore,
		"blind_spots":     report.BlindSpots,
		"recommendations": report.Recommendations,
		"timestamp":       report.Timestamp.Format("2006-01-02T15:04:05Z"),
	})
}

// handleMetacognitionEntropy handles muninn_metacognition_entropy MCP tool calls.
func (s *MCPServer) handleMetacognitionEntropy(ctx context.Context, w http.ResponseWriter, id json.RawMessage, vault string, args map[string]any) {
	engrams := s.getAllEngramsForAnalysis(ctx, vault)
	analyzer := metacognition.NewEntropyAnalyzer(engrams)
	report := analyzer.Analyze()

	sendResult(w, id, map[string]any{
		"confidence_entropy": report.ConfidenceEntropy,
		"contradictions":     report.Contradictions,
		"risk_level":         report.RiskLevel,
		"recommendations":    report.Recommendations,
	})
}

// getAllEngramsForAnalysis retrieves a sample of engrams from a vault for metacognition analysis.
// For large vaults, this samples up to 1000 engrams to keep analysis fast.
func (s *MCPServer) getAllEngramsForAnalysis(ctx context.Context, vault string) []*storage.Engram {
	engrams := make([]*storage.Engram, 0, 100)

	// Use Activate with empty context to get engrams
	resp, err := s.engine.Activate(ctx, &mbp.ActivateRequest{
		Context:    []string{""},
		MaxResults: 1000,
		Vault:      vault,
	})
	if err != nil {
		return engrams
	}

	for _, item := range resp.Activations {
		// Read full engram
		readResp, err := s.engine.Read(ctx, &mbp.ReadRequest{
			Vault: vault,
			ID:    item.ID,
		})
		if err != nil {
			continue
		}
		// Convert ReadResponse to storage.Engram
		engram := &storage.Engram{
			Concept:    readResp.Concept,
			Content:    readResp.Content,
			Confidence: readResp.Confidence,
			Relevance:  readResp.Relevance,
		}
		engrams = append(engrams, engram)
	}

	return engrams
}
