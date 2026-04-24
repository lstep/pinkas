package sse

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// EventType defines the type of SSE event.
type EventType string

const (
	EventPageCreated EventType = "page_created"
	EventPageUpdated EventType = "page_updated"
	EventPageMoved   EventType = "page_moved"
	EventPageDeleted EventType = "page_deleted"

	EventDirectoryCreated EventType = "directory_created"
	EventDirectoryUpdated EventType = "directory_updated"
	EventDirectoryMoved   EventType = "directory_moved"
	EventDirectoryDeleted EventType = "directory_deleted"
)

// Event is a single SSE event.
type Event struct {
	ID        int64       `json:"event_id"`
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// subscriber represents an active SSE connection.
type subscriber struct {
	ch     chan Event
	lastID int64
}

// Hub manages SSE subscribers and a ring buffer for replay.
type Hub struct {
	logger      *slog.Logger
	subscribers sync.Map // map[*subscriber]struct{}
	buffer      *ringBuffer
	nextID      int64
	mu          sync.Mutex
}

// NewHub creates a new SSE hub with the given ring buffer capacity.
func NewHub(logger *slog.Logger, bufferCapacity int) *Hub {
	if bufferCapacity <= 0 {
		bufferCapacity = 256
	}
	return &Hub{
		logger: logger,
		buffer: newRingBuffer(bufferCapacity),
	}
}

// Subscribe adds a new subscriber and returns a channel of events.
// The subscriber receives events with ID > lastID.
func (h *Hub) Subscribe(lastID int64) (<-chan Event, func()) {
	sub := &subscriber{
		ch:     make(chan Event, 16),
		lastID: lastID,
	}
	h.subscribers.Store(sub, struct{}{})

	// Replay missed events from ring buffer
	if events := h.buffer.Since(lastID); len(events) > 0 {
		go func() {
			for _, ev := range events {
				select {
				case sub.ch <- ev:
				case <-time.After(5 * time.Second):
					h.logger.Warn("sse replay timeout", "event_id", ev.ID)
					return
				}
			}
		}()
	}

	unsub := func() {
		h.subscribers.Delete(sub)
		close(sub.ch)
	}
	return sub.ch, unsub
}

// Broadcast sends an event to all subscribers and stores it in the ring buffer.
func (h *Hub) Broadcast(ev Event) {
	h.mu.Lock()
	h.nextID++
	ev.ID = h.nextID
	ev.Timestamp = time.Now()
	h.buffer.Push(ev)
	h.mu.Unlock()

	h.subscribers.Range(func(key, value interface{}) bool {
		sub := key.(*subscriber)
		select {
		case sub.ch <- ev:
		default:
			// Subscriber is slow, drop the event
			h.logger.Warn("sse subscriber slow, dropping event", "event_id", ev.ID)
		}
		return true
	})
}

// ringBuffer is a circular buffer of events for replay.
type ringBuffer struct {
	events []Event
	head   int // index of oldest event
	size   int // number of valid events
	cap    int
}

func newRingBuffer(capacity int) *ringBuffer {
	return &ringBuffer{
		events: make([]Event, capacity),
		cap:    capacity,
	}
}

func (rb *ringBuffer) Push(ev Event) {
	if rb.cap == 0 {
		return
	}
	idx := (rb.head + rb.size) % rb.cap
	rb.events[idx] = ev
	if rb.size < rb.cap {
		rb.size++
	} else {
		rb.head = (rb.head + 1) % rb.cap
	}
}

// Since returns all events with ID > lastID.
func (rb *ringBuffer) Since(lastID int64) []Event {
	if rb.size == 0 {
		return nil
	}
	var result []Event
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.cap
		if rb.events[idx].ID > lastID {
			result = append(result, rb.events[idx])
		}
	}
	return result
}

// FormatSSE formats an event as an SSE data stream.
func FormatSSE(ev Event) string {
	data, _ := json.Marshal(ev.Data)
	return fmt.Sprintf("id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, string(data))
}
