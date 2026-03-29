# MuninnDB 阶段 2 技术设计文档
## 认知深化：GNN 关联图 + 情境上下文

**版本**: v1.0  
**日期**: 2026-03-29  
**状态**: 设计审查中

---

## 概述

阶段 2 目标：实现深度联想网络与情境感知召回，使 MuninnDB 能够：
1. 理解复杂语义关系（因果/类比/隐喻）
2. 基于当前情境动态调整召回结果
3. 自动归纳形成知识原型

---

## 架构设计

### 2.1 Neuro-Symbolic 关联图

```
┌──────────────────────────────────────────────────────────┐
│              Neuro-Symbolic Association Graph            │
├──────────────────────────────────────────────────────────┤
│  Neural Layer (数值，隐式):                               │
│    - Hebbian co-activation weights                       │
│    - Semantic similarity (ONNX 嵌入)                      │
│    - Activation spreading                                │
├──────────────────────────────────────────────────────────┤
│  Symbolic Layer (符号，显式):                             │
│    - Causal relationships (A→B)                          │
│    - Analogical mappings (A:B::C:D)                      │
│    - Metaphorical links (A 是 B)                          │
│    - Temporal sequences (A 先于 B)                        │
│    - Taxonomic hierarchy (A 是 B的子类)                    │
└──────────────────────────────────────────────────────────┘
```

**技术选型**:
- 图引擎：`gonum.org/v1/gonum/graph` (纯 Go，无 CGO)
- 存储：Pebble DB (现有)
- 遍历：BFS/DFS + 启发式剪枝

---

## 模块设计

### COG-21: GNN 图结构

**文件**: `internal/index/gnn/graph.go`

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

**核心接口**:
```go
func (g *AssociationGraph) AddEngramNode(engramID, concept, memType) int64
func (g *AssociationGraph) AddRelation(from, to, relType, weight, confidence) (*Edge, error)
func (g *AssociationGraph) StrengthenRelation(from, to, delta) float64
func (g *AssociationGraph) Traverse(startID, maxDepth) []*CognitiveNode
func (g *AssociationGraph) GetRelationsByType(engramID, relType) []*Edge
```

**性能目标**:
- 节点添加：<1ms
- 2-hop 遍历：<50ms (p99)
- 边更新：<0.1ms

---

### COG-22: 关系类型系统

**文件**: `internal/index/gnn/relations.go` ✅ (已完成)

**关系类型**:
| 类型 | 值 | 方向性 | 对称性 | 示例 |
|------|---|--------|--------|------|
| `RelCoActivation` | 1 | 否 | ✅ | 共激活 |
| `RelCausal` | 2 | ✅ | ❌ | A 导致 B |
| `RelAnalogical` | 3 | 否 | ✅ | A:B::C:D |
| `RelMetaphorical` | 4 | 否 | ✅ | A 像 B |
| `RelTemporal` | 5 | ✅ | ❌ | A 先于 B |
| `RelPartonomic` | 6 | ✅ | ❌ | A 是 B 的部分 |
| `RelTaxonomic` | 7 | ✅ | ❌ | A 是 B 的子类 |
| `RelSupports` | 8 | ✅ | ❌ | A 支持 B |
| `RelContradicts` | 9 | ✅ | ❌ | A 矛盾 B |

**关系推断规则**:
```go
// 因果推断：基于时间顺序 + 共现频率
func InferCausal(rel1, rel2 *Edge) *Edge {
    if rel1.Timestamp < rel2.Timestamp && 
       rel1.CooccurrenceCount > threshold {
        return NewEdge(RelCausal, confidence=0.7)
    }
    return nil
}

// 类比推断：基于结构映射理论
func InferAnalogical(pair1, pair2 EdgePair) *Edge {
    if structuralSimilarity(pair1, pair2) > 0.8 {
        return NewEdge(RelAnalogical, confidence=0.8)
    }
    return nil
}
```

**验证规则**:
- 检测循环因果 (A→B→C→A)
- 检测矛盾关系 (A 支持 B 且 A 矛盾 B)
- 检测层级冲突 (A 是 B 子类 且 B 是 A 子类)

---

### COG-23: 情境上下文 Schema

**文件**: `internal/engine/context_schema.go`

```go
type ActivationContext struct {
    // 任务上下文
    Task        string           // 当前任务描述
    Goals       []string         // 目标列表
    SubGoalOf   *ULID            // 父目标 (支持层级)
    
    // 认知状态
    CognitiveLoad float64        // 0-1: 认知负荷
    TimePressure  float64        // 0-1: 时间压力
    WorkingMemory []ULID         // 当前工作记忆内容
    
    // 情感状态 (可选)
    EmotionalState map[string]float64  // {"anxiety": 0.7, "curiosity": 0.3}
    
    // 环境上下文
    Environment   map[string]string     // {"location": "office", "device": "mobile"}
    
    // 历史激活 (用于序列模式)
    RecentActivations []ULID
    
    // 时间上下文
    TimeOfDay     time.Time
    SessionStart  time.Time
}

// 序列化支持
func (ctx *ActivationContext) Marshal() ([]byte, error)
func (ctx *ActivationContext) Unmarshal(data []byte) error
```

**上下文提取**:
```go
// 从请求中提取上下文
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
    
    // 从 context 数组提取语义
    if len(req.Context) > 0 {
        ctx.WorkingMemory = resolveToULIDs(req.Context)
    }
    
    return ctx, nil
}
```

---

### COG-24: 动态权重计算

**文件**: `internal/scoring/contextual_weighting.go`

