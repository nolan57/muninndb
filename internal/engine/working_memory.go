package engine

import (
	"sync"
	"time"

	"github.com/scrypster/muninndb/internal/storage"
)

// WorkingMemoryBuffer implements a short-term cognitive buffer with
// capacity limits (Miller's Law: 7±2) and time-based decay.
//
// Working memory holds items currently being cognitively processed.
// Items decay after a configurable duration unless rehearsed.
// When capacity is exceeded or items decay, they are flushed to LTM.
type WorkingMemoryBuffer struct {
	items         []*storage.Engram
	maxSize       int           // default: 7 (Miller's Law)
	decayTime     time.Duration // default: 30s without rehearsal
	priorityEvict bool          // if true, evict low-relevance items first
	mu            sync.RWMutex
	itemExpiry    map[storage.ULID]time.Time // tracks when each item expires
	metrics       *ConsolidationMetrics      // optional metrics
}

// WMConfig holds configuration for WorkingMemoryBuffer.
type WMConfig struct {
	MaxSize       int           // maximum capacity (default: 7)
	DecayTime     time.Duration // decay timeout (default: 30s)
	PriorityEvict bool          // if true, use priority-based eviction (default: true)
}

// DefaultWMConfig returns the default working memory configuration.
func DefaultWMConfig() WMConfig {
	return WMConfig{
		MaxSize:       7, // Miller's Law
		DecayTime:     30 * time.Second,
		PriorityEvict: true, // priority-based eviction enabled by default
	}
}

// NewWorkingMemoryBuffer creates a new working memory buffer.
func NewWorkingMemoryBuffer(config WMConfig) *WorkingMemoryBuffer {
	if config.MaxSize <= 0 {
		config.MaxSize = 7
	}
	if config.DecayTime <= 0 {
		config.DecayTime = 30 * time.Second
	}
	return &WorkingMemoryBuffer{
		items:         make([]*storage.Engram, 0, config.MaxSize),
		maxSize:       config.MaxSize,
		decayTime:     config.DecayTime,
		priorityEvict: config.PriorityEvict,
		itemExpiry:    make(map[storage.ULID]time.Time),
	}
}

// Push adds an engram to working memory.
// If the buffer is at capacity, evicts based on priority strategy:
//   - If PriorityEviction is true: evicts lowest Relevance item (LRU as tiebreaker)
//   - If PriorityEviction is false: evicts oldest item (pure LRU)
//
// Returns the evicted item (if any) for consolidation.
func (wm *WorkingMemoryBuffer) Push(engram *storage.Engram) *storage.Engram {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Set expiry time
	wm.itemExpiry[engram.ID] = time.Now().Add(wm.decayTime)

	// Check if already in buffer (update position)
	for i, item := range wm.items {
		if item.ID == engram.ID {
			// Move to end (most recent)
			wm.items = append(wm.items[:i], wm.items[i+1:]...)
			wm.items = append(wm.items, engram)
			return nil
		}
	}

	// Check capacity
	var evicted *storage.Engram
	if len(wm.items) >= wm.maxSize {
		evicted = wm.evictItem()
	}

	// Add to end
	wm.items = append(wm.items, engram)
	return evicted
}

// evictItem selects and removes an item for eviction based on configured strategy.
// Must be called with wm.mu held.
func (wm *WorkingMemoryBuffer) evictItem() *storage.Engram {
	if len(wm.items) == 0 {
		return nil
	}

	if !wm.priorityEvict {
		// Pure LRU: evict oldest (first item)
		evicted := wm.items[0]
		wm.items = wm.items[1:]
		delete(wm.itemExpiry, evicted.ID)
		if wm.metrics != nil {
			wm.metrics.RecordEviction()
		}
		return evicted
	}

	// Priority-based eviction: find lowest relevance item
	evictIdx := 0
	lowestRelevance := wm.items[0].Relevance

	for i := 1; i < len(wm.items); i++ {
		if wm.items[i].Relevance < lowestRelevance {
			lowestRelevance = wm.items[i].Relevance
			evictIdx = i
		}
	}

	evicted := wm.items[evictIdx]
	wm.items = append(wm.items[:evictIdx], wm.items[evictIdx+1:]...)
	delete(wm.itemExpiry, evicted.ID)
	if wm.metrics != nil {
		wm.metrics.RecordEviction()
	}
	return evicted
}

// Rehearse extends the decay time for an item (prevents forgetting).
// Returns true if the item was found and rehearsed.
func (wm *WorkingMemoryBuffer) Rehearse(id storage.ULID) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.itemExpiry[id]; exists {
		wm.itemExpiry[id] = time.Now().Add(wm.decayTime)
		if wm.metrics != nil {
			wm.metrics.RecordRehearsal()
		}
		return true
	}
	return false
}

// GetActive returns all non-expired items in working memory.
// Expired items are removed and returned separately for consolidation.
func (wm *WorkingMemoryBuffer) GetActive() (active, expired []*storage.Engram) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	now := time.Now()
	active = make([]*storage.Engram, 0, len(wm.items))

	for _, item := range wm.items {
		if expiry, exists := wm.itemExpiry[item.ID]; exists {
			if expiry.Before(now) {
				expired = append(expired, item)
				delete(wm.itemExpiry, item.ID)
			} else {
				active = append(active, item)
			}
		}
	}

	wm.items = active
	return active, expired
}

// Flush removes all items from working memory for consolidation to LTM.
// Returns all flushed items.
func (wm *WorkingMemoryBuffer) Flush() []*storage.Engram {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	flushed := wm.items
	wm.items = make([]*storage.Engram, 0, wm.maxSize)
	wm.itemExpiry = make(map[storage.ULID]time.Time)
	return flushed
}

// FlushByID removes specific items by ID for consolidation.
// Returns the flushed items.
func (wm *WorkingMemoryBuffer) FlushByID(ids []storage.ULID) []*storage.Engram {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	toFlush := make(map[storage.ULID]struct{})
	for _, id := range ids {
		toFlush[id] = struct{}{}
	}

	var flushed []*storage.Engram
	remaining := make([]*storage.Engram, 0, len(wm.items))

	for _, item := range wm.items {
		if _, shouldFlush := toFlush[item.ID]; shouldFlush {
			flushed = append(flushed, item)
			delete(wm.itemExpiry, item.ID)
		} else {
			remaining = append(remaining, item)
		}
	}

	wm.items = remaining
	return flushed
}

// Count returns the current number of items in working memory.
func (wm *WorkingMemoryBuffer) Count() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.items)
}

// Capacity returns the maximum capacity of the working memory buffer.
func (wm *WorkingMemoryBuffer) Capacity() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.maxSize
}

// Contains returns true if the item is in working memory.
func (wm *WorkingMemoryBuffer) Contains(id storage.ULID) bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	_, exists := wm.itemExpiry[id]
	return exists
}

// Get returns an item by ID without removing it.
func (wm *WorkingMemoryBuffer) Get(id storage.ULID) *storage.Engram {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	for _, item := range wm.items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

// GetAll returns all items currently in working memory.
func (wm *WorkingMemoryBuffer) GetAll() []*storage.Engram {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	result := make([]*storage.Engram, len(wm.items))
	copy(result, wm.items)
	return result
}

// WithMetrics sets the metrics collector for the working memory buffer.
func (wm *WorkingMemoryBuffer) WithMetrics(metrics *ConsolidationMetrics) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.metrics = metrics
}
