package sse

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/a-h/templ"
	"github.com/leomorpho/goship/framework/core"
)

type Hub struct {
	mu     sync.RWMutex
	ps     core.PubSub
	topics map[string]*topicState
}

type topicState struct {
	subs      map[chan string]struct{}
	psSub     core.Subscription
	isClosing bool
}

func NewHub(ps core.PubSub) *Hub {
	return &Hub{
		ps:     ps,
		topics: make(map[string]*topicState),
	}
}

func (h *Hub) Subscribe(topic string) (chan string, func()) {
	ch := make(chan string, 8)
	if h == nil {
		return ch, func() {}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ts, ok := h.topics[topic]
	if !ok {
		ts = &topicState{
			subs: make(map[chan string]struct{}),
		}
		h.topics[topic] = ts

		if h.ps != nil {
			sub, err := h.ps.Subscribe(context.Background(), topic, func(ctx context.Context, _ string, payload []byte) error {
				h.fanoutLocal(topic, string(payload))
				return nil
			})
			if err != nil {
				// Log error? For now, we continue with local only if pubsub fails
				fmt.Printf("failed to subscribe to pubsub for topic %s: %v\n", topic, err)
			} else {
				ts.psSub = sub
			}
		}
	}
	ts.subs[ch] = struct{}{}

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		ts, ok := h.topics[topic]
		if !ok {
			return
		}

		delete(ts.subs, ch)
		if len(ts.subs) == 0 && ts.psSub != nil {
			_ = ts.psSub.Close()
			delete(h.topics, topic)
		}
	}
}

func (h *Hub) Publish(topic string, data string) {
	if h == nil {
		return
	}

	if h.ps != nil {
		// Publish to distributed pubsub
		if err := h.ps.Publish(context.Background(), topic, []byte(data)); err != nil {
			// Fallback to local fanout if pubsub fails?
			// Some might prefer to only publish to pubsub and let the message come back.
			// But for reliability/latency, we might want both or just pubsub.
			// If we do both, we need to handle duplicates.
			// For now, if we have ps, we let it handle it.
			// If ps is nil, we fanout locally.
		}
		return
	}

	h.fanoutLocal(topic, data)
}

func (h *Hub) fanoutLocal(topic, data string) {
	h.mu.RLock()
	ts, ok := h.topics[topic]
	if !ok {
		h.mu.RUnlock()
		return
	}

	subscribers := make([]chan string, 0, len(ts.subs))
	for ch := range ts.subs {
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

func (h *Hub) PublishHTMLComponent(topic string, component templ.Component) error {
	return h.PublishHTML(topic, component)
}
