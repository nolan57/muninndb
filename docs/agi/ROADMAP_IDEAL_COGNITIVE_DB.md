# MuninnDB Ideal Cognitive Database Technical Roadmap (18 Months)

> **Vision**: Upgrade MuninnDB from "a database with cognitive primitives" to "a fully ideal cognitive database" — a storage system capable of simulating core mechanisms of human memory.

---

## Definition of an Ideal Cognitive Database

A storage system capable of simulating core mechanisms of human memory, with the following capabilities:

| Capability | Description | Corresponding Human Brain Mechanism |
|------------|-------------|-------------------------------------|
| **Multi-scale Memory Hierarchy** | Distinguish sensory memory, working memory, episodic memory, semantic memory | Atkinson-Shiffrin memory model |
| **Dynamic Forgetting & Consolidation** | Automatically adjust memory strength based on usage frequency, emotional weight, contextual importance | Ebbinghaus forgetting curve + synaptic plasticity |
| **Deep Associative Networks** | Support cross-modal associations with complex relationships like causal, analogical, metaphorical | Hebbian learning + semantic network theory |
| **Context-Aware Recall** | Recall results dynamically adjust based on current task, emotion, environment | ACT-R contextual activation theory |
| **Memory Generalization & Prototype Formation** | Automatically generalize similar experiences, generate reusable knowledge prototypes | Prototype theory + inductive learning |
| **Metacognition** | Evaluate own knowledge completeness, confidence, and blind spots | Flavell metacognition theory |

---

## Constraints

| Constraint | Implementation Plan |
|------------|---------------------|
| **Protocol Compatibility** | Keep existing MCP/REST/gRPC/MBP protocols unchanged; all new features implemented via extension fields for backward compatibility |
| **Technology Stack** | Prioritize Go ecosystem; ML components introduce only ONNX Runtime (existing) + lightweight GNN (gonum) |
| **License** | Comply with BSL 1.1; do not replicate patented core logic (existing patents: Ebbinghaus decay, Hebbian learning, Bayesian confidence, semantic triggers) |
| **Deployment** | Maintain single-binary deployment capability, support local execution; cluster features as optional extensions |

---

# Phase 1: Foundation Enhancements (0–6 Months)

> **Objective**: Extend engram model to support multi-scale memory hierarchy, implement working memory buffer and dynamic consolidation mechanisms.

## 1.1 Memory Type Hierarchy System

### 1.1.1 Extend Engram Model

**Existing Fields** (`internal/storage/engram.go`):
```go
type Engram struct {
    ID          ULID
    Concept     string
    Content     string
    Tags        []string
    Confidence  float64
    Stability   float64
    Relevance   float64
    MemoryType  MemoryType  // ← Existing but underutilized
    // ...
}
```

**New/Modified Modules**:

| File | Responsibility | Key Interfaces |
|------|----------------|----------------|
| `internal/engine/memory_types.go` | Define 4 memory types and transformation rules | `type MemoryType uint8` |
| `internal/storage/engram_metadata.go` | Extend metadata to support memory type tags | `UpdateMemoryType(id, type)` |
| `internal/engine/working_memory.go` | **New**: Working memory buffer management | `PushToWM(), FlushFromWM()` |
| `internal/engine/consolidation_scheduler.go` | **New**: Memory consolidation scheduler | `ScheduleConsolidation(engramID)` |

**MemoryType Enum Extension**:
```go
type MemoryType uint8

const (
    MemoryTypeSensory    MemoryType = 0  // Sensory memory: <100ms temporary storage
    MemoryTypeWorking    MemoryType = 1  // Working memory: seconds to minutes
    MemoryTypeEpisodic   MemoryType = 2  // Episodic memory: specific events
    MemoryTypeSemantic   MemoryType = 3  // Semantic memory: abstract knowledge
    MemoryTypeProcedural MemoryType = 4  // Procedural memory: skills/procedures (reserved)
)
```

### 1.1.2 Working Memory Buffer Design

**Architecture**:
```
┌─────────────────────────────────────────────────────────┐
│                   Working Memory Buffer                 │
│  Capacity: 7±2 items (Miller's Law)                     │
│  Decay: 15-30 seconds without rehearsal                 │
└─────────────────────────────────────────────────────────┘
         ↑                       ↓
   PushToWM()             FlushToLTM() (consolidation)
         ↑                       ↓
┌─────────────────┐     ┌─────────────────────────┐
│  Input Stream   │     │  Long-Term Memory       │
│  (REST/MCP)     │     │  (Pebble Store)         │
└─────────────────┘     └─────────────────────────┘
```

