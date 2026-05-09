package mcp

import (
	"fmt"
	"sync"
	"time"
)

// SpaceWriteLock serializes write operations within a space.
// This prevents race conditions when multiple MCP clients concurrently
// modify pages in the same space.
type SpaceWriteLock struct {
	mu      sync.Mutex
	locks   map[string]*spaceLock
	timeout time.Duration
}

type spaceLock struct {
	mu       sync.Mutex
	held     bool
	unlocked chan struct{}
}

// NewSpaceWriteLock creates a write lock with the given timeout.
func NewSpaceWriteLock(timeout time.Duration) *SpaceWriteLock {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &SpaceWriteLock{
		locks:   make(map[string]*spaceLock),
		timeout: timeout,
	}
}

// Lock acquires the write lock for a space. Blocks until acquired or timeout.
func (w *SpaceWriteLock) Lock(spaceID string) error {
	w.mu.Lock()
	sl, ok := w.locks[spaceID]
	if !ok {
		sl = &spaceLock{unlocked: make(chan struct{}, 1)}
		w.locks[spaceID] = sl
	}
	w.mu.Unlock()

	// Try to acquire with timeout
	acquired := make(chan struct{}, 1)
	go func() {
		sl.mu.Lock()
		sl.held = true
		acquired <- struct{}{}
	}()

	select {
	case <-acquired:
		return nil
	case <-time.After(w.timeout):
		return fmt.Errorf("timeout acquiring write lock for space %s (waited %v)", spaceID, w.timeout)
	}
}

// Unlock releases the write lock for a space.
func (w *SpaceWriteLock) Unlock(spaceID string) {
	w.mu.Lock()
	sl, ok := w.locks[spaceID]
	w.mu.Unlock()
	if !ok || !sl.held {
		return
	}
	sl.held = false
	sl.mu.Unlock()
	select {
	case sl.unlocked <- struct{}{}:
	default:
	}
}

// UnlockAll releases all held write locks (useful for cleanup/defer).
func (w *SpaceWriteLock) UnlockAll() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, sl := range w.locks {
		if sl.held {
			sl.held = false
			sl.mu.Unlock()
		}
	}
}