```go
// 多因素评分公式
func ContextualScore(engram *Engram, ctx *ActivationContext) float64 {
    // 基础分 (ACT-R + Hebbian)
    base := scoring.BaseLevelActivation(engram.AccessCount, engram.LastAccess)
    hebbian := scoring.HebbianBoost(engram.CoActivations, ctx.RecentActivations)
    
    // 情境调节因子
    taskRelevance := SemanticSimilarity(engram.Content, ctx.Task)
    goalAlignment := AlignsWithGoals(engram, ctx.Goals)
    emotionalCongruence := MatchEmotionalState(engram, ctx.EmotionalState)
    
    // 认知负荷调节 (高负荷→降低复杂度)
    complexityPenalty := ctx.CognitiveLoad * engram.ComplexityScore
    
    // 时间压力调节 (高压力→偏好高置信度)
    confidenceBonus := ctx.TimePressure * engram.Confidence
    
    return base + hebbian + 
           0.3*taskRelevance + 
           0.2*goalAlignment + 
           0.1*emotionalCongruence - 
           0.1*complexityPenalty + 
           0.2*confidenceBonus
}
```

**子模块**:

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
    // 从 engram 提取情感极性
    engramSentiment := AnalyzeSentiment(engram.Content)
    
    // 与当前情感状态匹配
    matchScore := 0.0
    for emotion, intensity := range state {
        if emotionMatches(engramSentiment, emotion) {
            matchScore += intensity * 0.5 // 情感一致性奖励
        }
    }
    return math.Min(matchScore, 1.0)
}
```

---

## 集成设计

### ACTIVATE 流程增强

```
原始流程:
1. 全文搜索 + 向量搜索 → 融合
2. Hebbian 提升
3. ACT-R 时间加权
4. 返回结果

增强后流程:
1. 提取 ActivationContext
2. 全文搜索 + 向量搜索 → 融合
3. GNN 图遍历 (2-hop, 关系过滤)
4. 情境权重计算 (ContextualScore)
5. 认知负荷调节
6. 返回结果 + 解释 (可选)
```

**API 变更**:
```json
// POST /api/activate
{
  "context": ["debug payment"],
  "task": "fix retry logic bug",
  "goals": ["prevent double-charge", "maintain idempotency"],
  "cognitive_load": 0.7,
  "time_pressure": 0.5,
  "max_results": 10,
  "explain": true  // 新增：返回评分解释
}

// Response
{
  "activations": [
    {
      "id": "...",
      "concept": "idempotency key conflict",
      "score": 0.89,
      "score_breakdown": {  // 新增
        "base": 0.5,
        "hebbian": 0.2,
        "task_relevance": 0.15,
        "goal_alignment": 0.18,
        "complexity_penalty": -0.07,
        "confidence_bonus": 0.1
      },
      "relations": [  // 新增
        {"type": "causal", "to": "payment failure", "weight": 0.8}
      ]
    }
  ]
}
```

---

## 测试计划

### GNN 图遍历测试
```go
func TestGNN_Traversal(t *testing.T) {
    graph := gnn.NewAssociationGraph()
    
    // 创建测试图
    // A --causal--> B --temporal--> C
    // A --analogical--> D
    
    nodes := graph.Traverse(A, 2)
    // 验证：应返回 [A, B, C, D]
}
```

### 情境感知测试
```go
func TestContextualRecall_TaskDependent(t *testing.T) {
    ctx1 := &ActivationContext{Task: "debug payment"}
    ctx2 := &ActivationContext{Task: "design feature"}
    
    result1 := engine.Activate(ctx1, "payment system", 10)
    result2 := engine.Activate(ctx2, "payment system", 10)
    
    // 验证：不同任务下排序应不同
    if rankingsAreIdentical(result1, result2) {
        t.Error("Contextual recall failed")
    }
}
```

### 关系推断测试
```go
func TestRelationInference_Causal(t *testing.T) {
    // A→B (时间顺序，高频共现)
    // 验证：应推断出 RelCausal
}
```

---

## 性能优化策略

### 1. 图分区
- 按 vault 分图
- 按主题域分区 (payment, auth, deployment...)
- 惰性加载 (仅加载 2-hop 邻居)

### 2. 缓存
- 热点节点缓存 (LRU)
- 遍历结果缓存 (TTL: 5 分钟)
- 情境 - 结果对缓存

### 3. 并行化
- 多线程图遍历
- 批量关系推断
- 异步情境权重计算

---

## 里程碑

| 里程碑 | 日期 | 交付物 |
|--------|------|--------|
| M1: GNN 基础 | Day 1 | graph.go + relations.go |
| M2: 关系推断 | Day 2 | inference.go + 验证 |
| M3: 情境 Schema | Day 2 | context_schema.go |
| M4: 动态权重 | Day 3 | contextual_weighting.go |
| M5: 集成测试 | Day 3 | e2e 测试套件 |
| M6: 文档 | Day 3 | API 文档 + 使用指南 |

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| gonum 性能不足 | 高 | 预加载 + 缓存；备选：自定义图结构 |
| 关系推断准确率低 | 中 | 人工审核模式；置信度阈值 |
| 情境计算延迟高 | 中 | 异步计算；降级策略 |
| 图规模爆炸 | 高 | 分区 + 剪枝；TTL 自动清理 |

---

## 下一步行动

1. ✅ 完成关系类型定义 (`relations.go`)
2. ⏳ 修复 GNN 图结构 (`graph.go`)
3. ⏳ 实现情境 Schema (`context_schema.go`)
4. ⏳ 实现动态权重 (`contextual_weighting.go`)
5. ⏳ 编写集成测试
6. ⏳ 更新 API 文档

---

*文档版本：v1.0*  
*创建日期：2026-03-29*  
*审查状态：待审查*