**Interface Definition** (`internal/engine/working_memory.go`):
```go
type WorkingMemoryBuffer struct {
    items      []*Engram
    maxSize    int           // Default 7
    decayTime  time.Duration // Default 30s
    mu         sync.RWMutex
}

func NewWorkingMemoryBuffer(config WMConfig) *WorkingMemoryBuffer
func (wm *WorkingMemoryBuffer) Push(engram *Engram) error
func (wm *WorkingMemoryBuffer) Flush(consolidate bool) []*Engram
func (wm *WorkingMemoryBuffer) Rehearse(id ULID) // Rehearsal to extend retention
func (wm *WorkingMemoryBuffer) GetActive() []*Engram
```

**Acceptance Criteria**:
- [ ] Working memory capacity configurable (default 7±2)
- [ ] Automatic decay after 30 seconds without rehearsal
- [ ] Support manual rehearsal to extend retention time
- [ ] Preserve association edges when consolidating to long-term memory

### 1.1.3 Memory Consolidation Trigger Mechanism

**Consolidation Conditions**:
1. **Repeated Trigger**: Rehearsed ≥3 times in working memory
2. **Emotional Weight**: High emotional intensity detected (via LLM emotion analysis)
3. **Contextual Importance**: Highly relevant to current task (ACT-R activation > threshold)

**Scheduler Design** (`internal/engine/consolidation_scheduler.go`):
```go
type ConsolidationScheduler struct {
    queue    chan ConsolidationJob
    workers  int
    rules    []ConsolidationRule
}

type ConsolidationRule struct {
    Name        string
    Condition   func(*Engram, *ActivityContext) bool
    Priority    int
    Action      func(ULID) error
}

// Built-in rules
func DefaultConsolidationRules() []ConsolidationRule {
    return []ConsolidationRule{
        {Name: "rehearsal", Condition: checkRehearsalCount, Priority: 1},
        {Name: "emotional_salience", Condition: checkEmotionalWeight, Priority: 2},
        {Name: "contextual_relevance", Condition: checkACTRActivation, Priority: 3},
    }
}
```

**API Extension** (Backward Compatible):
```json
// POST /api/engrams
{
  "concept": "...",
  "content": "...",
  "memory_type": "working",  // New field, optional
  "metadata": {
    "emotional_weight": 0.8  // New field, optional
  }
}
```

## 1.2 Dynamic Forgetting & Strength Modulation

### 1.2.1 Extend Ebbinghaus Decay Model

**Existing Implementation** (`internal/scoring/temporal.go`):
```go
// Currently based only on ACT-R base activation
func BaseLevelActivation(n int, ageDays float64) float64 {
    return math.Log(float64(n+1)) - 0.5*math.Log(ageDays/float64(n+1))
}
```

**Enhancement Plan**:
```go
// New: Multi-factor strength modulation
func DynamicForgetting(engram *Engram, ctx *AccessContext) float64 {
    base := BaseLevelActivation(engram.AccessCount, engram.AgeDays())

    // Emotional weight modulation (high emotion → slower decay)
    emotionalMod := 1.0 - (engram.EmotionalWeight * 0.3)

    // Sleep consolidation simulation (decay paused during offline periods)
    sleepMod := ctx.DuringSleep ? 0.1 : 1.0

    // Spaced repetition reward
    spacingBonus := calculateSpacingBonus(engram.AccessHistory)

    return base * emotionalMod * sleepMod * spacingBonus
}
```

**New Modules**:
| File | Responsibility |
|------|----------------|
| `internal/scoring/emotional_weight.go` | Emotional weight calculation (via LLM or rules) |
| `internal/scoring/spacing_effect.go` | Spaced repetition reward calculation |
| `internal/engine/sleep_scheduler.go` | Simulate sleep consolidation cycles |

### 1.2.2 Automatic Memory Strength Modulation

**Strength Levels** (5 levels):
```go
type MemoryStrength uint8

const (
    StrengthFragile   MemoryStrength = 0  // Fragile: newly encoded, unconsolidated
    StrengthLabile    MemoryStrength = 1  // Labile: requires rehearsal
    StrengthStable    MemoryStrength = 2  // Stable: normal forgetting curve
    StrengthRobust    MemoryStrength = 3  // Robust: interference-resistant
    StrengthPermanent MemoryStrength = 4  // Permanent: almost never forgotten
)
```

**Automatic Modulation Logic** (`internal/engine/strength_adaptor.go`):
```go
func AdaptStrength(engram *Engram, history []AccessEvent) MemoryStrength {
    // Calculate based on:
    // 1. Access count
    // 2. Access interval distribution
    // 3. Consolidation state
    // 4. Number of association edges

    if engram.Consolidated && len(engram.Associations) > 10 {
        return StrengthRobust
    }
    // ...
}
```

## 1.3 Engineering Safeguards (Phase 1)

### 1.3.1 Compatibility Maintenance Strategy

| Change Type | Compatibility Plan | Acceptance Test |
|-------------|-------------------|-----------------|
| New `memory_type` field | Default to `MemoryTypeSemantic`, old clients unaware | Backward compatibility test suite |
| Extend MCP tools | Add `muninn_remember_enhanced`, keep original `muninn_remember` | MCP protocol regression test |
| Storage format change | ERF format versioning (`erf/v2`), support automatic migration | Upgrade/downgrade test |

