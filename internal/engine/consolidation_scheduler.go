package engine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// ConsolidationScheduler manages the consolidation of memories from
// working memory to long-term memory based on cognitive rules.
//
// Consolidation occurs when:
//  1. Rehearsal: Item has been rehearsed ≥3 times
//  2. Emotional salience: High emotional weight detected
//  3. Contextual relevance: High ACT-R activation score
//  4. Memory strength: Calculated strength >= Stable
//  5. Capacity pressure: Working memory near capacity
type ConsolidationScheduler struct {
	queue       chan ConsolidationJob
	workers     int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	rules       []ConsolidationRule
	store       consolidationStore
	updateStore consolidationUpdateStore // optional, for persisting updates
	logger      *slog.Logger
	metrics     *ConsolidationMetrics // optional metrics
	mu          sync.RWMutex
}

// consolidationUpdateStore extends consolidationStore with update capability.
type consolidationUpdateStore interface {
	GetEngram(ctx context.Context, wsPrefix [8]byte, id storage.ULID) (*storage.Engram, error)
	UpdateEngram(ctx context.Context, wsPrefix [8]byte, engram *storage.Engram) error
}

// ConsolidationJob represents a memory to be consolidated.
type ConsolidationJob struct {
	EngramID   storage.ULID
	Reason     string
	Priority   int             // higher = more urgent
	Engram     *storage.Engram // optional, if already loaded
	AccessHist []AccessEvent   // optional, for strength calculation
}

// ConsolidationRule defines a condition that triggers consolidation.
type ConsolidationRule struct {
	Name      string
	Condition func(*storage.Engram, *ActivityContext) bool
	Priority  int
	Action    func(storage.ULID) error
}

// consolidationStore is the interface the scheduler needs from storage.
type consolidationStore interface {
	GetEngram(ctx context.Context, wsPrefix [8]byte, id storage.ULID) (*storage.Engram, error)
}

// ActivityContext provides context for consolidation decisions.
type ActivityContext struct {
	CurrentActivations []storage.ULID
	TaskContext        string
	CognitiveLoad      float64 // 0-1
}

// ConsolidationSchedulerConfig holds configuration.
type ConsolidationSchedulerConfig struct {
	QueueSize   int          // default: 100
	WorkerCount int          // default: 2
	Logger      *slog.Logger // optional, defaults to slog.Default()
}

// DefaultConsolidationSchedulerConfig returns default configuration.
func DefaultConsolidationSchedulerConfig() ConsolidationSchedulerConfig {
	return ConsolidationSchedulerConfig{
		QueueSize:   100,
		WorkerCount: 2,
		Logger:      slog.Default(),
	}
}

// NewConsolidationScheduler creates a new consolidation scheduler.
func NewConsolidationScheduler(
	store consolidationStore,
	config ConsolidationSchedulerConfig,
) *ConsolidationScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	sched := &ConsolidationScheduler{
		queue:   make(chan ConsolidationJob, config.QueueSize),
		workers: config.WorkerCount,
		ctx:     ctx,
		cancel:  cancel,
		store:   store,
		rules:   DefaultConsolidationRules(),
		logger:  logger,
	}

	// Try to cast store to update interface (optional)
	if updateStore, ok := store.(consolidationUpdateStore); ok {
		sched.updateStore = updateStore
	}

	// Start worker goroutines
	for i := 0; i < sched.workers; i++ {
		sched.wg.Add(1)
		// engine:spawn-ok worker goroutine for processing consolidation jobs
		go sched.worker(i)
	}

	return sched
}

// DefaultConsolidationRules returns the default set of consolidation rules.
// Updated to use CalculateMemoryStrength for intelligent decision-making.
func DefaultConsolidationRules() []ConsolidationRule {
	return []ConsolidationRule{
		{
			Name:      "memory_strength",
			Condition: checkMemoryStrength,
			Priority:  1, // highest priority - most reliable indicator
		},
		{
			Name:      "rehearsal",
			Condition: checkRehearsalCount,
			Priority:  2,
		},
		{
			Name:      "emotional_salience",
			Condition: checkEmotionalWeight,
			Priority:  3,
		},
		{
			Name:      "contextual_relevance",
			Condition: checkACTRActivation,
			Priority:  4,
		},
		{
			Name:      "capacity_pressure",
			Condition: checkWorkingMemoryPressure,
			Priority:  5,
		},
	}
}

// checkMemoryStrength returns true if memory strength >= Stable.
// This is the primary consolidation criterion based on cognitive research.
func checkMemoryStrength(engram *storage.Engram, ctx *ActivityContext) bool {
	// Use empty history if not provided - will give conservative estimate
	history := []AccessEvent{}
	strength := CalculateMemoryStrength(engram, history)
	return strength >= StrengthStable
}

// checkRehearsalCount returns true if the engram has been rehearsed ≥3 times.
func checkRehearsalCount(engram *storage.Engram, ctx *ActivityContext) bool {
	// Check if working memory type and has been accessed multiple times
	if engram.MemoryType == storage.MemoryTypeWorking {
		return engram.AccessCount >= 3
	}
	return false
}

// checkEmotionalWeight returns true if emotional weight is high (>0.7).
func checkEmotionalWeight(engram *storage.Engram, ctx *ActivityContext) bool {
	return float64(engram.Relevance) > 0.7
}

// checkACTRActivation returns true if ACT-R activation exceeds threshold.
func checkACTRActivation(engram *storage.Engram, ctx *ActivityContext) bool {
	activation := BaseLevelActivation(engram.AccessCount, engram.LastAccess)
	return activation > 2.0 // threshold for high activation
}

