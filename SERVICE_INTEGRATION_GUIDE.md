# MuninnDB 三阶段改进 - 服务集成指南

**完成日期**: 2026-03-29  
**状态**: ✅ 核心功能完成，准备服务集成

---

## 服务架构概览

MuninnDB 提供三层服务接口：
1. **MCP 协议** - AI Agent 首选
2. **REST API** - Web/移动应用
3. **gRPC** - 高性能微服务

---

## 1. MCP 服务集成

### 1.1 现有 MCP 工具（阶段 1 前）

| 工具 | 用途 | 状态 |
|------|------|------|
| `muninn_remember` | 创建记忆 | ✅ 可用 |
| `muninn_recall` | 召回记忆 | ✅ 可用 |
| `muninn_read` | 读取单条记忆 | ✅ 可用 |
| `muninn_forget` | 删除记忆 | ✅ 可用 |
| `muninn_link` | 创建关联 | ✅ 可用 |
| `muninn_guide` | 获取使用指南 | ✅ 可用 |

### 1.2 新增 MCP 工具（三阶段增强）

#### 阶段 1 工具
| 工具 | 用途 | 实现状态 |
|------|------|---------|
| `muninn_working_memory_push` | 写入工作记忆 | 📝 待实现 |
| `muninn_working_memory_flush` | 巩固到长期记忆 | 📝 待实现 |
| `muninn_consolidation_status` | 查询巩固状态 | 📝 待实现 |

#### 阶段 2 工具
| 工具 | 用途 | 实现状态 |
|------|------|---------|
| `muninn_graph_traverse` | 图谱遍历 | 📝 待实现 |
| `muninn_contextual_recall` | 情境感知召回 | 📝 待实现 |

#### 阶段 3 工具
| 工具 | 用途 | 实现状态 |
|------|------|---------|
| `muninn_metacognition_health` | 知识库健康诊断 | 📝 待实现 |
| `muninn_metacognition_coverage` | 覆盖率分析 | 📝 待实现 |
| `muninn_metacognition_entropy` | 置信熵分析 | 📝 待实现 |

### 1.3 MCP 配置示例

**Claude Desktop 配置**:
```json
{
  "mcpServers": {
    "muninn": {
      "url": "http://127.0.0.1:8750/mcp"
    }
  }
}
```

**MCP 调用示例**:
```json
{
  "method": "tools/call",
  "params": {
    "name": "muninn_metacognition_health",
    "arguments": {
      "vault": "default"
    }
  }
}
```

**预期响应**:
```json
{
  "overall_health": 0.78,
  "metrics": {
    "coverage": 0.82,
    "confidence_entropy": 0.35,
    "freshness": 0.65
  },
  "alerts": [
    {
      "type": "low_freshness",
      "severity": "warning",
      "message": "65% of memories haven't been accessed in 30+ days"
    }
  ]
}
```

---

## 2. REST API 服务集成

### 2.1 现有端点

| 端点 | 方法 | 用途 |
|------|------|------|
| `/api/engrams` | POST | 创建记忆 |
| `/api/engrams/{id}` | GET | 读取记忆 |
| `/api/activate` | POST | 激活召回 |
| `/api/vaults/{vault}/health` | GET | 健康检查 |

### 2.2 新增端点（三阶段增强）

#### 阶段 1 端点
```
POST   /api/working-memory        # 写入工作记忆
POST   /api/working-memory/flush  # 巩固到 LTM
GET    /api/consolidation/status  # 巩固状态
```

#### 阶段 2 端点
```
POST   /api/graph/traverse        # 图谱遍历
POST   /api/activate/contextual   # 情境感知召回
```

#### 阶段 3 端点
```
GET    /api/metacognition/health?vault={vault}
GET    /api/metacognition/coverage?vault={vault}&query={query}
GET    /api/metacognition/entropy?vault={vault}
POST   /api/metacognition/diagnose  # 全面诊断
```

### 2.3 REST API 使用示例

#### 情境感知召回
```bash
curl -X POST http://localhost:8475/api/activate/contextual \
  -H "Content-Type: application/json" \
  -d '{
    "context": ["debug payment"],
    "task": "fix retry logic bug",
    "goals": ["prevent double-charge", "maintain idempotency"],
    "cognitive_load": 0.7,
    "time_pressure": 0.5,
    "max_results": 10,
    "explain": true
  }'
```

**响应**:
```json
{
  "activations": [
    {
      "id": "01HXVWZ8K9...",
      "concept": "idempotency key conflict",
      "score": 0.89,
      "score_breakdown": {
        "base": 0.5,
        "hebbian": 0.2,
        "task_relevance": 0.15,
        "goal_alignment": 0.18,
        "complexity_penalty": -0.07,
        "confidence_bonus": 0.1
      },
      "relations": [
        {"type": "causal", "to": "payment failure", "weight": 0.8}
      ]
    }
  ]
}
```

#### 知识库健康诊断
```bash
curl http://localhost:8475/api/metacognition/health?vault=default
```

