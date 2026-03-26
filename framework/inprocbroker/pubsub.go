package inprocbroker

import (
	"context"
	"sync"

	"github.com/leomorpho/goship/framework/core"
)

type inProcPubSub struct {
	mu     sync.RWMutex
	topics map[string]map[*inProcSubscription]struct{}
}

type inProcSubscription struct {
	topic   string
	handler core.MessageHandler
	ps      *inProcPubSub
	once    sync.Once
}

// NewInProc creates a new in-process pubsub implementation.
func NewInProc() core.PubSub {
	return &inProcPubSub{
		topics: make(map[string]map[*inProcSubscription]struct{}),
	}
}

func (p *inProcPubSub) Publish(ctx context.Context, topic string, payload []byte) error {
	p.mu.RLock()
	subs := make([]*inProcSubscription, 0, len(p.topics[topic]))
	for s := range p.topics[topic] {
		subs = append(subs, s)
	}
	p.mu.RUnlock()

	for _, s := range subs {
		if err := s.handler(ctx, topic, payload); err != nil {
			// In a real pubsub, we might log this or handle it differently.
			// For in-process, we just continue to other subscribers.
		}
	}
	return nil
}

func (p *inProcPubSub) Subscribe(ctx context.Context, topic string, handler core.MessageHandler) (core.Subscription, error) {
	s := &inProcSubscription{
		topic:   topic,
		handler: handler,
		ps:      p,
	}

	p.mu.Lock()
	if p.topics[topic] == nil {
		p.topics[topic] = make(map[*inProcSubscription]struct{})
	}
	p.topics[topic][s] = struct{}{}
	p.mu.Unlock()

	return s, nil
}

func (p *inProcPubSub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topics = make(map[string]map[*inProcSubscription]struct{})
	return nil
}

func (s *inProcSubscription) Close() error {
	s.once.Do(func() {
		s.ps.mu.Lock()
		defer s.ps.mu.Unlock()
		if subs, ok := s.ps.topics[s.topic]; ok {
			delete(subs, s)
			if len(subs) == 0 {
				delete(s.ps.topics, s.topic)
			}
		}
	})
	return nil
}
