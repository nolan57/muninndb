# MuninnDB 认知引擎增强补丁 (v2)

**补丁版本**: v2  
**日期**: 2026-03-29  
**状态**: 待实施

---

## 改进 1: 暴露配置参数 (消除魔法数字)

### File: `internal/engine/dynamic_forgetting.go`

**Change**: 将魔法数字移至可配置结构体

```go
// SpacingEffectConfig holds configuration for spacing effect calculation.
type SpacingEffectConfig struct {
	// MinBonus is the minimum bonus floor (default: 0.7)
	MinBonus float64
	// MaxBonus is the maximum bonus ceiling (default: 1.3)
	MaxBonus float64
	// StabilityPen0 is bonus for 0 accesses (default: 0.7)
	StabilityPen0 float64
	// StabilityPen1 is bonus for 1 access (default: 0.9)
	StabilityPen1 float64
	// CVCoefficient is the coefficient for CV bonus (default: 0.2)
	CVCoefficient float64
	// TrendCoefficient is the coefficient for trend bonus (default: 0.3)
	TrendCoefficient float64
}

// DefaultSpacingEffectConfig returns default spacing effect configuration.
func DefaultSpacingEffectConfig() SpacingEffectConfig {
	return SpacingEffectConfig{
		MinBonus:        0.7,
		MaxBonus:        1.3,
		StabilityPen0:   0.7,
		StabilityPen1:   0.9,
		CVCoefficient:   0.2,
		TrendCoefficient: 0.3,
	}
}

// SpacingEffectBonusWithConfig calculates bonus with custom configuration.
func SpacingEffectBonusWithConfig(history []AccessEvent, config SpacingEffectConfig) float64 {
	historyLen := len(history)

	// Short history handling: apply stability penalty for new memories
	if historyLen < 2 {
		// 0 accesses = StabilityPen0, 1 access = StabilityPen1
		if historyLen == 0 {
			return config.StabilityPen0
		}
		return config.StabilityPen1
	}

	// ... (existing interval calculation logic)

	cvBonus := math.Min(cv, 1.5) * config.CVCoefficient
	trendBonus := increasingTrend * config.TrendCoefficient

	bonus := 1.0 + cvBonus + trendBonus

	// Clamp to [MinBonus, MaxBonus] range
	if bonus < config.MinBonus {
		return config.MinBonus
	}
	if bonus > config.MaxBonus {
		return config.MaxBonus
	}
	return bonus
}

// SpacingEffectBonus uses default configuration for backward compatibility.
func SpacingEffectBonus(history []AccessEvent) float64 {
	return SpacingEffectBonusWithConfig(history, DefaultSpacingEffectConfig())
}
```

---

## 改进 2: 并发压力测试

### File: `internal/engine/cognitive_e2e_test.go`

**Add**: 并发压力测试

```go
// TestConcurrentPushRehearse tests concurrent access to WorkingMemoryBuffer.
func TestConcurrentPushRehearse(t *testing.T) {
	config := WMConfig{
		MaxSize:       10,
		DecayTime:     30 * time.Second,
		PriorityEvict: true,
	}
	wm := NewWorkingMemoryBuffer(config)

	var wg sync.WaitGroup
	numGoroutines := 100
	opsPerGoroutine := 50

	// Track successful operations
	var pushCount, rehearseCount atomic.Int64

	// Launch 100 concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < opsPerGoroutine; j++ {
				// Create test engram
				eng := &storage.Engram{
					ID:        storage.NewULID(),
					Relevance: float32(rand.Float64()),
				}

				// Push to working memory
				wm.Push(eng)
				pushCount.Add(1)

				// Randomly rehearse some items
				if rand.Float64() < 0.3 {
					wm.Rehearse(eng.ID)
					rehearseCount.Add(1)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify no race conditions (would panic if detected)
	t.Logf("✓ Concurrent test passed: %d pushes, %d rehearses completed",
		pushCount.Load(), rehearseCount.Load())

	// Verify buffer is within capacity
	if wm.Count() > wm.Capacity() {
		t.Errorf("Working memory exceeded capacity: got %d, want ≤%d",
			wm.Count(), wm.Capacity())
	}
}

// TestConcurrentConsolidation tests concurrent scheduling and processing.
func TestConcurrentConsolidation(t *testing.T) {
	store := newMockConsolidationStore()
	config := DefaultConsolidationSchedulerConfig()
	config.WorkerCount = 4 // Multiple workers for concurrency
	sched := NewConsolidationScheduler(store, config)
	defer sched.Stop()

	var wg sync.WaitGroup
	numGoroutines := 50
	schedulesPerGoroutine := 20

	var scheduledCount atomic.Int64

	// Launch concurrent schedulers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < schedulesPerGoroutine; j++ {
				// Create engram in store
				engram := store.createTestEngram(byte(id%256), storage.MemoryTypeWorking)
				engram.AccessCount = uint32(10 + j) // Ensure consolidation

				// Schedule consolidation
				sched.Schedule(engram.ID, "concurrent_test", 0)
				scheduledCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	// Allow time for processing
	time.Sleep(2 * time.Second)

	t.Logf("✓ Concurrent consolidation test passed: %d schedules completed",
		scheduledCount.Load())
}
```

---

## 改进 3: Prometheus 监控指标

