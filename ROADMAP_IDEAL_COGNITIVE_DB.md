# MuninnDB 理想认知数据库技术路线图 (18 个月)

> **愿景**：将 MuninnDB 从"具有认知原语的数据库"升级为"完全理想的认知数据库"——能够模拟人类记忆核心机制的存储系统。

---

## 理想认知数据库的定义

一个能模拟人类记忆核心机制的存储系统，具备以下能力：

| 能力 | 描述 | 对应人脑机制 |
|------|------|-------------|
| **多尺度记忆分层** | 区分感觉记忆、工作记忆、情景记忆、语义记忆 | Atkinson-Shiffrin 记忆模型 |
| **动态遗忘与巩固** | 基于使用频率、情感权重、上下文重要性自动调节记忆强度 | Ebbinghaus 遗忘曲线 + 突触可塑性 |
| **深度联想网络** | 支持因果、类比、隐喻等复杂关系的跨模态联想 | Hebbian 学习 + 语义网络理论 |
| **情境感知召回** | 召回结果随当前任务、情绪、环境动态调整 | ACT-R 情境激活理论 |
| **记忆泛化与原型形成** | 自动归纳相似经历，生成可复用的知识原型 | 原型理论 + 归纳学习 |
| **元认知能力** | 评估自身知识完整性、置信度与盲区 | Flavell 元认知理论 |

---

## 约束条件

| 约束 | 实施方案 |
|------|---------|
| **协议兼容性** | 保持现有 MCP/REST/gRPC/MBP 协议不变，所有新功能通过扩展字段实现向后兼容 |
| **技术栈** | 优先 Go 生态，ML 组件仅引入 ONNX Runtime (已有) + 轻量 GNN (gonum) |
| **许可证** | 遵守 BSL 1.1，不复制受专利保护的核心逻辑（现有专利：Ebbinghaus decay, Hebbian learning, Bayesian confidence, semantic triggers） |
| **部署** | 保持单体部署能力，支持本地运行；集群功能为可选扩展 |

---

# 阶段 1：基础增强 (0–6 个月)

> **目标**：扩展 engram 模型以支持多尺度记忆分层，实现工作记忆缓冲区与动态巩固机制。

## 1.1 记忆类型分层系统

### 1.1.1 扩展 Engram 模型

**现有字段** (`internal/storage/engram.go`):
```go
type Engram struct {
    ID          ULID
    Concept     string
    Content     string
    Tags        []string
    Confidence  float64
    Stability   float64
    Relevance   float64
    MemoryType  MemoryType  // ← 已有但未充分利用
    // ...
}
```

**新增/修改模块**:

| 文件 | 职责 | 关键接口 |
|------|------|---------|
| `internal/engine/memory_types.go` | 定义 4 种记忆类型及其转换规则 | `type MemoryType uint8` |
| `internal/storage/engram_metadata.go` | 扩展元数据以支持记忆类型标签 | `UpdateMemoryType(id, type)` |
| `internal/engine/working_memory.go` | **新增**: 工作记忆缓冲区管理 | `PushToWM(), FlushFromWM()` |
| `internal/engine/consolidation_scheduler.go` | **新增**: 记忆巩固调度器 | `ScheduleConsolidation(engramID)` |

**MemoryType 枚举扩展**:
```go
type MemoryType uint8

const (
    MemoryTypeSensory    MemoryType = 0  // 感觉记忆：<100ms 暂存
    MemoryTypeWorking    MemoryType = 1  // 工作记忆：秒 - 分钟级
    MemoryTypeEpisodic   MemoryType = 2  // 情景记忆：具体事件
    MemoryTypeSemantic   MemoryType = 3  // 语义记忆：抽象知识
    MemoryTypeProcedural MemoryType = 4  // 程序记忆：技能/流程（预留）
)
```

### 1.1.2 工作记忆缓冲区设计

**架构**:
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

**接口定义** (`internal/engine/working_memory.go`):
```go
type WorkingMemoryBuffer struct {
    items      []*Engram
    maxSize    int           // 默认 7
    decayTime  time.Duration // 默认 30s
    mu         sync.RWMutex
}

func NewWorkingMemoryBuffer(config WMConfig) *WorkingMemoryBuffer
func (wm *WorkingMemoryBuffer) Push(engram *Engram) error
func (wm *WorkingMemoryBuffer) Flush(consolidate bool) []*Engram
func (wm *WorkingMemoryBuffer) Rehearse(id ULID) // 复述以延长保持
func (wm *WorkingMemoryBuffer) GetActive() []*Engram
```

