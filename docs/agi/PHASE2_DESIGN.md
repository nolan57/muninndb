# MuninnDB Phase 2 Technical Design Document
## Cognitive Deepening: GNN Association Graph + Contextual Context

**Version**: v1.0
**Date**: 2026-03-29
**Status**: Under design review

---

## Overview

Phase 2 objectives: Implement deep associative networks and context-aware recall, enabling MuninnDB to:
1. Understand complex semantic relationships (causal/analogical/metaphorical)
2. Dynamically adjust recall results based on current context
3. Automatically induce and form knowledge prototypes

---

## Architecture Design

### 2.1 Neuro-Symbolic Association Graph

```
┌──────────────────────────────────────────────────────────┐
│              Neuro-Symbolic Association Graph            │
├──────────────────────────────────────────────────────────┤
│  Neural Layer (numeric, implicit):                       │
│    - Hebbian co-activation weights                       │
│    - Semantic similarity (ONNX embeddings)               │
│    - Activation spreading                                │
├──────────────────────────────────────────────────────────┤
│  Symbolic Layer (symbolic, explicit):                    │
│    - Causal relationships (A→B)                          │
│    - Analogical mappings (A:B::C:D)                      │
│    - Metaphorical links (A is B)                         │
│    - Temporal sequences (A before B)                     │
│    - Taxonomic hierarchy (A is subclass of B)            │
└──────────────────────────────────────────────────────────┘
```

**Technology Selection**:
- Graph engine: `gonum.org/v1/gonum/graph` (pure Go, no CGO)
- Storage: Pebble DB (existing)
- Traversal: BFS/DFS + heuristic pruning

---

## Module Design

### COG-21: GNN Graph Structure

**File**: `internal/index/gnn/graph.go`

```go
type AssociationGraph struct {
    graph    *simple.WeightedDirectedGraph
    nodeMap  map[[16]byte]*CognitiveNode
    nextID   int64
}

type CognitiveNode struct {
    nodeID    int64
    EngramID  [16]byte
    Concept   string
    MemType   uint8
    Strength  float64
}

type CognitiveEdge struct {
    fromNode   *CognitiveNode
    toNode     *CognitiveNode
    RelType    RelationType
    edgeWeight float64  // Hebbian weight
    Confidence float64  // relationship confidence
}
```

**Core Interface**:
```go
func (g *AssociationGraph) AddEngramNode(engramID, concept, memType) int64
func (g *AssociationGraph) AddRelation(from, to, relType, weight, confidence) (*Edge, error)
func (g *AssociationGraph) StrengthenRelation(from, to, delta) float64
func (g *AssociationGraph) Traverse(startID, maxDepth) []*CognitiveNode
func (g *AssociationGraph) GetRelationsByType(engramID, relType) []*Edge
```

**Performance Goals**:
- Node addition: <1ms
- 2-hop traversal: <50ms (p99)
- Edge update: <0.1ms

---

### COG-22: Relation Type System

**File**: `internal/index/gnn/relations.go` ✅ (Complete)

**Relation Types**:
| Type | Value | Directional | Symmetric | Example |
|------|-------|-------------|-----------|---------|
| `RelCoActivation` | 1 | No | ✅ | Co-activation |
| `RelCausal` | 2 | ✅ | ❌ | A causes B |
| `RelAnalogical` | 3 | No | ✅ | A:B::C:D |
| `RelMetaphorical` | 4 | No | ✅ | A is like B |
| `RelTemporal` | 5 | ✅ | ❌ | A before B |
| `RelPartonomic` | 6 | ✅ | ❌ | A is part of B |
| `RelTaxonomic` | 7 | ✅ | ❌ | A is subclass of B |
| `RelSupports` | 8 | ✅ | ❌ | A supports B |
| `RelContradicts` | 9 | ✅ | ❌ | A contradicts B |

**Relation Inference Rules**:
```go
// Causal inference: based on temporal order + co-occurrence frequency
func InferCausal(rel1, rel2 *Edge) *Edge {
    if rel1.Timestamp < rel2.Timestamp &&
       rel1.CooccurrenceCount > threshold {
        return NewEdge(RelCausal, confidence=0.7)
    }
    return nil
}

// Analogical inference: based on structure mapping theory
func InferAnalogical(pair1, pair2 EdgePair) *Edge {
    if structuralSimilarity(pair1, pair2) > 0.8 {
        return NewEdge(RelAnalogical, confidence=0.8)
    }
    return nil
}
```

