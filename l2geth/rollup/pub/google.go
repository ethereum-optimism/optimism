package pub

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/ethereum-optimism/optimism/l2geth/log"
)

const messageOrderingKey = "o"

type Config struct {
	Enable    bool
	ProjectID string
	TopicID   string
	Timeout   time.Duration
}

type GooglePublisher struct {
	client          *pubsub.Client
	topic           *pubsub.Topic
	publishSettings pubsub.PublishSettings
	timeout         time.Duration
	mutex           sync.Mutex
}

func NewGooglePublisher(ctx context.Context, config Config) (Publisher, error) {
	if !config.Enable {
		return &NoopPublisher{}, nil
	}

	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, err
	}
	topic := client.Topic(config.TopicID)
	topic.EnableMessageOrdering = true

	// Publish messages immediately
	publishSettings := pubsub.PublishSettings{
		DelayThreshold: 0,
		CountThreshold: 0,
	}
	timeout := config.Timeout
	if timeout == 0 {
		log.Info("Sanitizing publisher timeout to 2 seconds")
		timeout = time.Second * 2
	}

	log.Info("Initialized transaction log to PubSub", "topic", config.TopicID)
	return &GooglePublisher{
		client:          client,
		topic:           topic,
		publishSettings: publishSettings,
		timeout:         timeout,
	}, nil
}

func (p *GooglePublisher) Publish(ctx context.Context, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	pmsg := pubsub.Message{
		Data:        msg,
		OrderingKey: messageOrderingKey,
	}

	p.mutex.Lock()
	// If there was an error previously, clear it out to allow publishing to proceed again
	p.topic.ResumePublish(messageOrderingKey)
	result := p.topic.Publish(ctx, &pmsg)
	_, err := result.Get(ctx)
	p.mutex.Unlock()

	return err
}