### 1.3.2 Performance Control

| Scenario | Latency Target | Optimization Strategy |
|----------|----------------|----------------------|
| Working memory write | <5ms | In-memory operation, no persistence |
| Working memory → long-term memory consolidation | <50ms | Asynchronous batch processing |
| Dynamic forgetting calculation | <1ms/engram | Cache precomputed results |

### 1.3.3 Test Validation Plan

**Cognitive Behavior Simulation Test Suite** (`internal/engine/cognitive_sim_test.go`):
```go
// Test case example: validate spacing effect
func TestSpacingEffect_Consolidation(t *testing.T) {
    eng := createTestEngram()

    // Scenario A: Massed learning (5 times/1 hour)
    massed := simulateMassedPractice(eng, 5, time.Hour)

    // Scenario B: Spaced learning (5 times/5 days)
    spaced := simulateSpacedPractice(eng, 5, 5*24*time.Hour)

    // Verify: spaced learning should maintain higher strength
    if spaced.Strength <= massed.Strength {
        t.Errorf("Spacing effect violated")
    }
}
```

**Phase 1 Acceptance Criteria**:
- [ ] Working memory buffer passes Miller's Law validation (7±2 capacity)
- [ ] Dynamic forgetting curve fits Ebbinghaus model (R² > 0.85)
- [ ] Spaced repetition effect passes behavioral simulation test
- [ ] MCP protocol backward compatible (old clients work normally)
- [ ] Working memory write latency <5ms (p99)

---

# Phase 2: Cognitive Deepening (6–12 Months)

> **Objective**: Implement deep associative networks and context-aware recall, supporting complex relational reasoning and dynamic weight calculation.

## 2.1 Associative Network Upgrade

### 2.1.1 Neuro-Symbolic Graph Representation

**Current State**: Current associations only support simple co-activation edges (`internal/storage/association.go`)

**Target Architecture**:
```
┌──────────────────────────────────────────────────────────┐
│              Neuro-Symbolic Association Graph            │
├──────────────────────────────────────────────────────────┤
│  Neural Layer (existing):                                 │
│    - Hebbian co-activation weights (numeric, implicit)    │
│    - Semantic similarity (ONNX embeddings)                │
├──────────────────────────────────────────────────────────┤
│  Symbolic Layer (new):                                    │
│    - Causal relationships (causal: A→B)                   │
│    - Analogical mappings (analogical: A:B::C:D)           │
│    - Metaphorical links (metaphorical: A is B)            │
│    - Temporal sequences (temporal: A before B)            │
└──────────────────────────────────────────────────────────┘
```

**New Modules**:
| File | Responsibility | Dependencies |
|------|----------------|--------------|
| `internal/index/gnn/graph.go` | **New**: GNN graph structure definition | gonum/graph |
| `internal/index/gnn/edge_types.go` | **New**: Relation type enum | - |
| `internal/engine/association_inference.go` | **New**: Infer relation types from text | Local LLM (optional) |
| `internal/scoring/gnn_scoring.go` | **New**: GNN-assisted association scoring | gonum/optimize |

**Relation Type Definition** (`internal/index/gnn/edge_types.go`):
```go
type RelationType string

const (
    RelCoActivation   RelationType = "co_activation"   // Co-activation (existing Hebbian)
    RelCausal         RelationType = "causal"          // A causes B
    RelAnalogical     RelationType = "analogical"      // A:B::C:D
    RelMetaphorical   RelationType = "metaphorical"    // A is like B
    RelTemporal       RelationType = "temporal"        // A before B
    RelPartonomic     RelationType = "partonomic"      // A is part of B
    RelTaxonomic      RelationType = "taxonomic"       // A is subclass of B
)

type SymbolicEdge struct {
    Source    ULID
    Target    ULID
    Type      RelationType
    Confidence float64
    Evidence  []string  // Text fragments supporting this relation
}
```

### 2.1.2 Relation Inference Pipeline

**Automatic Extraction Flow**:
```
Engram Content
      ↓
┌─────────────────┐
│  LLM/Rule-Based │ ← Local LLM (Ollama) or rule engine
│  Relation Extractor │
└─────────────────┘
      ↓
┌─────────────────┐
│  Validation Layer  │ ← Check logical consistency (no circular causality, etc.)
└─────────────────┘
      ↓
┌─────────────────┐
│  Graph Insertion   │ ← Write to GNN index
└─────────────────┘
```

**Interface Definition** (`internal/engine/association_inference.go`):
```go
type RelationInferrer struct {
    llmClient  *ollama.Client  // Optional
    ruleEngine *RuleEngine
}

func (ri *RelationInferrer) ExtractRelations(content string, context []ULID) []SymbolicEdge
func (ri *RelationInferrer) ValidateGraph(edges []SymbolicEdge) error  // Detect circular, etc.
```

## 2.2 Contextual Context Schema