**Validation Rules**:
- Detect circular causality (A→B→C→A)
- Detect contradictory relations (A supports B and A contradicts B)
- Detect hierarchy conflicts (A is subclass of B and B is subclass of A)

---

### COG-23: Contextual Context Schema

**File**: `internal/engine/context_schema.go`

```go
type ActivationContext struct {
    // Task context
    Task        string           // Current task description
    Goals       []string         // Goal list
    SubGoalOf   *ULID            // Parent goal (supports hierarchy)

    // Cognitive state
    CognitiveLoad float64        // 0-1: Cognitive load
    TimePressure  float64        // 0-1: Time pressure
    WorkingMemory []ULID         // Current working memory content

    // Emotional state (optional)
    EmotionalState map[string]float64  // {"anxiety": 0.7, "curiosity": 0.3}

    // Environment context
    Environment   map[string]string     // {"location": "office", "device": "mobile"}

    // Historical activations (for sequence patterns)
    RecentActivations []ULID

    // Time context
    TimeOfDay     time.Time
    SessionStart  time.Time
}

// Serialization support
func (ctx *ActivationContext) Marshal() ([]byte, error)
func (ctx *ActivationContext) Unmarshal(data []byte) error
```

**Context Extraction**:
```go
// Extract context from request
func ExtractContext(r *http.Request) (*ActivationContext, error) {
    var req struct {
        Context   []string `json:"context"`
        Task      string   `json:"task"`
        Goals     []string `json:"goals"`
        Timestamp int64    `json:"timestamp"`
    }

    ctx := &ActivationContext{
        Task:        req.Task,
        Goals:       req.Goals,
        TimeOfDay:   time.Unix(req.Timestamp, 0),
        CognitiveLoad: 0.5, // default
    }

    // Extract semantics from context array
    if len(req.Context) > 0 {
        ctx.WorkingMemory = resolveToULIDs(req.Context)
    }

    return ctx, nil
}
```

---

### COG-24: Dynamic Weighting Calculation

**File**: `internal/scoring/contextual_weighting.go`

```go
// Multi-factor scoring formula
func ContextualScore(engram *Engram, ctx *ActivationContext) float64 {
    // Base score (ACT-R + Hebbian)
    base := scoring.BaseLevelActivation(engram.AccessCount, engram.LastAccess)
    hebbian := scoring.HebbianBoost(engram.CoActivations, ctx.RecentActivations)

    // Context modulation factors
    taskRelevance := SemanticSimilarity(engram.Content, ctx.Task)
    goalAlignment := AlignsWithGoals(engram, ctx.Goals)
    emotionalCongruence := MatchEmotionalState(engram, ctx.EmotionalState)

    // Cognitive load modulation (high load → reduce complexity)
    complexityPenalty := ctx.CognitiveLoad * engram.ComplexityScore

    // Time pressure modulation (high pressure → prefer high confidence)
    confidenceBonus := ctx.TimePressure * engram.Confidence

    return base + hebbian +
           0.3*taskRelevance +
           0.2*goalAlignment +
           0.1*emotionalCongruence -
           0.1*complexityPenalty +
           0.2*confidenceBonus
}
```

**Sub-modules**:

#### Goal Alignment
```go
func AlignsWithGoals(engram *Engram, goals []string) float64 {
    maxAlign := 0.0
    for _, goal := range goals {
        align := SemanticSimilarity(engram.Content, goal)
        if align > maxAlign {
            maxAlign = align
        }
    }
    return maxAlign
}
```

#### Emotional Congruence
```go
func MatchEmotionalState(engram *Engram, state map[string]float64) float64 {
    // Extract emotional polarity from engram
    engramSentiment := AnalyzeSentiment(engram.Content)

    // Match with current emotional state
    matchScore := 0.0
    for emotion, intensity := range state {
        if emotionMatches(engramSentiment, emotion) {
            matchScore += intensity * 0.5 // Emotional consistency reward
        }
    }
    return math.Min(matchScore, 1.0)
}
```