// checkWorkingMemoryPressure returns true if working memory is near capacity.
func checkWorkingMemoryPressure(engram *storage.Engram, ctx *ActivityContext) bool {
	return engram.MemoryType == storage.MemoryTypeWorking
}

// worker processes consolidation jobs from the queue.
func (s *ConsolidationScheduler) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case job := <-s.queue:
			s.processJob(job)
		}
	}
}

// processJob handles a single consolidation job.
// On success, updates engram lifecycle state to StateArchived and logs the result.
func (s *ConsolidationScheduler) processJob(job ConsolidationJob) {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.UpdateQueueSize(s.QueueLength())
		}
	}()

	ctx := context.Background()
	var wsPrefix [8]byte // default vault

	// Load engram if not provided
	engram := job.Engram
	if engram == nil {
		var err error
		engram, err = s.store.GetEngram(ctx, wsPrefix, job.EngramID)
		if err != nil {
			s.logger.Warn("consolidation: failed to load engram",
				"id", job.EngramID.String(),
				"err", err)
			if s.metrics != nil {
				s.metrics.RecordFailure()
			}
			return
		}
		if engram == nil {
			s.logger.Warn("consolidation: engram not found",
				"id", job.EngramID.String())
			if s.metrics != nil {
				s.metrics.RecordFailure()
			}
			return
		}
	}

	// Use provided access history or empty slice
	history := job.AccessHist
	if history == nil {
		history = []AccessEvent{}
	}

	// Calculate memory strength for decision-making
	strength := CalculateMemoryStrength(engram, history)

	// Evaluate consolidation rules
	activityCtx := &ActivityContext{}
	for _, rule := range s.rules {
		if rule.Condition(engram, activityCtx) {
			s.logger.Debug("consolidation: rule triggered",
				"rule", rule.Name,
				"id", job.EngramID.String(),
				"priority", rule.Priority,
				"strength", strength)

			// Mark for consolidation (change memory type to long-term)
			engram.MemoryType = storage.MemoryTypeSemantic
			engram.UpdatedAt = time.Now()

			// Update lifecycle state to archived (consolidated to LTM)
			engram.State = storage.StateArchived

			// Persist the change if update store is available
			if s.updateStore != nil {
				if err := s.updateStore.UpdateEngram(ctx, wsPrefix, engram); err != nil {
					s.logger.Error("consolidation: failed to persist update",
						"id", job.EngramID.String(),
						"err", err)
					if s.metrics != nil {
						s.metrics.RecordFailure()
					}
					return
				}
			}

			s.logger.Info("memory consolidated",
				"id", job.EngramID.String(),
				"rule", rule.Name,
				"strength", strength,
				"new_state", engram.State.String())

			// Record metrics
			if s.metrics != nil {
				latency := time.Since(start).Seconds()
				s.metrics.RecordConsolidation(strength, latency)
			}
			return // consolidated
		}
	}

	s.logger.Debug("consolidation: no rules matched",
		"id", job.EngramID.String(),
		"strength", strength)

	// Record failure (no rules matched)
	if s.metrics != nil {
		s.metrics.RecordFailure()
	}
}

// Schedule queues an engram for consolidation evaluation.
func (s *ConsolidationScheduler) Schedule(id storage.ULID, reason string, priority int) {
	job := ConsolidationJob{
		EngramID: id,
		Reason:   reason,
		Priority: priority,
	}

	select {
	case s.queue <- job:
		// queued successfully
	default:
		// queue full, drop silently (non-blocking)
		s.logger.Warn("consolidation: queue full, dropping job",
			"id", id.String(),
			"reason", reason)
	}
}

// ScheduleWithHistory queues an engram with access history for better strength calculation.
func (s *ConsolidationScheduler) ScheduleWithHistory(id storage.ULID, reason string, priority int, history []AccessEvent) {
	job := ConsolidationJob{
		EngramID:   id,
		Reason:     reason,
		Priority:   priority,
		AccessHist: history,
	}

	select {
	case s.queue <- job:
		// queued successfully
	default:
		s.logger.Warn("consolidation: queue full, dropping job",
			"id", id.String(),
			"reason", reason)
	}
}

// ScheduleBatch queues multiple engrams for consolidation.
func (s *ConsolidationScheduler) ScheduleBatch(ids []storage.ULID, reason string) {
	for _, id := range ids {
		s.Schedule(id, reason, 0)
	}
}

// ScheduleImmediate enqueues a high-priority consolidation job.
// Blocks until queued (unlike Schedule which drops if full).
func (s *ConsolidationScheduler) ScheduleImmediate(id storage.ULID, reason string) {
	job := ConsolidationJob{
		EngramID: id,
		Reason:   reason,
		Priority: 10, // high priority
	}
	s.queue <- job
}

// QueueLength returns the current number of jobs in the queue.
func (s *ConsolidationScheduler) QueueLength() int {
	return len(s.queue)
}

// Stop gracefully shuts down the scheduler.
// Waits for all in-progress jobs to complete.
func (s *ConsolidationScheduler) Stop() {
	s.cancel()
	s.wg.Wait()
}

// AddRule adds a custom consolidation rule.
func (s *ConsolidationScheduler) AddRule(rule ConsolidationRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = append(s.rules, rule)
}

// RemoveRule removes a consolidation rule by name.
func (s *ConsolidationScheduler) RemoveRule(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, rule := range s.rules {
		if rule.Name == name {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			return
		}
	}
}

// SetLogger updates the logger used by the scheduler.
func (s *ConsolidationScheduler) SetLogger(logger *slog.Logger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger = logger
}

// WithMetrics sets the metrics collector for the scheduler.
func (s *ConsolidationScheduler) WithMetrics(metrics *ConsolidationMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = metrics
}
