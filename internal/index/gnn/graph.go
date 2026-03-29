package gnn

import (
	"fmt"
	"sort"
	"strings"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// RelationType represents the type of semantic relationship.
type RelationType uint8

const (
	RelUnknown RelationType = iota
	RelCoActivation
	RelCausal
	RelAnalogical
	RelMetaphorical
	RelTemporal
	RelPartonomic
	RelTaxonomic
	RelSupports
	RelContradicts
)

// RelationWeightMap maps relation types to traversal weights.
// Higher weight = higher priority during traversal.
type RelationWeightMap map[RelationType]float64

// DefaultRelationWeights returns default weights for relation types.
// Causal relations get highest priority, followed by analogical.
func DefaultRelationWeights() RelationWeightMap {
	return RelationWeightMap{
		RelCausal:       1.5,
		RelAnalogical:   1.3,
		RelMetaphorical: 1.2,
		RelTaxonomic:    1.1,
		RelPartonomic:   1.1,
		RelTemporal:     1.0,
		RelSupports:     1.0,
		RelCoActivation: 1.0,
		RelContradicts:  0.8, // Lower priority for contradictions
	}
}

// CognitiveNode represents a cognitive node in the association graph.
type CognitiveNode struct {
	nodeID    int64
	EngramID  [16]byte
	Concept   string
	MemType   uint8
	Strength  float64
	Activated bool
}

// ID returns the node's unique identifier.
func (n *CognitiveNode) ID() int64 {
	return n.nodeID
}

// CognitiveEdge represents a cognitive association between two nodes.
type CognitiveEdge struct {
	fromNode   *CognitiveNode
	toNode     *CognitiveNode
	RelType    RelationType
	edgeWeight float64
	Confidence float64
}

// From returns the source node.
func (e *CognitiveEdge) From() graph.Node {
	return e.fromNode
}

// To returns the destination node.
func (e *CognitiveEdge) To() graph.Node {
	return e.toNode
}

// Weight returns the edge weight.
func (e *CognitiveEdge) Weight() float64 {
	return e.edgeWeight
}

// ReversedEdge returns a reversed edge (required by gonum).
func (e *CognitiveEdge) ReversedEdge() graph.Edge {
	return &CognitiveEdge{
		fromNode:   e.toNode,
		toNode:     e.fromNode,
		RelType:    e.RelType,
		edgeWeight: e.edgeWeight,
		Confidence: e.Confidence,
	}
}

// AssociationGraph is a weighted directed graph for cognitive associations.
type AssociationGraph struct {
	graph   *simple.WeightedDirectedGraph
	nodeMap map[[16]byte]*CognitiveNode
	nextID  int64
}

// NewAssociationGraph creates a new association graph.
func NewAssociationGraph() *AssociationGraph {
	return &AssociationGraph{
		graph:   simple.NewWeightedDirectedGraph(0, 0),
		nodeMap: make(map[[16]byte]*CognitiveNode),
		nextID:  1,
	}
}

// AddEngramNode adds a node representing an engram.
func (g *AssociationGraph) AddEngramNode(engramID [16]byte, concept string, memType uint8) int64 {
	if node, exists := g.nodeMap[engramID]; exists {
		return node.nodeID
	}

	node := &CognitiveNode{
		nodeID:   g.nextID,
		EngramID: engramID,
		Concept:  concept,
		MemType:  memType,
		Strength: 0.5,
	}

	g.graph.AddNode(node)
	g.nodeMap[engramID] = node
	g.nextID++

	return node.nodeID
}

// AddRelation adds a weighted relation between two engrams.
func (g *AssociationGraph) AddRelation(
	fromEngramID, toEngramID [16]byte,
	relType RelationType,
	weight, confidence float64,
) (*CognitiveEdge, error) {
	fromNode, ok1 := g.nodeMap[fromEngramID]
	toNode, ok2 := g.nodeMap[toEngramID]

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("gnn: node not found")
	}

	edge := &CognitiveEdge{
		fromNode:   fromNode,
		toNode:     toNode,
		RelType:    relType,
		edgeWeight: weight,
		Confidence: confidence,
	}

	g.graph.SetWeightedEdge(edge)
	return edge, nil
}

// StrengthenRelation increases the weight of an existing relation.
func (g *AssociationGraph) StrengthenRelation(fromID, toID [16]byte, delta float64) float64 {
	fromNode, ok1 := g.nodeMap[fromID]
	toNode, ok2 := g.nodeMap[toID]

	if !ok1 || !ok2 {
		return 0
	}

	edge := g.graph.Edge(fromNode.nodeID, toNode.nodeID)
	if edge == nil {
		return 0
	}

	if weightedEdge, ok := edge.(*CognitiveEdge); ok {
		newWeight := weightedEdge.edgeWeight + delta
		if newWeight > 1.0 {
			newWeight = 1.0
		}
		weightedEdge.edgeWeight = newWeight
		return newWeight
	}

	return 0
}

