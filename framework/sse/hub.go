package sse

import (
	"bytes"
	"context"
	"sync"

	"github.com/a-h/templ"
)

type Hub struct {
	mu     sync.RWMutex
	topics map[string]map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{
		topics: map[string]map[chan string]struct{}{},
	}
}

func (h *Hub) Subscribe(topic string) (chan string, func()) {
	ch := make(chan string, 8)
	if h == nil {
		return ch, func() {}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = map[chan string]struct{}{}
	}
	h.topics[topic][ch] = struct{}{}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subscribers := h.topics[topic]
		delete(subscribers, ch)
		if len(subscribers) == 0 {
			delete(h.topics, topic)
		}
	}
}

func (h *Hub) Publish(topic string, data string) {
	if h == nil {
		return
	}

	h.mu.RLock()
	subscribers := make([]chan string, 0, len(h.topics[topic]))
	for ch := range h.topics[topic] {
		subscribers = append(subscribers, ch)
	}
	h.mu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- data:
		default:
		}
	}
}

func (h *Hub) PublishHTML(topic string, component templ.Component) error {
	if component == nil {
		h.Publish(topic, "")
		return nil
	}

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		return err
	}
	h.Publish(topic, buf.String())
	return nil
}