### File: `internal/engine/consolidation_metrics.go` (新增)

**Add**: Prometheus metrics for consolidation monitoring

```go
package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ConsolidationMetrics holds Prometheus metrics for consolidation monitoring.
type ConsolidationMetrics struct {
	// consolidationRate tracks the rate of successful consolidations per second
	consolidationRate prometheus.Counter

	// consolidationFailures tracks failed consolidation attempts
	consolidationFailures prometheus.Counter

	// consolidationLatency tracks time taken for consolidation processing
	consolidationLatency prometheus.Histogram

	// queueSize tracks current queue length
	queueSize prometheus.Gauge

	// strengthDistribution tracks distribution of memory strength at consolidation
	strengthDistribution prometheus.Histogram

	// evictionRate tracks working memory eviction rate
	evictionRate prometheus.Counter
}

// NewConsolidationMetrics creates and registers consolidation metrics.
func NewConsolidationMetrics() *ConsolidationMetrics {
	return &ConsolidationMetrics{
		consolidationRate: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "muninndb",
			Subsystem: "consolidation",
			Name:      "total",
			Help:      "Total number of successful consolidations",
		}),

		consolidationFailures: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "muninndb",
			Subsystem: "consolidation",
			Name:      "failures_total",
			Help:      "Total number of failed consolidations",
		}),

		consolidationLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "muninndb",
			Subsystem: "consolidation",
			Name:      "latency_seconds",
			Help:      "Latency of consolidation processing",
			Buckets:   prometheus.DefBuckets,
		}),

		queueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "muninndb",
			Subsystem: "consolidation",
			Name:      "queue_size",
			Help:      "Current size of consolidation queue",
		}),

		strengthDistribution: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: "muninndb",
			Subsystem: "consolidation",
			Name:      "strength_distribution",
			Help:      "Distribution of memory strength at consolidation time",
			Buckets:   []float64{0, 0.25, 0.5, 0.75, 1.0},
		}),

		evictionRate: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "muninndb",
			Subsystem: "working_memory",
			Name:      "evictions_total",
			Help:      "Total number of working memory evictions",
		}),
	}
}

// RecordConsolidation records a successful consolidation event.
func (m *ConsolidationMetrics) RecordConsolidation(strength MemoryStrength, latency float64) {
	m.consolidationRate.Inc()
	m.consolidationLatency.Observe(latency)
	m.strengthDistribution.Observe(float64(strength) / 4.0) // Normalize to 0-1
}

// RecordFailure records a failed consolidation event.
func (m *ConsolidationMetrics) RecordFailure() {
	m.consolidationFailures.Inc()
}

// RecordEviction records a working memory eviction.
func (m *ConsolidationMetrics) RecordEviction() {
	m.evictionRate.Inc()
}

// UpdateQueueSize updates the current queue size gauge.
func (m *ConsolidationMetrics) UpdateQueueSize(size int) {
	m.queueSize.Set(float64(size))
}

// ConsolidationScheduler integration
func (s *ConsolidationScheduler) WithMetrics(metrics *ConsolidationMetrics) {
	s.metrics = metrics
}

// In processJob(), add metrics recording:
func (s *ConsolidationScheduler) processJob(job ConsolidationJob) {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.UpdateQueueSize(s.QueueLength())
		}
	}()

	// ... existing processing logic ...

	// On success:
	if s.metrics != nil {
		latency := time.Since(start).Seconds()
		s.metrics.RecordConsolidation(strength, latency)
	}

	// On failure:
	if s.metrics != nil {
		s.metrics.RecordFailure()
	}
}
```

---

## 改进 4: WorkingMemory 指标集成

### File: `internal/engine/working_memory.go`

**Add**: Metrics integration

```go
type WorkingMemoryBuffer struct {
	// ... existing fields ...
	metrics *ConsolidationMetrics // optional
}

func (wm *WorkingMemoryBuffer) WithMetrics(metrics *ConsolidationMetrics) {
	wm.metrics = metrics
}

func (wm *WorkingMemoryBuffer) Push(engram *storage.Engram) *storage.Engram {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// ... existing logic ...

	// Record eviction if occurred
	if evicted != nil && wm.metrics != nil {
		wm.metrics.RecordEviction()
	}

	return evicted
}
```

---

## 验证命令

```bash
# 运行并发测试
go test ./internal/engine/... \
  -run "TestConcurrent" \
  -race \
  -v

# 验证 Prometheus 指标
curl http://localhost:8475/metrics | grep muninndb_consolidation
```

---

## Prometheus 仪表板示例

```promql
# Consolidation rate (per minute)
rate(muninndb_consolidation_total[1m]) * 60

# Consolidation failure rate (%)
rate(muninndb_consolidation_failures_total[5m]) / 
  (rate(muninndb_consolidation_total[5m]) + rate(muninndb_consolidation_failures_total[5m])) * 100

# Working memory eviction rate (per minute)
rate(muninndb_working_memory_evictions_total[1m]) * 60

# Consolidation latency (p95)
histogram_quantile(0.95, rate(muninndb_consolidation_latency_seconds_bucket[5m]))

# Queue size over time
muninndb_consolidation_queue_size
```

---

*补丁版本：v2*  
*状态：待实施*  
*下一步：代码审查 → 合并*
