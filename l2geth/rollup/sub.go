package rollup

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/ethereum-optimism/optimism/l2geth/log"
)

type QueueSubscriberMessage interface {
	Data() []byte
	Ack()
	Nack()
}

type QueueSubscriber interface {
	ReceiveMessage(ctx context.Context, cb func(ctx context.Context, msg QueueSubscriberMessage)) error
	Close() error
}

type QueueSubscriberConfig struct {
	Enable                 bool
	ProjectID              string
	SubscriptionID         string
	MaxOutstandingMessages int
	MaxOutstandingBytes    int
}

type queueSubscriber struct {
	client *pubsub.Client
	sub    *pubsub.Subscription
}

func NewQueueSubscriber(ctx context.Context, config QueueSubscriberConfig) (QueueSubscriber, error) {
	if !config.Enable {
		return &noopQueueSubscriber{}, nil
	}

	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, err
	}

	sub := client.Subscription(config.SubscriptionID)

	maxOutstandingMsgs := config.MaxOutstandingMessages
	if maxOutstandingMsgs == 0 {
		maxOutstandingMsgs = 10000
	}
	maxOutstandingBytes := config.MaxOutstandingBytes
	if maxOutstandingBytes == 0 {
		maxOutstandingBytes = 1e9
	}
	sub.ReceiveSettings = pubsub.ReceiveSettings{
		MaxOutstandingMessages: maxOutstandingMsgs,
		MaxOutstandingBytes:    maxOutstandingBytes,
	}

	log.Info("Created Queue Subscriber", "projectID", config.ProjectID, "subscriptionID", config.SubscriptionID)
	return &queueSubscriber{client, sub}, nil
}

func (q *queueSubscriber) ReceiveMessage(ctx context.Context, cb func(ctx context.Context, msg QueueSubscriberMessage)) error {
	return q.sub.Receive(ctx, func(ctx context.Context, pmsg *pubsub.Message) {
		cb(ctx, &queueSubscriberMessage{pmsg})
	})
}

func (q *queueSubscriber) Close() error {
	return q.client.Close()
}

type queueSubscriberMessage struct {
	inner *pubsub.Message
}

func (q *queueSubscriberMessage) Data() []byte {
	return q.inner.Data
}

func (q *queueSubscriberMessage) Ack() {
	q.inner.Ack()
}

func (q *queueSubscriberMessage) Nack() {
	q.inner.Nack()
}

type noopQueueSubscriber struct{}

func (q *noopQueueSubscriber) ReceiveMessage(ctx context.Context, cb func(ctx context.Context, msg QueueSubscriberMessage)) error {
	return nil
}
func (q *noopQueueSubscriber) Close() error { return nil }