---

## Integration Design

### ACTIVATE Flow Enhancement

```
Original flow:
1. Full-text search + Vector search → Fusion
2. Hebbian boost
3. ACT-R time weighting
4. Return results

Enhanced flow:
1. Extract ActivationContext
2. Full-text search + Vector search → Fusion
3. GNN graph traversal (2-hop, relation filtering)
4. Contextual weight calculation (ContextualScore)
5. Cognitive load modulation
6. Return results + Explanation (optional)
```

**API Changes**:
```json
// POST /api/activate
{
  "context": ["debug payment"],
  "task": "fix retry logic bug",
  "goals": ["prevent double-charge", "maintain idempotency"],
  "cognitive_load": 0.7,
  "time_pressure": 0.5,
  "max_results": 10,
  "explain": true  // New: return score explanation
}

// Response
{
  "activations": [
    {
      "id": "...",
      "concept": "idempotency key conflict",
      "score": 0.89,
      "score_breakdown": {  // New
        "base": 0.5,
        "hebbian": 0.2,
        "task_relevance": 0.15,
        "goal_alignment": 0.18,
        "complexity_penalty": -0.07,
        "confidence_bonus": 0.1
      },
      "relations": [  // New
        {"type": "causal", "to": "payment failure", "weight": 0.8}
      ]
    }
  ]
}
```

---

## Test Plan

### GNN Graph Traversal Test
```go
func TestGNN_Traversal(t *testing.T) {
    graph := gnn.NewAssociationGraph()

    // Create test graph
    // A --causal--> B --temporal--> C
    // A --analogical--> D

    nodes := graph.Traverse(A, 2)
    // Verify: should return [A, B, C, D]
}
```

### Context-Aware Test
```go
func TestContextualRecall_TaskDependent(t *testing.T) {
    ctx1 := &ActivationContext{Task: "debug payment"}
    ctx2 := &ActivationContext{Task: "design feature"}

    result1 := engine.Activate(ctx1, "payment system", 10)
    result2 := engine.Activate(ctx2, "payment system", 10)

    // Verify: rankings should differ under different tasks
    if rankingsAreIdentical(result1, result2) {
        t.Error("Contextual recall failed")
    }
}
```

### Relation Inference Test
```go
func TestRelationInference_Causal(t *testing.T) {
    // A→B (temporal order, high-frequency co-occurrence)
    // Verify: should infer RelCausal
}
```

---

## Performance Optimization Strategies

### 1. Graph Partitioning
- Partition by vault
- Partition by topic domain (payment, auth, deployment...)
- Lazy loading (load only 2-hop neighbors)

### 2. Caching
- Hot node cache (LRU)
- Traversal result cache (TTL: 5 minutes)
- Context-result pair cache

### 3. Parallelization
- Multi-threaded graph traversal
- Batch relation inference
- Asynchronous contextual weight calculation

---

## Milestones

| Milestone | Date | Deliverables |
|-----------|------|-------------|
| M1: GNN Foundation | Day 1 | graph.go + relations.go |
| M2: Relation Inference | Day 2 | inference.go + validation |
| M3: Context Schema | Day 2 | context_schema.go |
| M4: Dynamic Weighting | Day 3 | contextual_weighting.go |
| M5: Integration Tests | Day 3 | e2e test suite |
| M6: Documentation | Day 3 | API docs + usage guide |

---

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| gonum performance insufficient | High | Preload + cache; fallback: custom graph structure |
| Low relation inference accuracy | Medium | Manual review mode; confidence threshold |
| High context calculation latency | Medium | Asynchronous calculation; degradation strategy |
| Graph scale explosion | High | Partitioning + pruning; TTL auto-cleanup |

---

## Next Actions

1. ✅ Complete relation type definition (`relations.go`)
2. ⏳ Fix GNN graph structure (`graph.go`)
3. ⏳ Implement context schema (`context_schema.go`)
4. ⏳ Implement dynamic weighting (`contextual_weighting.go`)
5. ⏳ Write integration tests
6. ⏳ Update API documentation

---

*Document Version: v1.0*
*Creation Date: 2026-03-29*
*Review Status: Pending review*