**验收标准**:
- [ ] 工作记忆容量可配置 (默认 7±2)
- [ ] 无复述时 30 秒自动衰减
- [ ] 支持手动复述延长保持时间
- [ ] 巩固到长期记忆时保留关联边

### 1.1.3 记忆巩固触发机制

**巩固条件**:
1. **重复触发**: 工作记忆中被复述≥3 次
2. **情感权重**: 检测到高情感强度 (通过 LLM 情感分析)
3. **上下文重要性**: 与当前任务高度相关 (ACT-R 激活>阈值)

**调度器设计** (`internal/engine/consolidation_scheduler.go`):
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

// 内置规则
func DefaultConsolidationRules() []ConsolidationRule {
    return []ConsolidationRule{
        {Name: "rehearsal", Condition: checkRehearsalCount, Priority: 1},
        {Name: "emotional_salience", Condition: checkEmotionalWeight, Priority: 2},
        {Name: "contextual_relevance", Condition: checkACTRActivation, Priority: 3},
    }
}
```

**API 扩展** (向后兼容):
```json
// POST /api/engrams
{
  "concept": "...",
  "content": "...",
  "memory_type": "working",  // 新增字段，可选
  "metadata": {
    "emotional_weight": 0.8  // 新增字段，可选
  }
}
```

## 1.2 动态遗忘与强度调节

### 1.2.1 扩展 Ebbinghaus 衰减模型

**现有实现** (`internal/scoring/temporal.go`):
```go
// 当前仅基于 ACT-R 基础激活
func BaseLevelActivation(n int, ageDays float64) float64 {
    return math.Log(float64(n+1)) - 0.5*math.Log(ageDays/float64(n+1))
}
```

**增强方案**:
```go
// 新增：多因素强度调节
func DynamicForgetting(engram *Engram, ctx *AccessContext) float64 {
    base := BaseLevelActivation(engram.AccessCount, engram.AgeDays())
    
    // 情感权重调节 (高情感→衰减慢)
    emotionalMod := 1.0 - (engram.EmotionalWeight * 0.3)
    
    // 睡眠巩固模拟 (离线期间衰减暂停)
    sleepMod := ctx.DuringSleep ? 0.1 : 1.0
    
    // 间隔重复奖励
    spacingBonus := calculateSpacingBonus(engram.AccessHistory)
    
    return base * emotionalMod * sleepMod * spacingBonus
}
```

**新增模块**:
| 文件 | 职责 |
|------|------|
| `internal/scoring/emotional_weight.go` | 情感权重计算 (通过 LLM 或规则) |
| `internal/scoring/spacing_effect.go` | 间隔重复奖励计算 |
| `internal/engine/sleep_scheduler.go` | 模拟睡眠巩固周期 |

### 1.2.2 记忆强度自动调节

**强度等级** (5 级):
```go
type MemoryStrength uint8