### 2.2.1 Context Representation Model

**Design Objective**: Recall results dynamically adjust based on current task, emotion, environment

**Context Schema** (`internal/engine/context_schema.go`):
```go
type ActivationContext struct {
    // Task context
    Task        string           // Current task description
    Goals       []string         // Goal list
    SubGoalOf   *ULID            // Parent goal (supports hierarchy)

    // Cognitive state
    CognitiveLoad float64        // 0-1: Cognitive load (high → return fewer but more relevant)
    TimePressure  float64        // 0-1: Time pressure (high → skip deep reasoning)

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
```

### 2.2.2 Dynamic Weighting Calculation Model

**Multi-Factor Scoring Formula**:
```go
func ContextualScore(engram *Engram, ctx *ActivationContext) float64 {
    // Base score (ACT-R + Hebbian)
    base := scoring.BaseLevelActivation(engram.AccessCount, engram.AgeDays())
    hebbian := scoring.HebbianBoost(engram.CoActivations, ctx.RecentActivations)

    // Context modulation factors
    taskRelevance := semanticSimilarity(engram.Content, ctx.Task)
    goalAlignment := alignsWithGoals(engram, ctx.Goals)
    emotionalCongruence := matchEmotionalState(engram, ctx.EmotionalState)

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
| File | Responsibility |
|------|----------------|
| `internal/engine/context_schema.go` | Context schema definition and validation |
| `internal/scoring/contextual_weighting.go` | Contextual weight calculation |
| `internal/engine/goal_tracker.go` | Goal hierarchy tracking and management |
| `internal/plugin/emotion_analyzer.go` | Emotional state analysis (optional plugin) |

## 2.3 Memory Generalization Pipeline

### 2.3.1 Prototype Formation Architecture

**Flow**:
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Clustering  │ →   │  LLM Summary │ →   │  Prototype   │
│  (similar    │     │  (extract    │     │  Storage     │
│  experiences)│     │  commonalities)│   │              │
└──────────────┘     └──────────────┘     └──────────────┘
       ↑                                        ↓
       └────────────────────────────────────────┘
                  New experience match → Update prototype
```

### 2.3.2 Implementation Plan

**Phase 1: Clustering** (`internal/engine/prototype_clusterer.go`):
```go
type PrototypeClusterer struct {
    embedder  Embedder
    algorithm string  // "kmeans", "dbscan", "hierarchical"
}

func (pc *PrototypeClusterer) FindClusters(engrams []*Engram, vault string) []Cluster {
    // 1. Embed all engrams
    // 2. Run clustering algorithm
    // 3. Return clustering results (each cluster contains member IDs + centroid vector)
}
```

**Phase 2: LLM Summary** (`internal/engine/prototype_summarizer.go`):
```go
type PrototypeSummarizer struct {
    llmClient *ollama.Client  // Optional
}

func (ps *PrototypeSummarizer) GeneratePrototype(cluster Cluster) *Prototype {
    // Input: Content of all engrams in cluster
    // Output: Abstracted prototype description
    // Example:
    //   Input: [Payment failure case 1, Payment failure case 2, ...]
    //   Output: "Payment system failures are typically caused by idempotency key conflicts..."
}
```

**Phase 3: Prototype Storage** (`internal/storage/prototype.go`):
```go
type Prototype struct {
    ID          ULID
    Concept     string          // Prototype name
    Description string          // Abstract description
    MemberCount int             // Number of experiences supporting this prototype
    Members     []ULID          // Member engram IDs
    Embedding   []float32       // Prototype embedding (cluster centroid)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Prototypes also stored in Pebble, supporting retrieval
func (ps *PebbleStore) UpsertPrototype(proto *Prototype) error
func (ps *PebbleStore) FindPrototypes(query []float32, k int) ([]*Prototype, error)
```

### 2.3.3 Prototype Matching & Update

**Matching Logic**:
```go
func (pc *PrototypeClusterer) MatchToPrototype(engram *Engram) (*Prototype, float64) {
    // Calculate similarity between engram and all prototypes
    // Return best match + similarity score
    // If score < threshold, trigger new prototype formation
}
```

**Update Rules**:
- New member joins → Update prototype embedding (moving average)
- Member count ≥10 → Trigger LLM re-summary
- Prototype similarity >0.9 → Merge prototypes

## 2.4 Engineering Safeguards (Phase 2)

### 2.4.1 GNN Performance Optimization

| Optimization | Implementation Plan | Expected Benefit |
|--------------|---------------------|------------------|
| Graph partitioning | Partition by vault or topic domain | Reduce traversal scope |
| Lazy loading | Load only 2-hop neighbors | Reduce memory usage |
| Parallel traversal | Use goroutines for parallel exploration | 3-5x speedup |

### 2.4.2 LLM Dependency Management

