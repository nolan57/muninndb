package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ConsolidationMetrics holds Prometheus metrics for consolidation monitoring.
type ConsolidationMetrics struct {
	// consolidationTotal tracks total successful consolidations
	consolidationTotal prometheus.Counter

	// consolidationFailures tracks failed consolidation attempts
	consolidationFailures prometheus.Counter

	// consolidationLatency tracks time taken for consolidation processing (seconds)
	consolidationLatency prometheus.Histogram

	// queueSize tracks current queue length
	queueSize prometheus.Gauge

	// strengthDistribution tracks distribution of memory strength at consolidation
	strengthDistribution prometheus.Histogram

	// evictionTotal tracks working memory evictions
	evictionTotal prometheus.Counter

	// rehearsalTotal tracks working memory rehearsals
	rehearsalTotal prometheus.Counter
}

// NewConsolidationMetrics creates and registers consolidation metrics.
func NewConsolidationMetrics() *ConsolidationMetrics {
	return &ConsolidationMetrics{
		consolidationTotal: promauto.NewCounter(prometheus.CounterOpts{
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

		evictionTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "muninndb",
			Subsystem: "working_memory",
			Name:      "evictions_total",
			Help:      "Total number of working memory evictions",
		}),

		rehearsalTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "muninndb",
			Subsystem: "working_memory",
			Name:      "rehearsals_total",
			Help:      "Total number of working memory rehearsals",
		}),
	}
}

// RecordConsolidation records a successful consolidation event.
func (m *ConsolidationMetrics) RecordConsolidation(strength MemoryStrength, latency float64) {
	if m == nil {
		return
	}
	m.consolidationTotal.Inc()
	m.consolidationLatency.Observe(latency)
	m.strengthDistribution.Observe(float64(strength) / 4.0) // Normalize to 0-1
}

// RecordFailure records a failed consolidation event.
func (m *ConsolidationMetrics) RecordFailure() {
	if m == nil {
		return
	}
	m.consolidationFailures.Inc()
}

// RecordEviction records a working memory eviction.
func (m *ConsolidationMetrics) RecordEviction() {
	if m == nil {
		return
	}
	m.evictionTotal.Inc()
}

// RecordRehearsal records a working memory rehearsal.
func (m *ConsolidationMetrics) RecordRehearsal() {
	if m == nil {
		return
	}
	m.rehearsalTotal.Inc()
}

// UpdateQueueSize updates the current queue size gauge.
func (m *ConsolidationMetrics) UpdateQueueSize(size int) {
	if m == nil {
		return
	}
	m.queueSize.Set(float64(size))
}
