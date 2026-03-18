package bootstrap

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/framework/core"
	"github.com/stretchr/testify/require"
)

type testSubscription struct {
	closed bool
}

func (s *testSubscription) Close() error {
	s.closed = true
	return nil
}

type testPubSub struct {
	publishTopic   string
	publishPayload []byte
	handler        core.MessageHandler
	sub            *testSubscription
}

func (p *testPubSub) Publish(_ context.Context, topic string, payload []byte) error {
	p.publishTopic = topic
	p.publishPayload = append([]byte(nil), payload...)
	return nil
}

func (p *testPubSub) Subscribe(_ context.Context, _ string, handler core.MessageHandler) (core.Subscription, error) {
	p.handler = handler
	p.sub = &testSubscription{}
	return p.sub, nil
}

func (p *testPubSub) Close() error { return nil }

func TestAdaptNotificationsPubSub(t *testing.T) {
	t.Parallel()

	require.Nil(t, AdaptNotificationsPubSub(nil))

	pub := &testPubSub{}
	adapter := AdaptNotificationsPubSub(pub)
	require.NotNil(t, adapter)

	require.NoError(t, adapter.Publish(context.Background(), "topic.a", []byte("hello")))
	require.Equal(t, "topic.a", pub.publishTopic)
	require.Equal(t, []byte("hello"), pub.publishPayload)

	called := false
	sub, err := adapter.Subscribe(context.Background(), "topic.b", func(_ context.Context, topic string, payload []byte) error {
		called = true
		require.Equal(t, "topic.b", topic)
		require.Equal(t, []byte("data"), payload)
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, sub)

	require.NoError(t, pub.handler(context.Background(), "topic.b", []byte("data")))
	require.True(t, called)
	require.NoError(t, sub.Close())
	require.True(t, pub.sub.closed)
}