| Scenario | Plan | Degradation Strategy |
|----------|------|---------------------|
| Relation extraction | Local Ollama (default) | Fallback to rule engine |
| Prototype summary | Local Ollama (default) | Fallback to template summary |
| Emotion analysis | Plugin (optional) | Skip emotional weight |

### 2.4.3 Test Validation

**Context-Aware Test** (`internal/engine/contextual_recall_test.go`):
```go
func TestContextualRecall_TaskDependent(t *testing.T) {
    ctx1 := &ActivationContext{Task: "debug payment"}
    ctx2 := &ActivationContext{Task: "design new feature"}

    result1 := engine.Activate(ctx1, "payment system", 10)
    result2 := engine.Activate(ctx2, "payment system", 10)

    // Verify: recall result rankings should differ under different tasks
    if resultsAreIdentical(result1, result2) {
        t.Error("Contextual recall failed: results should differ by task")
    }
}
```

**Phase 2 Acceptance Criteria**:
- [ ] Support ≥5 relation types (causal, analogical, metaphorical, temporal, taxonomic)
- [ ] Relation extraction accuracy >70% (compared to manual annotation)
- [ ] Context-aware recall passes task-dependent test
- [ ] Prototype formation pipeline end-to-end latency <500ms
- [ ] GNN graph traversal (2-hop) <50ms (p99)

---

# Phase 3: Metacognition & AGI Readiness (12–18 Months)

> **Objective**: Implement metacognitive monitoring and self-diagnosis capabilities, deep integration with mainstream Agent frameworks.

## 3.1 Metacognitive Monitoring Metrics

### 3.1.1 Knowledge Coverage Assessment

**Definition**: Degree of current knowledge base coverage over query domain

**Calculation Method** (`internal/metacognition/coverage.go`):
```go
type CoverageAnalyzer struct {
    entityIndex *EntityIndex
    topicModel  *TopicModel
}

func (ca *CoverageAnalyzer) CalculateCoverage(query string, vault string) *CoverageReport {
    // 1. Extract entities and topics from query
    entities := extractEntities(query)
    topics := inferTopics(query)

    // 2. Calculate entity coverage (known entities / total entities)
    entityCoverage := ca.countKnownEntities(entities, vault) / len(entities)

    // 3. Calculate topic coverage (vector space density)
    topicDensity := ca.calculateTopicDensity(topics, vault)

    // 4. Overall score
    return &CoverageReport{
        EntityCoverage: entityCoverage,
        TopicDensity:   topicDensity,
        OverallScore:   0.6*entityCoverage + 0.4*topicDensity,
        BlindSpots:     ca.identifyBlindSpots(entities, topics, vault),
    }
}
```

**Output Example**:
```json
{
  "query": "payment system architecture",
  "coverage": {
    "entity_coverage": 0.85,
    "topic_density": 0.72,
    "overall_score": 0.80,
    "blind_spots": ["PCI compliance", "fraud detection"],
    "recommendations": [
      "Add memories about PCI DSS requirements",
      "Document fraud detection workflow"
    ]
  }
}
```

### 3.1.2 Confidence Entropy Calculation

**Definition**: Degree of consistency in knowledge base (high entropy = many contradictions/low confidence)

**Calculation Formula**:
```go
func CalculateConfidenceEntropy(vault string) float64 {
    // 1. Get confidence distribution of all engrams
    confidences := getConfidenceDistribution(vault)

    // 2. Calculate Shannon entropy
    // H = -Σ p(x) × log2(p(x))
    // where x is confidence interval (0-0.2, 0.2-0.4, ...)

    // 3. Normalize to 0-1 (1 = highest uncertainty)
    return normalizeEntropy(shannonEntropy(confidences))
}

// Contradiction detection enhancement
func DetectContradictionHotspots(vault string) []ContradictionCluster {
    // Identify engram groups with confidence <0.5 and contradicts relations
}
```

### 3.1.3 Metacognitive Metrics System

| Metric | Calculation Method | Threshold | Alert |
|--------|-------------------|-----------|-------|
| **Knowledge Coverage** | Entity coverage×0.6 + Topic density×0.4 | <0.5 | Recommend knowledge supplementation |
| **Confidence Entropy** | Shannon entropy normalized | >0.7 | Many contradictions exist |
| **Memory Freshness** | Percentage of engrams accessed in last 7 days | <0.2 | Knowledge may be outdated |
| **Association Density** | Average association edges per engram | <2 | Weak associative ability |
| **Prototype Coverage** | Percentage of clusters with prototypes | <0.3 | Weak generalization ability |

## 3.2 Self-Diagnosis API

### 3.2.1 API Design

**Endpoints**:
```
GET  /api/metacognition/coverage?query={query}&vault={vault}
GET  /api/metacognition/entropy?vault={vault}
GET  /api/metacognition/health?vault={vault}
POST /api/metacognition/diagnose  # Comprehensive diagnosis
```

