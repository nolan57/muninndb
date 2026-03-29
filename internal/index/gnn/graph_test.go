package gnn

import (
	"testing"
)

func TestAssociationGraph_Basic(t *testing.T) {
	graph := NewAssociationGraph()

	// Add nodes
	id1 := graph.AddEngramNode([16]byte{1}, "payment system", 0x83)
	id2 := graph.AddEngramNode([16]byte{2}, "idempotency key", 0x83)
	id3 := graph.AddEngramNode([16]byte{3}, "payment failure", 0x82)

	if id1 == 0 || id2 == 0 || id3 == 0 {
		t.Error("Node IDs should be non-zero")
	}

	if graph.NodeCount() != 3 {
		t.Errorf("Expected 3 nodes, got %d", graph.NodeCount())
	}

	// Add relations
	_, err := graph.AddRelation([16]byte{1}, [16]byte{2}, RelTaxonomic, 0.8, 0.9)
	if err != nil {
		t.Errorf("AddRelation failed: %v", err)
	}

	_, err = graph.AddRelation([16]byte{2}, [16]byte{3}, RelCausal, 0.9, 0.95)
	if err != nil {
		t.Errorf("AddRelation failed: %v", err)
	}

	if graph.EdgeCount() != 2 {
		t.Errorf("Expected 2 edges, got %d", graph.EdgeCount())
	}
}

func TestAssociationGraph_Traverse(t *testing.T) {
	graph := NewAssociationGraph()

	// Create a graph: A -> B -> C, A -> D
	graph.AddEngramNode([16]byte{1}, "A", 0x83)
	graph.AddEngramNode([16]byte{2}, "B", 0x83)
	graph.AddEngramNode([16]byte{3}, "C", 0x83)
	graph.AddEngramNode([16]byte{4}, "D", 0x83)

	graph.AddRelation([16]byte{1}, [16]byte{2}, RelCausal, 0.8, 0.9)
	graph.AddRelation([16]byte{2}, [16]byte{3}, RelTemporal, 0.7, 0.85)
	graph.AddRelation([16]byte{1}, [16]byte{4}, RelAnalogical, 0.6, 0.8)

	// Traverse from A
	nodes := graph.Traverse([16]byte{1}, 2)

	if len(nodes) < 3 {
		t.Errorf("Expected at least 3 nodes from traversal, got %d", len(nodes))
	}

	t.Logf("Traversed %d nodes", len(nodes))
}

func TestAssociationGraph_StrengthenRelation(t *testing.T) {
	graph := NewAssociationGraph()

	graph.AddEngramNode([16]byte{1}, "concept1", 0x83)
	graph.AddEngramNode([16]byte{2}, "concept2", 0x83)

	graph.AddRelation([16]byte{1}, [16]byte{2}, RelCoActivation, 0.5, 0.8)

	// Strengthen the relation
	newWeight := graph.StrengthenRelation([16]byte{1}, [16]byte{2}, 0.2)

	if newWeight != 0.7 {
		t.Errorf("Expected weight 0.7, got %f", newWeight)
	}

	// Strengthen beyond 1.0 (should clamp)
	newWeight = graph.StrengthenRelation([16]byte{1}, [16]byte{2}, 0.5)
	if newWeight != 1.0 {
		t.Errorf("Expected clamped weight 1.0, got %f", newWeight)
	}
}

func TestAssociationGraph_FilterByType(t *testing.T) {
	graph := NewAssociationGraph()

	// Create nodes
	graph.AddEngramNode([16]byte{1}, "A", 0x83)
	graph.AddEngramNode([16]byte{2}, "B", 0x83)
	graph.AddEngramNode([16]byte{3}, "C", 0x83)

	// Add different relation types
	graph.AddRelation([16]byte{1}, [16]byte{2}, RelCausal, 0.8, 0.9)
	graph.AddRelation([16]byte{1}, [16]byte{3}, RelAnalogical, 0.7, 0.85)
	graph.AddRelation([16]byte{2}, [16]byte{3}, RelTemporal, 0.6, 0.8)

	// Filter by causal
	causal := graph.GetRelationsByType([16]byte{1}, RelCausal)
	if len(causal) != 1 {
		t.Errorf("Expected 1 causal relation, got %d", len(causal))
	}
	if causal[0].RelType != RelCausal {
		t.Errorf("Expected RelCausal, got %v", causal[0].RelType)
	}

	// Filter by analogical
	analogical := graph.GetRelationsByType([16]byte{1}, RelAnalogical)
	if len(analogical) != 1 {
		t.Errorf("Expected 1 analogical relation, got %d", len(analogical))
	}
}

func TestAssociationGraph_Search(t *testing.T) {
	graph := NewAssociationGraph()

	graph.AddEngramNode([16]byte{1}, "payment system", 0x83)
	graph.AddEngramNode([16]byte{2}, "authentication module", 0x83)
	graph.AddEngramNode([16]byte{3}, "payment gateway", 0x83)

	// Search for "payment"
	results := graph.FindNodesByConcept("payment", 10)

	if len(results) != 2 {
		t.Errorf("Expected 2 payment-related nodes, got %d", len(results))
	}

	// Search with limit
	results = graph.FindNodesByConcept("payment", 1)
	if len(results) != 1 {
		t.Errorf("Expected 1 result with limit, got %d", len(results))
	}
}
