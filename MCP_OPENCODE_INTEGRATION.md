# MuninnDB MCP 元认知工具 - opencode 集成验证

**日期**: 2026-03-29  
**状态**: ✅ MCP 工具已注册，准备验证

---

## 已实施的 MCP 工具

### 新增工具 (3 个)

| 工具名称 | 类型 | 描述 |
|---------|------|------|
| `muninn_metacognition_health` | 只读 | 获取知识库健康报告 |
| `muninn_metacognition_coverage` | 只读 | 分析特定查询的覆盖率 |
| `muninn_metacognition_entropy` | 只读 | 计算置信熵 |

### 工具定义位置
- **工具定义**: `internal/mcp/tools.go` (第 718-752 行)
- **处理程序**: `internal/mcp/metacognition_handlers.go`
- **路由注册**: `internal/mcp/server.go` (第 278-280 行)
- **权限控制**: `internal/mcp/context.go` (第 198-200 行)

---

## opencode 集成验证

### 方法 1: 使用 MCP CLI 测试

```bash
# 1. 启动 MuninnDB 服务器
cd /home/urio/Documents/muninndb
go run ./cmd/muninn/... start

# 2. 使用 MCP CLI 测试工具
mcp call muninn_metacognition_health --vault=default

# 3. 测试覆盖率分析
mcp call muninn_metacognition_coverage \
  --vault=default \
  --query="payment system architecture"

# 4. 测试熵分析
mcp call muninn_metacognition_entropy --vault=default
```

### 方法 2: 使用 JSON-RPC 直接调用

```bash
# 启动服务器后，使用 netcat 测试
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"muninn_metacognition_health","arguments":{}}}' | nc localhost 8750
```

**预期响应**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "overall_health": 0.78,
    "metrics": {
      "coverage": 0.82,
      "confidence_entropy": 0.35,
      "freshness": 0.65
    },
    "alerts": [...],
    "recommendations": [...]
  }
}
```

### 方法 3: opencode 配置集成

**opencode 配置文件** (`~/.opencode/config.json`):
```json
{
  "mcpServers": {
    "muninn": {
      "command": "go",
      "args": ["run", "./cmd/muninn/...", "start"],
      "cwd": "/home/urio/Documents/muninndb",
      "env": {
        "MUNINN_VAULT_DIR": "/tmp/muninn"
      }
    }
  }
}
```

**opencode 使用示例**:
```bash
# 启动 opencode 与 MuninnDB MCP 服务器
opencode

# 在 opencode 会话中调用工具
> @muninn metacognition_health

> @muninn metacognition_coverage for "payment system"
```

---

## 验证步骤

### 步骤 1: 验证工具定义

```bash
cd /home/urio/Documents/muninndb
go run ./cmd/muninn/... &
sleep 3

# 获取工具列表
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | nc localhost 8750 | jq '.result.tools[] | select(.name | contains("metacognition"))'
```

**预期输出**:
```json
{
  "name": "muninn_metacognition_health",
  "description": "Get comprehensive knowledge base health report..."
}
{
  "name": "muninn_metacognition_coverage",
  "description": "Analyze knowledge coverage for a specific query..."
}
{
  "name": "muninn_metacognition_entropy",
  "description": "Calculate confidence entropy..."
}
```

### 步骤 2: 验证工具调用

```bash
# 测试健康诊断
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"muninn_metacognition_health","arguments":{"vault":"default"}}}' | nc localhost 8750 | jq .
```

### 步骤 3: 验证与 opencode 集成

```bash
# 1. 配置 opencode
cat > ~/.opencode/config.json << 'EOF'
{
  "mcpServers": {
    "muninn": {
      "url": "http://localhost:8750/mcp"
    }
  }
}
EOF

# 2. 启动 MuninnDB
cd /home/urio/Documents/muninndb && go run ./cmd/muninn/... start &

# 3. 启动 opencode
opencode

# 4. 在 opencode 中测试
> 请帮我分析 default 知识库的健康状况
> 请检查关于 "payment" 主题的知识覆盖率
```

---

## 故障排查

### 问题 1: 工具未找到

**错误**: `unknown tool: muninn_metacognition_health`

**解决**:
1. 验证工具定义已添加到 `internal/mcp/tools.go`
2. 验证处理程序已注册到 `internal/mcp/server.go`
3. 重新编译：`go build ./internal/mcp/...`

### 问题 2: 处理程序编译错误

**错误**: `undefined: handleMetacognitionHealth`

**解决**:
1. 确认 `internal/mcp/metacognition_handlers.go` 存在
2. 验证函数签名正确
3. 重新编译：`go build ./...`

### 问题 3: opencode 连接失败

**错误**: `Failed to connect to MCP server`

**解决**:
1. 检查 MuninnDB 服务器是否运行：`pgrep -f muninn`
2. 验证端口 8750 未被占用：`lsof -i :8750`
3. 检查 opencode 配置路径正确

---

## 测试脚本

**自动测试脚本**: `/tmp/test_mcp_metacognition.sh`

**使用方法**:
```bash
# 启动服务器
cd /home/urio/Documents/muninndb && go run ./cmd/muninn/... start &
sleep 5

# 运行测试
/tmp/test_mcp_metacognition.sh
```

---

## 性能基准

| 操作 | 目标延迟 | 实测 | 状态 |
|------|---------|------|------|
| Health Analysis | <500ms | - | 📝 待测 |
| Coverage Analysis | <100ms | - | 📝 待测 |
| Entropy Calculation | <50ms | - | 📝 待测 |

---

## 下一步

1. ✅ MCP 工具定义已添加
2. ✅ 处理程序已注册
3. ✅ 编译通过
4. ⏳ 运行集成测试
5. ⏳ 验证与 opencode 集成
6. ⏳ 性能基准测试

---

**状态**: 🚧 准备集成验证  
**编译**: ✅ 通过  
**下一步**: 启动 MuninnDB 并测试 MCP 工具调用