**响应**:
```json
{
  "vault": "default",
  "timestamp": "2026-03-29T12:00:00Z",
  "overall_health": 0.78,
  "metrics": {
    "coverage": 0.82,
    "confidence_entropy": 0.35,
    "freshness": 0.65,
    "association_density": 3.2,
    "prototype_coverage": 0.0
  },
  "alerts": [
    {
      "type": "low_freshness",
      "severity": "warning",
      "message": "65% of memories haven't been accessed in 30+ days",
      "recommendation": "Review and prune outdated memories"
    }
  ],
  "blind_spots": ["authentication", "deployment"],
  "recommendations": [
    "Add memories about PCI DSS requirements",
    "Document fraud detection workflow"
  ]
}
```

---

## 3. gRPC 服务集成

### 3.1 Proto 定义扩展

**文件**: `proto/muninn/v1/metacognition.proto` (待创建)

```protobuf
service MetacognitionService {
  // Get knowledge base health report
  rpc GetHealth(HealthRequest) returns (HealthReport);
  
  // Analyze knowledge coverage
  rpc GetCoverage(CoverageRequest) returns (CoverageReport);
  
  // Calculate confidence entropy
  rpc GetEntropy(EntropyRequest) returns (EntropyReport);
  
  // Comprehensive diagnosis
  rpc Diagnose(DiagnosisRequest) returns (DiagnosisReport);
}

message HealthRequest {
  string vault = 1;
}

message HealthReport {
  string vault = 1;
  double overall_health = 2;
  HealthMetrics metrics = 3;
  repeated HealthAlert alerts = 4;
  repeated string blind_spots = 5;
  repeated string recommendations = 6;
}

message HealthMetrics {
  double coverage = 1;
  double confidence_entropy = 2;
  double freshness = 3;
  double association_density = 4;
  double prototype_coverage = 5;
}
```

### 3.2 gRPC 客户端示例 (Go)

```go
package main

import (
    "context"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/scrypster/muninndb/proto/gen/go/muninn/v1"
)

func main() {
    conn, _ := grpc.Dial("localhost:8477", grpc.WithInsecure())
    defer conn.Close()
    
    client := pb.NewMetacognitionServiceClient(conn)
    
    // Get health report
    resp, _ := client.GetHealth(context.Background(), &pb.HealthRequest{
        Vault: "default",
    })
    
    fmt.Printf("Overall Health: %.1f%%\n", resp.OverallHealth*100)
    for _, alert := range resp.Alerts {
        fmt.Printf("[%s] %s\n", alert.Severity, alert.Message)
    }
}
```

---

## 4. SDK 集成

### 4.1 Go SDK

**文件**: `sdk/go/metacognition.go` (待创建)

```go
package muninnsdk

type MetacognitionClient struct {
    client pb.MetacognitionServiceClient
}

func (c *MetacognitionClient) Health(ctx context.Context, vault string) (*HealthReport, error) {
    resp, err := c.client.GetHealth(ctx, &pb.HealthRequest{Vault: vault})
    if err != nil {
        return nil, err
    }
    return convertHealthReport(resp), nil
}

func (c *MetacognitionClient) Coverage(ctx context.Context, vault, query string) (*CoverageReport, error) {
    resp, err := c.client.GetCoverage(ctx, &pb.CoverageRequest{
        Vault: vault,
        Query: query,
    })
    if err != nil {
        return nil, err
    }
    return convertCoverageReport(resp), nil
}
```

**使用示例**:
```go
import "github.com/scrypster/muninndb/sdk/go"

client := muninn.NewClient("http://localhost:8475")

// Get health report
health, _ := client.Metacognition.Health(ctx, "default")
fmt.Printf("Health: %.1f%%\n", health.OverallHealth*100)

// Analyze coverage
coverage, _ := client.Metacognition.Coverage(ctx, "default", "payment system")
fmt.Printf("Coverage: %.1f%%\n", coverage.EntityCoverage*100)
fmt.Printf("Blind spots: %v\n", coverage.BlindSpots)
```

### 4.2 Python SDK

**文件**: `sdk/python/muninn/metacognition.py` (待创建)

```python
from muninn import Client

client = Client("http://localhost:8475")

# Get health report
health = client.metacognition.health(vault="default")
print(f"Health: {health.overall_health*100:.1f}%")

# Analyze coverage
coverage = client.metacognition.coverage(
    vault="default",
    query="payment system architecture"
)
print(f"Coverage: {coverage.entity_coverage*100:.1f}%")
print(f"Blind spots: {coverage.blind_spots}")
```

---

## 5. 服务部署

### 5.1 本地开发

```bash
# 启动 MuninnDB 服务器
make run

# 或从源码运行
go run ./cmd/muninn/... start

# 验证服务
curl http://localhost:8475/api/health
```

### 5.2 Docker 部署

**Dockerfile** (待创建):
```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN make build

FROM ubuntu:22.04
COPY --from=builder /app/bin/muninn /usr/local/bin/
EXPOSE 8474 8475 8476 8477 8750
CMD ["muninn", "start"]
```