// GetRelations returns all relations from a given engram.
func (g *AssociationGraph) GetRelations(engramID [16]byte) []*CognitiveEdge {
	node, ok := g.nodeMap[engramID]
	if !ok {
		return nil
	}

	var edges []*CognitiveEdge
	successors := g.graph.From(node.nodeID)
	for successors.Next() {
		edge := g.graph.Edge(node.nodeID, successors.Node().ID())
		if weightedEdge, ok := edge.(*CognitiveEdge); ok {
			edges = append(edges, weightedEdge)
		}
	}

	return edges
}

// GetRelationsByType returns relations filtered by type.
func (g *AssociationGraph) GetRelationsByType(engramID [16]byte, relType RelationType) []*CognitiveEdge {
	all := g.GetRelations(engramID)
	var filtered []*CognitiveEdge

	for _, edge := range all {
		if edge.RelType == relType {
			filtered = append(filtered, edge)
		}
	}

	return filtered
}

// weightedNode holds a node with its traversal score.
type weightedNode struct {
	node  *CognitiveNode
	score float64
}

// TraverseWithWeights performs BFS with relation-type-aware weighting.
// Edges are prioritized based on: edge.Weight * relWeights[edge.RelType]
// Returns nodes sorted by effective weight (highest first).
func (g *AssociationGraph) TraverseWithWeights(
	startEngramID [16]byte,
	maxDepth int,
	relWeights RelationWeightMap,
) []*CognitiveNode {
	startNode, ok := g.nodeMap[startEngramID]
	if !ok {
		return nil
	}

	if relWeights == nil {
		relWeights = DefaultRelationWeights()
	}

	// Collect nodes with their effective weights
	var results []weightedNode
	visited := make(map[int64]bool)

	// BFS queue: {node, depth, cumulativeScore}
	type queueItem struct {
		node  *CognitiveNode
		depth int
		score float64
	}

	queue := []queueItem{{startNode, 0, 1.0}}
	visited[startNode.nodeID] = true

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item.depth > maxDepth {
			continue
		}

		// Add to results
		results = append(results, weightedNode{item.node, item.score})

		// Explore neighbors
		successors := g.graph.From(item.node.nodeID)
		for successors.Next() {
			nextNodeID := successors.Node().ID()
			if visited[nextNodeID] {
				continue
			}

			edge := g.graph.Edge(item.node.nodeID, nextNodeID)
			if cognitiveEdge, ok := edge.(*CognitiveEdge); ok {
				// Calculate effective weight
				relWeight := relWeights[cognitiveEdge.RelType]
				if relWeight == 0 {
					relWeight = 1.0 // Default weight
				}
				effectiveWeight := cognitiveEdge.edgeWeight * relWeight

				// Propagate score with decay
				newScore := item.score * effectiveWeight * 0.9 // 0.9 = depth decay

				if nextNode, exists := g.nodeMap[cognitiveEdge.toNode.EngramID]; exists {
					queue = append(queue, queueItem{nextNode, item.depth + 1, newScore})
					visited[nextNodeID] = true
				}
			}
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Extract nodes
	nodes := make([]*CognitiveNode, len(results))
	for i, wn := range results {
		nodes[i] = wn.node
	}

	return nodes
}

// Traverse maintains backward compatibility by calling TraverseWithWeights with defaults.
func (g *AssociationGraph) Traverse(startEngramID [16]byte, maxDepth int) []*CognitiveNode {
	return g.TraverseWithWeights(startEngramID, maxDepth, DefaultRelationWeights())
}

// GetNode returns a node by engram ID.
func (g *AssociationGraph) GetNode(engramID [16]byte) *CognitiveNode {
	return g.nodeMap[engramID]
}

// NodeCount returns the number of nodes.
func (g *AssociationGraph) NodeCount() int {
	return g.graph.Nodes().Len()
}

// EdgeCount returns the number of edges.
func (g *AssociationGraph) EdgeCount() int {
	return g.graph.Edges().Len()
}

// FindNodesByConcept searches for nodes by concept string.
func (g *AssociationGraph) FindNodesByConcept(concept string, maxResults int) []*CognitiveNode {
	var results []*CognitiveNode

	for _, node := range g.nodeMap {
		if strings.Contains(strings.ToLower(node.Concept), strings.ToLower(concept)) {
			results = append(results, node)
			if len(results) >= maxResults {
				break
			}
		}
	}

	return results
}

// RemoveNode removes a node and all its edges.
func (g *AssociationGraph) RemoveNode(engramID [16]byte) {
	node, ok := g.nodeMap[engramID]
	if !ok {
		return
	}

	g.graph.RemoveNode(node.nodeID)
	delete(g.nodeMap, engramID)
}