**Response Example** (`GET /api/metacognition/health`):
```json
{
  "vault": "default",
  "timestamp": "2026-09-15T10:30:00Z",
  "overall_health": 0.78,
  "metrics": {
    "coverage": 0.82,
    "confidence_entropy": 0.35,
    "freshness": 0.65,
    "association_density": 3.2,
    "prototype_coverage": 0.45
  },
  "alerts": [
    {
      "type": "low_freshness",
      "severity": "warning",
      "message": "65% of memories haven't been accessed in 30+ days",
      "recommendation": "Review and prune outdated memories"
    }
  ],
  "blind_spots": [
    "authentication",
    "deployment"
  ]
}
```

### 3.2.2 Implementation Modules

| File | Responsibility |
|------|----------------|
| `internal/metacognition/coverage.go` | Coverage analysis |
| `internal/metacognition/entropy.go` | Confidence entropy calculation |
| `internal/metacognition/health.go` | Comprehensive health assessment |
| `internal/transport/rest/metacognition.go` | REST API handler |
| `internal/transport/grpc/metacognition.proto` | gRPC service definition |

## 3.3 Deep Integration with Agent Frameworks

### 3.3.1 LangGraph Integration Pattern

**Architecture**:
```
┌─────────────────┐     ┌─────────────────┐
│  LangGraph Node │ ←→  │  MuninnDB       │
│  (Agent State)  │     │  (Memory)       │
└─────────────────┘     └─────────────────┘
       ↓                         ↓
┌─────────────────┐     ┌─────────────────┐
│  Memory State   │ ←→  │  Working Memory │
│  (checkpoint)   │     │  Buffer         │
└─────────────────┘     └─────────────────┘
```

**Integration Package** (`sdk/python/langchain_muninn/`):
```python
from langgraph.graph import StateGraph, END
from muninn.langchain import MuninnDBMemory

# Create Muninn-backed state graph
memory = MuninnDBMemory(vault="agent-workspace")

workflow = StateGraph(AgentState)
workflow.add_node("agent", create_agent_node(memory))
workflow.add_node("memory_write", memory.write_node())
workflow.add_node("memory_read", memory.read_node())

# Define edges
workflow.add_edge("agent", "memory_write")
workflow.add_edge("memory_write", END)

app = workflow.compile()
```

**Key Interfaces**:
```python
class MuninnDBMemory:
    def write_node(self) -> Callable:
        """Return LangGraph node function, automatically write to working memory"""

    def read_node(self) -> Callable:
        """Return LangGraph node function, read relevant memories based on context"""

    async def checkpoint(self, state: AgentState):
        """Store Agent state snapshot as episodic memory"""

    async def restore(self, checkpoint_id: str) -> AgentState:
        """Restore Agent state from episodic memory"""
```

### 3.3.2 AutoGen Integration Pattern

**Architecture**:
```python
from autogen import ConversableAgent
from muninn.autogen import MuninnDBConversableAgent

# Extend AutoGen Agent to support MuninnDB memory
agent = MuninnDBConversableAgent(
    name="assistant",
    llm_config={"config_list": [...]},
    muninn_config={
        "vault": "autogen-agent",
        "working_memory_capacity": 7,
        "consolidation_enabled": True,
    }
)

# Automatic memory management
# - Every conversation turn automatically writes to working memory
# - Automatically consolidate to long-term memory when capacity reached
# - Automatically activate relevant memories when responding
```

**Integration Package** (`sdk/python/autogen_muninn/`):
```python
class MuninnDBConversableAgent(ConversableAgent):
    def __init__(self, *, muninn_config: dict, **kwargs):
        super().__init__(**kwargs)
        self.memory_client = MuninnClient(...)
        self.working_memory = WorkingMemoryBuffer(...)

    async def a_generate_reply(self, messages, **kwargs):
        # 1. Activate relevant context from long-term memory
        context = await self._activate_relevant_memory(messages)

        # 2. Enhance prompt with memory context
        enhanced_messages = self._inject_memory_context(messages, context)

        # 3. Generate response
        response = await super().a_generate_reply(enhanced_messages, **kwargs)

        # 4. Write to working memory
        await self.working_memory.push(Engram(...))

        return response
```

### 3.3.3 MCP Protocol Extension

**New MCP Tools**:
| Tool | Purpose | Parameters |
|------|---------|------------|
| `muninn_metacognition_health` | Get knowledge base health report | `vault` |
| `muninn_metacognition_coverage` | Query coverage for specific topic | `vault`, `query` |
| `muninn_checkpoint_create` | Create Agent state checkpoint | `vault`, `state`, `metadata` |
| `muninn_checkpoint_restore` | Restore Agent state checkpoint | `checkpoint_id` |
| `muninn_working_memory_push` | Write to working memory | `vault`, `engram` |
| `muninn_working_memory_flush` | Consolidate working memory to long-term memory | `vault`, `ids` |

**Example** (MCP call):
```json
{
  "method": "tools/call",
  "params": {
    "name": "muninn_metacognition_coverage",
    "arguments": {
      "vault": "my-agent",
      "query": "payment system architecture"
    }
  }
}
```