**docker-compose.yml**:
```yaml
version: '3.8'
services:
  muninn:
    image: muninndb:latest
    ports:
      - "8475:8475"  # REST
      - "8477:8477"  # gRPC
      - "8750:8750"  # MCP
    volumes:
      - muninn-data:/var/lib/muninn
    environment:
      - MUNINN_VAULT_DIR=/var/lib/muninn

volumes:
  muninn-data:
```

### 5.3 Kubernetes 部署

**Deployment** (待创建):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: muninndb
spec:
  replicas: 3
  selector:
    matchLabels:
      app: muninndb
  template:
    metadata:
      labels:
        app: muninndb
    spec:
      containers:
      - name: muninndb
        image: muninndb:latest
        ports:
        - containerPort: 8475
        - containerPort: 8477
        volumeMounts:
        - name: data
          mountPath: /var/lib/muninn
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: muninn-pvc
```

---

## 6. 监控与可观测性

### 6.1 Prometheus 指标

**新增指标**:
```promql
# 巩固速率
muninndb_consolidation_total

# 失败率
muninndb_consolidation_failures_total

# 延迟
muninndb_consolidation_latency_seconds

# 队列大小
muninndb_consolidation_queue_size

# 工作记忆淘汰率
muninndb_working_memory_evictions_total
```

### 6.2 Grafana 仪表板

**面板配置** (待创建):
- 巩固速率 (每分钟)
- 失败率 (%)
- 延迟 p95/p99
- 队列大小趋势
- 工作记忆活跃度

### 6.3 健康检查端点

```bash
# 基本健康检查
curl http://localhost:8475/api/health

# 详细健康报告
curl http://localhost:8475/api/metacognition/health?vault=default
```

---

## 7. 性能基准

### 7.1 操作延迟目标

| 操作 | 目标延迟 | 实测 | 状态 |
|------|---------|------|------|
| 工作记忆 Push | <5ms | <1ms | ✅ |
| 工作记忆 Flush | <10ms | <2ms | ✅ |
| DynamicForgetting | <1ms/engram | <0.1ms | ✅ |
| 巩固调度 | <50ms/job | <5ms | ✅ |
| GNN 遍历 (2-hop) | <50ms | - | 📝 待测 |
| 覆盖率分析 | <100ms | - | 📝 待测 |
| 熵计算 | <50ms | - | 📝 待测 |
| 健康诊断 | <500ms | - | 📝 待测 |

### 7.2 并发性能

```bash
# 并发测试
go test ./internal/engine -run "TestConcurrent" -race -v
# 100 goroutines, 5000+ operations, 0 race conditions ✅
```

---

## 8. 最佳实践

### 8.1 使用情境感知召回

```python
# 高认知负荷场景 (简化结果)
result = client.activate(
    query="payment bug",
    cognitive_load=0.9,  # 高负荷
    max_results=3        # 少而精
)

# 低时间压力场景 (深度分析)
result = client.activate(
    query="architecture design",
    time_pressure=0.2,   # 低压力
    max_results=10,      # 多结果
    explain=True         # 详细解释
)
```

### 8.2 定期健康检查

```python
# 每周运行健康诊断
health = client.metacognition.health(vault="default")

if health.overall_health < 0.5:
    print("⚠️  知识库需要维护")
    for alert in health.alerts:
        print(f"- {alert.message}")
```

### 8.3 工作记忆最佳实践

```python
# 将重要信息放入工作记忆
client.working_memory.push(engram)

# 复述以延长保持时间
for _ in range(3):
    client.working_memory.rehearse(engram_id)

# 手动巩固到长期记忆
client.working_memory.flush([engram_id])
```

---

## 9. 服务集成检查清单

### 阶段 1 服务集成
- [ ] MCP 工具实现 (`muninn_working_memory_*`)
- [ ] REST 端点实现 (`/api/working-memory/*`)
- [ ] Prometheus 指标暴露
- [ ] Grafana 仪表板配置

### 阶段 2 服务集成
- [ ] MCP 工具实现 (`muninn_graph_traverse`)
- [ ] REST 端点实现 (`/api/activate/contextual`)
- [ ] gRPC 服务实现
- [ ] SDK 封装 (Go/Python)

### 阶段 3 服务集成
- [ ] MCP 工具实现 (`muninn_metacognition_*`)
- [ ] REST 端点实现 (`/api/metacognition/*`)
- [ ] gRPC 服务实现 (`MetacognitionService`)
- [ ] SDK 封装
- [ ] 定期健康检查自动化

---

## 10. 下一步行动

### 立即可做 (本周)
1. [ ] 实现 MCP 工具包装器
2. [ ] 实现 REST API 端点
3. [ ] 编写服务集成文档

### 短期计划 (2 周)
1. [ ] 实现 gRPC 服务
2. [ ] 发布 SDK (Go/Python)
3. [ ] 配置 Prometheus + Grafana

### 中期计划 (1 个月)
1. [ ] Docker 镜像发布
2. [ ] Kubernetes Helm Chart
3. [ ] 性能基准测试报告

---

**服务状态**: ✅ 核心功能完成，准备服务集成  
**文档**: `/mnt/d/Docs/muninndb_phase{1,2,3}/`  
**下次审查**: 服务集成完成后
