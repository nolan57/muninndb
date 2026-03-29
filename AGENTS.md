# MuninnDB — Agent Instructions

## Build & Test Commands

### Quick Start
```bash
# Run from source
go run ./cmd/muninn/... start

# Download ML assets (required for embedder)
make fetch-assets
```

### Building
```bash
make build                              # Full build (web + server)
go build -tags localassets ./cmd/muninn/...
make web                                # Web UI only
make css                                # CSS bundle only (faster)
```

### Testing
```bash
# Run all tests
make test
go test ./...

# Run single test (add -tags localassets if test needs ML models)
go test -v -run TestName ./package/path
# Example: go test -v -run TestOpen_Remember_Recall .
# Example: go test -v -run TestEmbedder_Basic -tags localassets ./internal/plugin/embed/...

# Tests with coverage
go test -tags localassets ./... -timeout 300s -race -coverprofile=coverage.out

# Integration tests (full lifecycle: start/stop/restart)
make test-integration
go test -tags integration -v -timeout 120s ./cmd/muninn/...

# Benchmarks
make bench
go test -bench=BenchmarkE2E -benchmem -benchtime=3s ./internal/bench/...
```

### SDK Tests
```bash
# Python
pip install -r sdk/python/requirements.txt
cd sdk/python && PYTHONPATH=. pytest tests/test_sdk_smoke.py -v --timeout=30

# Node.js
cd sdk/node && npm install && npm test

# Web (Vitest + Playwright)
cd web && npm ci && npm test
npm run test:e2e                        # Playwright E2E
```

### Linting & Formatting
```bash
go fmt ./...                            # Format Go code
go vet ./...                            # Static analysis
gofmt -d .                              # Check formatting diffs
```

---

## Code Style

### Go

**Imports:** Standard library → external → internal. Group 3+ in parentheses.
```go
import (
  "context"
  "fmt"

  "github.com/google/uuid"

  "github.com/scrypster/muninndb/internal/engine"
)
```

**Formatting:** `gofmt` required. Tabs, ~120 char lines.

**Naming:** Exported=PascalCase, unexported=camelCase. Acronyms=all caps (API, DB, MCP).

**Error Handling:** Wrap with context, check immediately.
```go
if err != nil {
  return nil, fmt.Errorf("muninndb: open pebble: %w", err)
}
```

**Testing:** Use `t.Helper()`, `t.Cleanup()`. Async: retry loops, not `time.Sleep()`.
```go
func openTestDB(t *testing.T) *muninn.DB {
  t.Helper()
  db, err := muninn.Open(t.TempDir())
  if err != nil { t.Fatalf("Open: %v", err) }
  t.Cleanup(func() { _ = db.Close() })
  return db
}
```

**Context:** First param. Use `context.Background()` in tests.

**Comments:** No comments in code unless necessary for complex logic.

### TypeScript (Web)
- 2-space indent, semicolons, single quotes
- Components: PascalCase, files: lowercase-dash.tsx
- Use TypeScript, avoid `any`
- Build: `npm run build`, Test: `npm test`, E2E: `npm run test:e2e`

### Python (SDK)
- PEP 8, type hints, 4-space indent

### General
- **No emojis** in code or comments unless user explicitly requests
- **Concise output** — avoid verbose explanations in code
- **Security first** — never expose or log secrets/keys

---

## Git Workflow

**Branches:** `main` (releases), `develop` (integration), `feature/*`, `fix/*`, `bug/*`

```bash
git checkout develop && git pull origin develop
git checkout -b feature/my-feature
git push -u origin feature/my-feature
# Open PR into develop
```

**Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`)
```
feat: add semantic trigger subscriptions

- Add POST /api/triggers endpoint
- Includes vault-scoped isolation

Fixes #142
```

**PR Flow:**
1. Branch from `develop`
2. Work and commit
3. Open PR into `develop` (not `main`)
4. CI must pass before merge
5. Never push directly to `develop` or `main`

**NEVER commit unless user explicitly asks.**

---

## Architecture

**Key Directories:**
- `cmd/muninn/` — CLI, server startup
- `internal/engine/` — Cognitive engine
- `internal/storage/` — Pebble DB persistence
- `internal/index/` — HNSW vector, FTS
- `sdk/` — Go, Python, Node, PHP SDKs
- `web/` — Vite + React UI

**Ports:** 8474 (MBP), 8475 (REST), 8476 (Web), 8477 (gRPC), 8750 (MCP)

**Core:** Engram (memory), Vault (isolation), Activation (6-phase retrieval)

---

## AI Agent Integration

**MCP Config:**
```json
{
  "mcpServers": {
    "muninn": {
      "url": "http://127.0.0.1:8750/mcp"
    }
  }
}
```
*Omit `"type"` for Claude Desktop v1.1.4010+*

**MCP Tools (35 total):** `muninn_remember`, `muninn_recall`, `muninn_search`, `muninn_read`, `muninn_forget`, `muninn_link`, `muninn_guide`

First connection: call `muninn_guide` for vault instructions.

---

## CI/CD

**Workflows:** Build/test (Ubuntu/Windows), CLI integration, Python SDK, Playwright E2E, Shellcheck, API spec validation

**Release:**
```bash
git tag v0.2.4 && git push origin v0.2.4
```

---

## Troubleshooting

**Tests fail:** `make fetch-assets && go test -tags localassets ./...`

**ORT error:** `make fetch-ort-libs`

**Port in use:** `lsof -i :8475 && kill <PID>`

**Windows DLL error:** Install VC++ 2019+ Redistributable

---

**Docs:** https://muninndb.com/docs | **Contributing:** CONTRIBUTING.md