## 3.4 Engineering Safeguards (Phase 3)

### 3.4.1 Metacognitive Performance

| Metric | Target | Optimization Strategy |
|--------|--------|----------------------|
| Coverage calculation | <100ms | Precompute entity index |
| Entropy calculation | <50ms | Incremental update |
| Health diagnosis | <500ms | Parallel calculation of metrics |

### 3.4.2 Agent Integration Tests

**LangGraph Integration Test**:
```python
def test_langgraph_checkpoint():
    workflow = create_muninn_backed_workflow()

    # Run to intermediate state
    state1 = workflow.invoke({"input": "task 1"})

    # Create checkpoint
    checkpoint_id = memory.checkpoint(state1)

    # Continue running
    state2 = workflow.invoke({"input": "task 2"})

    # Restore and verify
    restored = memory.restore(checkpoint_id)
    assert restored == state1
```

### 3.4.3 Migration Path

| Existing Users | Migration Steps |
|----------------|-----------------|
| MCP users | `muninn init --upgrade` automatically registers new tools |
| REST users | API backward compatible, new endpoints optional |
| SDK users | Release new SDK version, support progressive upgrade |

**Phase 3 Acceptance Criteria**:
- [ ] Metacognitive API response latency <500ms (p99)
- [ ] Knowledge coverage assessment correlation with manual evaluation >0.7
- [ ] LangGraph integration package passes all official tests
- [ ] AutoGen integration package supports complete conversation memory management
- [ ] MCP new tools pass protocol compatibility tests

---

# Appendix A: Upstream MuninnDB Compatibility Maintenance

## A.1 Versioning Strategy

**Semantic Versioning**:
```
muninndb-cognitive v1.0.0
├── major: Incompatible API/storage format changes
├── minor: Backward compatible feature additions
└── patch: Backward compatible bug fixes
```

**Storage Format Versioning**:
```go
// internal/storage/erf/version.go
const ERFFormatVersion = "v2"  // Introduced in Phase 1

func Encode(engram *Engram) ([]byte, error) {
    if UseV2Format {
        return encodeV2(engram)
    }
    return encodeV1(engram)
}

func Decode(data []byte) (*Engram, error) {
    version := detectVersion(data)
    switch version {
    case "v1":
        return decodeV1(data)
    case "v2":
        return decodeV2(data)
    default:
        return nil, fmt.Errorf("unknown format version")
    }
}
```

## A.2 Branch Management

**Git Strategy**:
```
main (upstream MuninnDB)
  ↓ (merge regularly)
develop
  ↓ (feature branches)
  ├── feature/memory-types
  ├── feature/working-memory
  ├── feature/gnn-associations
  └── feature/metacognition
```

**Merge Window**: Merge from upstream/main every 2 weeks, resolve conflicts

## A.3 Configuration Compatibility

**Environment Variable Mapping**:
| New Variable | Old Variable | Default Value |
|--------------|--------------|---------------|
| `MUNINN_WM_CAPACITY` | - | 7 |
| `MUNINN_CONSOLIDATION_ENABLED` | - | true |
| `MUNINN_GNN_ENABLED` | - | false |
| `MUNINN_OLLAMA_URL` | `MUNINN_ENRICH_URL` | Reuse |

---

# Appendix B: Performance & Latency Control Strategies

## B.1 Performance Guarantees After GNN Introduction

| Operation | Baseline (no GNN) | Target (with GNN) | Optimization Approach |
|-----------|-------------------|-------------------|----------------------|
| Write | <10ms | <15ms | Asynchronous graph update |
| Activate | <20ms | <50ms | Lazy 2-hop loading |
| Relation inference | N/A | <100ms | Parallel traversal |

**Key Optimizations**:
1. **Graph Partitioning**: Partition by vault, avoid full graph traversal
2. **Edge Caching**: Cache hot engram association edges in memory
3. **Batch Updates**: GNN edge weights batch written (100ms window)

## B.2 Latency Control After LLM Introduction

| Scenario | Sync/Async | Timeout | Degradation Strategy |
|----------|------------|---------|---------------------|
| Relation extraction | Async | 5s | Fallback to rule engine |
| Prototype summary | Async | 30s | Template summary |
| Emotion analysis | Sync | 500ms | Skip emotional weight |

**Asynchronous Processing Architecture**:
```
HTTP Request (synchronous response)
      ↓
  Write to queue
      ↓
Background Worker (LLM processing)
      ↓
  Update storage
```

---

# Appendix C: Test Validation Plan

## C.1 Cognitive Behavior Simulation Test Suite

**Test File**: `internal/engine/cognitive_sim_test.go`

