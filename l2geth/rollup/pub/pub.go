package pub

import "context"

type Publisher interface {
	// Publish schedules an ordereed message to be sent
	Publish(ctx context.Context, msg []byte) error
}

type NoopPublisher struct{}

func (p *NoopPublisher) Publish(ctx context.Context, msg []byte) error {
	return nil
}