const (
    StrengthFragile   MemoryStrength = 0  // 易损：新编码，未巩固
    StrengthLabile    MemoryStrength = 1  // 不稳定：需复述
    StrengthStable    MemoryStrength = 2  // 稳定：正常遗忘曲线
    StrengthRobust    MemoryStrength = 3  // 强健：抗干扰
    StrengthPermanent MemoryStrength = 4  // 永久：几乎不遗忘
)
```

**自动调节逻辑** (`internal/engine/strength_adaptor.go`):
```go
func AdaptStrength(engram *Engram, history []AccessEvent) MemoryStrength {
    // 基于以下因素计算:
    // 1. 访问次数
    // 2. 访问间隔分布
    // 3. 巩固状态
    // 4. 关联边数量
    
    if engram.Consolidated && len(engram.Associations) > 10 {
        return StrengthRobust
    }
    // ...
}
```

## 1.3 工程保障 (阶段 1)

### 1.3.1 兼容性维护策略

| 变更类型 | 兼容性方案 | 验收测试 |
|---------|-----------|---------|
| 新增 `memory_type` 字段 | 默认为`MemoryTypeSemantic`，旧客户端不感知 | 向后兼容测试套件 |
| 扩展 MCP 工具 | 新增`muninn_remember_enhanced`，保留原`muninn_remember` | MCP 协议回归测试 |
| 存储格式变更 | ERF 格式版本化 (`erf/v2`)，支持自动迁移 | 升级/降级测试 |

### 1.3.2 性能控制

| 场景 | 延迟目标 | 优化策略 |
|------|---------|---------|
| 工作记忆写入 | <5ms | 内存操作，无持久化 |
| 工作记忆→长期记忆巩固 | <50ms | 异步批处理 |
| 动态遗忘计算 | <1ms/engram | 缓存预计算结果 |

### 1.3.3 测试验证方案

**认知行为仿真测试集** (`internal/engine/cognitive_sim_test.go`):
```go
// 测试用例示例：验证间隔重复效应
func TestSpacingEffect_Consolidation(t *testing.T) {
    eng := createTestEngram()
    
    // 场景 A: 集中学习 (5 次/1 小时)
    massed := simulateMassedPractice(eng, 5, time.Hour)
    
    // 场景 B: 分散学习 (5 次/5 天)
    spaced := simulateSpacedPractice(eng, 5, 5*24*time.Hour)
    
    // 验证：分散学习应保持更高强度
    if spaced.Strength <= massed.Strength {
        t.Errorf("Spacing effect violated")
    }
}
```

**阶段 1 验收标准**:
- [ ] 工作记忆缓冲区通过 Miller's Law 验证 (7±2 容量)
- [ ] 动态遗忘曲线符合 Ebbinghaus 模型 (R² > 0.85)
- [ ] 间隔重复效应通过行为仿真测试
- [ ] MCP 协议向后兼容 (旧客户端正常工作)
- [ ] 工作记忆写入延迟 <5ms (p99)

---

# 阶段 2：认知深化 (6–12 个月)

> **目标**：实现深度联想网络与情境感知召回，支持复杂关系推理与动态权重计算。

## 2.1 联想网络升级

### 2.1.1 Neuro-Symbolic 图表示

**现状**: 当前关联仅支持简单共激活边 (`internal/storage/association.go`)

**目标架构**:
```
┌──────────────────────────────────────────────────────────┐
│              Neuro-Symbolic Association Graph            │
├──────────────────────────────────────────────────────────┤
│  Neural Layer (现有):                                     │
│    - Hebbian co-activation weights (数值，隐式)           │
│    - Semantic similarity (ONNX 嵌入)                      │
├──────────────────────────────────────────────────────────┤
│  Symbolic Layer (新增):                                   │
│    - Causal relationships (因果：A→B)                    │
│    - Analogical mappings (类比：A:B::C:D)                │
│    - Metaphorical links (隐喻：A 是 B)                   │
│    - Temporal sequences (时序：A 先于 B)                  │
└──────────────────────────────────────────────────────────┘
```

**新增模块**:
| 文件 | 职责 | 依赖 |
|------|------|------|
| `internal/index/gnn/graph.go` | **新增**: GNN 图结构定义 | gonum/graph |
| `internal/index/gnn/edge_types.go` | **新增**: 关系类型枚举 | - |
| `internal/engine/association_inference.go` | **新增**: 从文本推断关系类型 | 本地 LLM (可选) |
| `internal/scoring/gnn_scoring.go` | **新增**: GNN 辅助的关联评分 | gonum/optimize |

**关系类型定义** (`internal/index/gnn/edge_types.go`):
```go
type RelationType string

const (
    RelCoActivation   RelationType = "co_activation"   // 共激活 (现有 Hebbian)
    RelCausal         RelationType = "causal"          // A 导致 B
    RelAnalogical     RelationType = "analogical"      // A:B::C:D
    RelMetaphorical   RelationType = "metaphorical"    // A 是 B
    RelTemporal       RelationType = "temporal"        // A 先于 B
    RelPartonomic     RelationType = "partonomic"      // A 是 B 的部分
    RelTaxonomic      RelationType = "taxonomic"       // A 是 B 的子类
)

type SymbolicEdge struct {
    Source    ULID
    Target    ULID
    Type      RelationType
    Confidence float64
    Evidence  []string  // 支持该关系的文本片段
}
```

### 2.1.2 关系推断 Pipeline

**自动提取流程**:
```
Engram Content
      ↓
┌─────────────────┐
│  LLM/Rule-Based │ ← 本地 LLM (Ollama) 或规则引擎
│  Relation Extractor │
└─────────────────┘
      ↓
┌─────────────────┐
│  Validation Layer  │ ← 检查逻辑一致性 (无循环因果等)
└─────────────────┘
      ↓
┌─────────────────┐
│  Graph Insertion   │ ← 写入 GNN 索引
└─────────────────┘
```

**接口定义** (`internal/engine/association_inference.go`):
```go
type RelationInferrer struct {
    llmClient  *ollama.Client  // 可选
    ruleEngine *RuleEngine
}