| Test Name | Validation Goal | Pass Criteria |
|-----------|----------------|---------------|
| `TestMillerLaw` | Working memory capacity | 7±2 items |
| `TestEbbinghausCurve` | Forgetting curve fitting | R² > 0.85 |
| `TestSpacingEffect` | Spaced repetition advantage | Spaced > Massed (p<0.05) |
| `TestHebbianLearning` | Co-activation enhancement | Weights monotonically increasing |
| `TestBayesianUpdating` | Confidence updating | Conforms to Bayes' formula |
| `TestContextualRecall` | Context-aware | Different context → different rankings |

## C.2 End-to-End Acceptance Tests

**Test File**: `tests/e2e/cognitive_features_test.go`

| Scenario | Steps | Expected Result |
|----------|-------|-----------------|
| Working memory → long-term memory | 1. Write to WM<br>2. Rehearse 3 times<br>3. Trigger consolidation | Engram appears in LTM query |
| Relation inference | 1. Store causal pairs<br>2. Query for reasons<br>3. Recall results | Engrams with causal relations |
| Prototype matching | 1. Store 10 similar cases<br>2. Query new case | Return matching prototype |
| Metacognitive diagnosis | 1. Inject low-confidence data<br>2. Call health API | Return high entropy alert |

## C.3 Performance Benchmarks

**Test File**: `internal/bench/cognitive_bench_test.go`

```go
func BenchmarkWorkingMemoryPush(b *testing.B) {
    wm := NewWorkingMemoryBuffer(DefaultConfig)
    for i := 0; i < b.N; i++ {
        wm.Push(createTestEngram())
    }
    // Target: <5ms
}

func BenchmarkContextualActivation(b *testing.B) {
    ctx := &ActivationContext{Task: "debug payment"}
    for i := 0; i < b.N; i++ {
        engine.Activate(ctx, "payment", 10)
    }
    // Target: <50ms (with GNN)
}
```

---

# Appendix D: Jira Task Breakdown Template

## D.1 Phase 1 Task Example

**EPIC**: COG-1 Foundation Enhancements (0-6 months)

| Task ID | Task Name | Priority | Estimate | Acceptance Criteria |
|---------|-----------|----------|----------|---------------------|
| COG-11 | Implement MemoryType enum | High | 2d | Pass type safety tests |
| COG-12 | Implement WorkingMemoryBuffer | High | 5d | Capacity/decay tests pass |
| COG-13 | Implement ConsolidationScheduler | High | 8d | Consolidation trigger tests pass |
| COG-14 | Extend Engram storage format | High | 3d | Backward compatibility tests pass |
| COG-15 | Implement DynamicForgetting | Medium | 5d | Ebbinghaus fit R²>0.85 |
| COG-16 | Implement spaced repetition reward | Medium | 3d | SpacingEffect tests pass |
| COG-17 | MCP tool extension | Medium | 4d | New tools registered successfully |
| COG-18 | Write cognitive behavior simulation tests | High | 5d | All tests pass |

## D.2 Task Template

```markdown
### Task: COG-XX [Task Name]

**EPIC**: COG-1/2/3

**Objective**: [One sentence description]

**Technical Design**:
- New files: `internal/path/to/file.go`
- Modified files: `internal/path/to/existing.go`
- Interface definitions: [Paste key interfaces]

**Dependencies**:
- [ ] COG-YY (prerequisite task)
- [ ] External dependencies: None

**Acceptance Criteria**:
- [ ] Functional tests pass
- [ ] Performance tests meet targets
- [ ] Backward compatibility tests pass
- [ ] Documentation updated

**Estimate**: X days

**Priority**: High/Medium/Low
```

---

# Summary

This roadmap progressively upgrades MuninnDB from the current "database with cognitive primitives" to a "fully ideal cognitive database", divided into three phases:

| Phase | Time | Core Capabilities | Key Deliverables |
|-------|------|-------------------|------------------|
| **Phase 1** | 0-6 months | Multi-scale memory hierarchy + Dynamic forgetting | Working memory buffer, Consolidation scheduler |
| **Phase 2** | 6-12 months | Deep associations + Context-aware | GNN association graph, Contextual weighting model, Prototype formation |
| **Phase 3** | 12-18 months | Metacognition + Agent integration | Metacognitive API, LangGraph/AutoGen integration packages |

**Technical Risks**:
1. GNN performance may become bottleneck → Mitigated via graph partitioning and lazy loading
2. LLM latency uncontrollable → Guaranteed via asynchronous processing and degradation strategies
3. Storage format changes affect compatibility → Guaranteed via versioning and migration tools

**Business Value**:
- Differentiated competition: No similar cognitive database currently supports complete memory hierarchy
- Agent economy: Deep integration with mainstream Agent frameworks, lowers adoption barrier
- Metacognitive capabilities: Provides knowledge base health management tools for enterprise users

**Next Actions**:
1. Launch Phase 1 development (COG-11 to COG-18)
2. Establish bi-weekly merge window mechanism (maintain synchronization with upstream)
3. Recruit beta testers (closed beta for Phase 1 features)

---

*Document Version: v1.0*
*Creation Date: 2026-03-29*
*Next Review: 2026-04-29*
