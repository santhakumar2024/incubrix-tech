package main

import "sync"

type eventHub struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[chan struct{}]struct{})}
}

func (h *eventHub) subscribe() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subscribers, ch)
		close(ch)
		h.mu.Unlock()
	}
}

func (h *eventHub) publish() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}