func (ri *RelationInferrer) ExtractRelations(content string, context []ULID) []SymbolicEdge
func (ri *RelationInferrer) ValidateGraph(edges []SymbolicEdge) error  // 检测循环等
```

## 2.2 情境上下文 Schema

### 2.2.1 情境表示模型

**设计目标**: 召回结果随当前任务、情绪、环境动态调整

**Context Schema** (`internal/engine/context_schema.go`):
```go
type ActivationContext struct {
    // 任务上下文
    Task        string           // 当前任务描述
    Goals       []string         // 目标列表
    SubGoalOf   *ULID            // 父目标 (支持层级)
    
    // 认知状态
    CognitiveLoad float64        // 0-1: 认知负荷 (高→返回更少但更相关)
    TimePressure  float64        // 0-1: 时间压力 (高→跳过深度推理)
    
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
```

### 2.2.2 动态权重计算模型

**多因素评分公式**:
```go
func ContextualScore(engram *Engram, ctx *ActivationContext) float64 {
    // 基础分 (ACT-R + Hebbian)
    base := scoring.BaseLevelActivation(engram.AccessCount, engram.AgeDays())
    hebbian := scoring.HebbianBoost(engram.CoActivations, ctx.RecentActivations)
    
    // 情境调节因子
    taskRelevance := semanticSimilarity(engram.Content, ctx.Task)
    goalAlignment := alignsWithGoals(engram, ctx.Goals)
    emotionalCongruence := matchEmotionalState(engram, ctx.EmotionalState)
    
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

**新增模块**:
| 文件 | 职责 |
|------|------|
| `internal/engine/context_schema.go` | 情境 Schema 定义与验证 |
| `internal/scoring/contextual_weighting.go` | 情境权重计算 |
| `internal/engine/goal_tracker.go` | 目标层级追踪与管理 |
| `internal/plugin/emotion_analyzer.go` | 情感状态分析 (可选插件) |

## 2.3 记忆泛化 Pipeline

### 2.3.1 原型形成架构

**流程**:
```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Clustering  │ →   │  LLM Summary │ →   │  Prototype   │
│  (相似经历)   │     │  (提取共性)   │     │  Storage     │
└──────────────┘     └──────────────┘     └──────────────┘
       ↑                                        ↓
       └────────────────────────────────────────┘
                  新经历匹配→更新原型
```

### 2.3.2 实现方案

**阶段 1: 聚类** (`internal/engine/prototype_clusterer.go`):
```go
type PrototypeClusterer struct {
    embedder  Embedder
    algorithm string  // "kmeans", "dbscan", "hierarchical"
}

func (pc *PrototypeClusterer) FindClusters(engrams []*Engram, vault string) []Cluster {
    // 1. 嵌入所有 engrams
    // 2. 运行聚类算法
    // 3. 返回聚类结果 (每个聚类包含成员 IDs + 中心向量)
}
```

**阶段 2: LLM 摘要** (`internal/engine/prototype_summarizer.go`):
```go
type PrototypeSummarizer struct {
    llmClient *ollama.Client  // 可选
}

func (ps *PrototypeSummarizer) GeneratePrototype(cluster Cluster) *Prototype {
    // 输入：聚类中所有 engrams 的内容
    // 输出：抽象化的原型描述
    // 示例:
    //   输入：[支付失败案例 1, 支付失败案例 2, ...]
    //   输出："支付系统失败通常由 idempotency key 冲突导致..."
}
```

**阶段 3: 原型存储** (`internal/storage/prototype.go`):
```go
type Prototype struct {
    ID          ULID
    Concept     string          // 原型名称
    Description string          // 抽象描述
    MemberCount int             // 支持该原型的经历数量
    Members     []ULID          // 成员 engram IDs
    Embedding   []float32       // 原型嵌入 (聚类中心)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 原型也存储在 Pebble 中，支持检索
func (ps *PebbleStore) UpsertPrototype(proto *Prototype) error
func (ps *PebbleStore) FindPrototypes(query []float32, k int) ([]*Prototype, error)
```

### 2.3.3 原型匹配与更新

**匹配逻辑**:
```go
func (pc *PrototypeClusterer) MatchToPrototype(engram *Engram) (*Prototype, float64) {
    // 计算 engram 与所有原型的相似度
    // 返回最佳匹配 + 相似度分数
    // 如果分数 < 阈值，则触发新原型形成
}
```

**更新规则**:
- 新成员加入→更新原型嵌入 (移动平均)
- 成员数量≥10→触发 LLM 重新摘要
- 原型间相似度>0.9→合并原型

## 2.4 工程保障 (阶段 2)

### 2.4.1 GNN 性能优化

| 优化 | 实施方案 | 预期收益 |
|------|---------|---------|
| 图分区 | 按 vault 或主题域分区 | 减少遍历范围 |
| 惰性加载 | 仅加载 2-hop 邻居 | 降低内存占用 |
| 并行遍历 | 使用 goroutine 并行探索 | 3-5x 速度提升 |

### 2.4.2 LLM 依赖管理

| 场景 | 方案 | 降级策略 |
|------|------|---------|
| 关系提取 | 本地 Ollama (默认) | 回退到规则引擎 |
| 原型摘要 | 本地 Ollama (默认) | 回退到模板摘要 |
| 情感分析 | 插件 (可选) | 跳过情感权重 |

### 2.4.3 测试验证

**情境感知测试** (`internal/engine/contextual_recall_test.go`):
```go
func TestContextualRecall_TaskDependent(t *testing.T) {
    ctx1 := &ActivationContext{Task: "debug payment"}
    ctx2 := &ActivationContext{Task: "design new feature"}
    
    result1 := engine.Activate(ctx1, "payment system", 10)
    result2 := engine.Activate(ctx2, "payment system", 10)
    
    // 验证：不同任务下召回结果排序应不同
    if resultsAreIdentical(result1, result2) {
        t.Error("Contextual recall failed: results should differ by task")
    }
}
```

**阶段 2 验收标准**:
- [ ] 支持≥5 种关系类型 (因果、类比、隐喻、时序、分类)
- [ ] 关系提取准确率 >70% (与人工标注对比)
- [ ] 情境感知召回通过任务依赖测试
- [ ] 原型形成 pipeline 端到端延迟 <500ms
- [ ] GNN 图遍历 (2-hop) <50ms (p99)

---

# 阶段 3：元认知与 AGI 就绪 (12–18 个月)

> **目标**：实现元认知监控与自我诊断能力，深度集成主流 Agent 框架。

## 3.1 元认知监控指标

### 3.1.1 知识覆盖率评估

**定义**: 当前知识库对查询领域的覆盖程度

**计算方法** (`internal/metacognition/coverage.go`):
```go
type CoverageAnalyzer struct {
    entityIndex *EntityIndex
    topicModel  *TopicModel
}

func (ca *CoverageAnalyzer) CalculateCoverage(query string, vault string) *CoverageReport {
    // 1. 提取查询中的实体和主题
    entities := extractEntities(query)
    topics := inferTopics(query)
    
    // 2. 计算实体覆盖率 (已知实体 / 总实体)
    entityCoverage := ca.countKnownEntities(entities, vault) / len(entities)
    
    // 3. 计算主题覆盖率 (向量空间密度)
    topicDensity := ca.calculateTopicDensity(topics, vault)
    
    // 4. 综合评分
    return &CoverageReport{
        EntityCoverage: entityCoverage,
        TopicDensity:   topicDensity,
        OverallScore:   0.6*entityCoverage + 0.4*topicDensity,
        BlindSpots:     ca.identifyBlindSpots(entities, topics, vault),
    }
}
```

**输出示例**:
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

### 3.1.2 置信熵计算

**定义**: 知识库中信息的一致性程度 (高熵=多矛盾/低置信度)

**计算公式**:
```go
func CalculateConfidenceEntropy(vault string) float64 {
    // 1. 获取所有 engrams 的置信度分布
    confidences := getConfidenceDistribution(vault)
    
    // 2. 计算 Shannon 熵
    // H = -Σ p(x) × log2(p(x))
    // 其中 x 为置信度区间 (0-0.2, 0.2-0.4, ...)
    
    // 3. 标准化到 0-1 (1=最高不确定性)
    return normalizeEntropy(shannonEntropy(confidences))
}

// 矛盾检测增强
func DetectContradictionHotspots(vault string) []ContradictionCluster {
    // 识别置信度<0.5 且存在 contradicts 关系的 engram 群组
}
```

### 3.1.3 元认知指标体系

| 指标 | 计算方式 | 阈值 | 告警 |
|------|---------|------|------|
| **知识覆盖率** | 实体覆盖×0.6 + 主题密度×0.4 | <0.5 | 推荐补充知识 |
| **置信熵** | Shannon 熵标准化 | >0.7 | 存在多矛盾 |
| **记忆新鲜度** | 最近 7 天访问 engram 占比 | <0.2 | 知识可能过时 |
| **关联密度** | 平均每 engram 关联边数 | <2 | 联想能力弱 |
| **原型覆盖率** | 有原型的聚类占比 | <0.3 | 泛化能力弱 |

## 3.2 自我诊断 API

### 3.2.1 API 设计

**端点**:
```
GET  /api/metacognition/coverage?query={query}&vault={vault}
GET  /api/metacognition/entropy?vault={vault}
GET  /api/metacognition/health?vault={vault}
POST /api/metacognition/diagnose  # 全面诊断
```

**响应示例** (`GET /api/metacognition/health`):
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

### 3.2.2 实现模块

| 文件 | 职责 |
|------|------|
| `internal/metacognition/coverage.go` | 覆盖率分析 |
| `internal/metacognition/entropy.go` | 置信熵计算 |
| `internal/metacognition/health.go` | 综合健康评估 |
| `internal/transport/rest/metacognition.go` | REST API 处理 |
| `internal/transport/grpc/metacognition.proto` | gRPC 服务定义 |

## 3.3 与 Agent 框架深度集成

### 3.3.1 LangGraph 集成模式

**架构**:
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

**集成包** (`sdk/python/langchain_muninn/`):
```python
from langgraph.graph import StateGraph, END
from muninn.langchain import MuninnDBMemory

# 创建 Muninn-backed 状态图
memory = MuninnDBMemory(vault="agent-workspace")

workflow = StateGraph(AgentState)
workflow.add_node("agent", create_agent_node(memory))
workflow.add_node("memory_write", memory.write_node())
workflow.add_node("memory_read", memory.read_node())

# 定义边
workflow.add_edge("agent", "memory_write")
workflow.add_edge("memory_write", END)

app = workflow.compile()
```

**关键接口**:
```python
class MuninnDBMemory:
    def write_node(self) -> Callable:
        """返回 LangGraph node 函数，自动写入工作记忆"""
    
    def read_node(self) -> Callable:
        """返回 LangGraph node 函数，基于上下文读取相关记忆"""
    
    async def checkpoint(self, state: AgentState):
        """将 Agent 状态快照存储为情景记忆"""
    
    async def restore(self, checkpoint_id: str) -> AgentState:
        """从情景记忆恢复 Agent 状态"""
```

### 3.3.2 AutoGen 集成模式

**架构**:
```python
from autogen import ConversableAgent
from muninn.autogen import MuninnDBConversableAgent

# 扩展 AutoGen Agent 以支持 MuninnDB 记忆
agent = MuninnDBConversableAgent(
    name="assistant",
    llm_config={"config_list": [...]},
    muninn_config={
        "vault": "autogen-agent",
        "working_memory_capacity": 7,
        "consolidation_enabled": True,
    }
)

# 自动记忆管理
# - 每轮对话自动写入工作记忆
# - 达到容量时自动巩固到长期记忆
# - 响应时自动激活相关记忆
```

**集成包** (`sdk/python/autogen_muninn/`):
```python
class MuninnDBConversableAgent(ConversableAgent):
    def __init__(self, *, muninn_config: dict, **kwargs):
        super().__init__(**kwargs)
        self.memory_client = MuninnClient(...)
        self.working_memory = WorkingMemoryBuffer(...)
        
    async def a_generate_reply(self, messages, **kwargs):
        # 1. 从长期记忆激活相关上下文
        context = await self._activate_relevant_memory(messages)
        
        # 2. 增强 prompt 与记忆上下文
        enhanced_messages = self._inject_memory_context(messages, context)
        
        # 3. 生成响应
        response = await super().a_generate_reply(enhanced_messages, **kwargs)
        
        # 4. 写入工作记忆
        await self.working_memory.push(Engram(...))
        
        return response
```

### 3.3.3 MCP 协议扩展

**新增 MCP 工具**:
| 工具 | 用途 | 参数 |
|------|------|------|
| `muninn_metacognition_health` | 获取知识库健康报告 | `vault` |
| `muninn_metacognition_coverage` | 查询特定主题覆盖率 | `vault`, `query` |
| `muninn_checkpoint_create` | 创建 Agent 状态快照 | `vault`, `state`, `metadata` |
| `muninn_checkpoint_restore` | 恢复 Agent 状态快照 | `checkpoint_id` |
| `muninn_working_memory_push` | 写入工作记忆 | `vault`, `engram` |
| `muninn_working_memory_flush` | 巩固工作记忆到长期记忆 | `vault`, `ids` |

**示例** (MCP 调用):
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

## 3.4 工程保障 (阶段 3)

### 3.4.1 元认知性能

| 指标 | 目标 | 优化策略 |
|------|------|---------|
| 覆盖率计算 | <100ms | 预计算实体索引 |
| 熵计算 | <50ms | 增量更新 |
| 健康诊断 | <500ms | 并行计算各指标 |

### 3.4.2 Agent 集成测试

**LangGraph 集成测试**:
```python
def test_langgraph_checkpoint():
    workflow = create_muninn_backed_workflow()
    
    # 运行到中间状态
    state1 = workflow.invoke({"input": "task 1"})
    
    # 创建检查点
    checkpoint_id = memory.checkpoint(state1)
    
    # 继续运行
    state2 = workflow.invoke({"input": "task 2"})
    
    # 恢复并验证
    restored = memory.restore(checkpoint_id)
    assert restored == state1
```

### 3.4.3 迁移路径

| 现有用户 | 迁移步骤 |
|---------|---------|
| MCP 用户 | `muninn init --upgrade` 自动注册新工具 |
| REST 用户 | API 向后兼容，新端点可选使用 |
| SDK 用户 | 发布新版本 SDK，支持渐进式升级 |

**阶段 3 验收标准**:
- [ ] 元认知 API 响应延迟 <500ms (p99)
- [ ] 知识覆盖率评估与人工评估相关性 >0.7
- [ ] LangGraph 集成包通过所有官方测试
- [ ] AutoGen 集成包支持完整对话记忆管理
- [ ] MCP 新增工具通过协议兼容性测试

---

# 附录 A：与上游 MuninnDB 兼容性维护

## A.1 版本化策略

**语义化版本**:
```
muninndb-cognitive v1.0.0
├── major: 不兼容的 API/存储格式变更
├── minor: 向后兼容的功能新增
└── patch: 向后兼容的 bug 修复
```

**存储格式版本化**:
```go
// internal/storage/erf/version.go
const ERFFormatVersion = "v2"  // 阶段 1 引入

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

## A.2 分支管理

**Git 策略**:
```
main (上游 MuninnDB)
  ↓ (定期合并)
develop
  ↓ (功能分支)
  ├── feature/memory-types
  ├── feature/working-memory
  ├── feature/gnn-associations
  └── feature/metacognition
```

**合并窗口**: 每 2 周从 upstream/main 合并一次，解决冲突

## A.3 配置兼容

**环境变量映射**:
| 新变量 | 旧变量 | 默认值 |
|--------|--------|--------|
| `MUNINN_WM_CAPACITY` | - | 7 |
| `MUNINN_CONSOLIDATION_ENABLED` | - | true |
| `MUNINN_GNN_ENABLED` | - | false |
| `MUNINN_OLLAMA_URL` | `MUNINN_ENRICH_URL` | 复用 |

---

# 附录 B：性能与延迟控制策略

## B.1 GNN 引入后的性能保障

| 操作 | 基准 (无 GNN) | 目标 (有 GNN) | 优化手段 |
|------|-------------|-------------|---------|
| 写入 | <10ms | <15ms | 异步图更新 |
| 激活 | <20ms | <50ms | 惰性 2-hop 加载 |
| 关系推理 | N/A | <100ms | 并行遍历 |

**关键优化**:
1. **图分区**: 按 vault 分图，避免全图遍历
2. **边缓存**: 热点 engram 的关联边缓存在内存
3. **批量更新**: GNN 边权重批量写入 (100ms 窗口)

## B.2 LLM 引入后的延迟控制

| 场景 | 同步/异步 | 超时 | 降级策略 |
|------|----------|------|---------|
| 关系提取 | 异步 | 5s | 回退规则引擎 |
| 原型摘要 | 异步 | 30s | 模板摘要 |
| 情感分析 | 同步 | 500ms | 跳过情感权重 |

**异步处理架构**:
```
HTTP Request (同步响应)
      ↓
  写入队列
      ↓
后台 Worker (LLM 处理)
      ↓
  更新存储
```

---

# 附录 C：测试验证方案

## C.1 认知行为仿真测试集

**测试文件**: `internal/engine/cognitive_sim_test.go`

| 测试名称 | 验证目标 | 通过标准 |
|---------|---------|---------|
| `TestMillerLaw` | 工作记忆容量 | 7±2 项 |
| `TestEbbinghausCurve` | 遗忘曲线拟合 | R² > 0.85 |
| `TestSpacingEffect` | 间隔重复优势 | 分散>集中 (p<0.05) |
| `TestHebbianLearning` | 共激活增强 | 权重单调递增 |
| `TestBayesianUpdating` | 置信度更新 | 符合贝叶斯公式 |
| `TestContextualRecall` | 情境感知 | 不同情境→不同排序 |

## C.2 端到端验收测试

**测试文件**: `tests/e2e/cognitive_features_test.go`

| 场景 | 步骤 | 预期结果 |
|------|------|---------|
| 工作记忆→长期记忆 | 1.写入 WM<br>2.复述 3 次<br>3.触发巩固 | engram 出现在 LTM 查询中 |
| 关系推理 | 1.存储因果对<br>2.查询原因<br>3.召回结果 | 包含因果相关的 engrams |
| 原型匹配 | 1.存储 10 个相似案例<br>2.查询新案例 | 返回匹配的原型 |
| 元认知诊断 | 1.注入低置信度数据<br>2.调用健康 API | 返回高熵告警 |

## C.3 性能基准测试

**测试文件**: `internal/bench/cognitive_bench_test.go`

```go
func BenchmarkWorkingMemoryPush(b *testing.B) {
    wm := NewWorkingMemoryBuffer(DefaultConfig)
    for i := 0; i < b.N; i++ {
        wm.Push(createTestEngram())
    }
    // 目标：<5ms
}

func BenchmarkContextualActivation(b *testing.B) {
    ctx := &ActivationContext{Task: "debug payment"}
    for i := 0; i < b.N; i++ {
        engine.Activate(ctx, "payment", 10)
    }
    // 目标：<50ms (含 GNN)
}
```

---

# 附录 D：Jira 任务拆解模板

## D.1 阶段 1 任务示例

**EPIC**: COG-1 基础增强 (0-6 个月)

| 任务 ID | 任务名称 | 优先级 | 估算 | 验收标准 |
|--------|---------|--------|------|---------|
| COG-11 | 实现 MemoryType 枚举 | High | 2d | 通过类型安全测试 |
| COG-12 | 实现 WorkingMemoryBuffer | High | 5d | 容量/衰减测试通过 |
| COG-13 | 实现 ConsolidationScheduler | High | 8d | 巩固触发测试通过 |
| COG-14 | 扩展 Engram 存储格式 | High | 3d | 向后兼容测试通过 |
| COG-15 | 实现 DynamicForgetting | Medium | 5d | Ebbinghaus 拟合 R²>0.85 |
| COG-16 | 实现间隔重复奖励 | Medium | 3d | SpacingEffect 测试通过 |
| COG-17 | MCP 工具扩展 | Medium | 4d | 新工具注册成功 |
| COG-18 | 编写认知行为仿真测试 | High | 5d | 所有测试通过 |

## D.2 任务模板

```markdown
### 任务：COG-XX [任务名称]

**所属 EPIC**: COG-1/2/3

**目标**: [一句话描述]

**技术设计**:
- 新增文件：`internal/path/to/file.go`
- 修改文件：`internal/path/to/existing.go`
- 接口定义：[粘贴关键接口]

**依赖项**:
- [ ] COG-YY (前置任务)
- [ ] 外部依赖：无

**验收标准**:
- [ ] 功能测试通过
- [ ] 性能测试达标
- [ ] 向后兼容测试通过
- [ ] 文档更新完成

**估算**: X 天

**优先级**: High/Medium/Low
```

---

# 总结

本路线图将 MuninnDB 从当前的"具有认知原语的数据库"逐步升级为"完全理想的认知数据库"，分为三个阶段：

| 阶段 | 时间 | 核心能力 | 关键交付物 |
|------|------|---------|-----------|
| **阶段 1** | 0-6 月 | 多尺度记忆分层 + 动态遗忘 | 工作记忆缓冲区、巩固调度器 |
| **阶段 2** | 6-12 月 | 深度联想 + 情境感知 | GNN 关联图、情境权重模型、原型形成 |
| **阶段 3** | 12-18 月 | 元认知 + Agent 集成 | 元认知 API、LangGraph/AutoGen 集成包 |

**技术风险**:
1. GNN 性能可能成为瓶颈 → 通过图分区和惰性加载缓解
2. LLM 延迟不可控 → 通过异步处理和降级策略保障
3. 存储格式变更影响兼容性 → 通过版本化和迁移工具保障

**商业价值**:
- 差异化竞争：目前无同类认知数据库支持完整记忆分层
- Agent 经济：深度集成主流 Agent 框架，降低采用门槛
- 元认知能力：为企业级用户提供知识库健康管理工具

**下一步行动**:
1. 启动阶段 1 开发 (COG-11 至 COG-18)
2. 建立双周合并窗口机制 (保持与上游同步)
3. 招募测试用户 (封闭测试阶段 1 功能)

---

*文档版本：v1.0*
*创建日期：2026-03-29*
*下次审查：2026-04-29*
