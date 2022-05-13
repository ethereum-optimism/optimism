package bsscore

import (
	"context"
)

// BatchSubmitter is a service that configures the necessary resources for
// running the TxBatchSubmitter and StateBatchSubmitter sub-services.
type BatchSubmitter struct {
	ctx      context.Context
	services []*Service
	cancel   func()
}

// NewBatchSubmitter initializes the BatchSubmitter, gathering any resources
// that will be needed by the TxBatchSubmitter and StateBatchSubmitter
// sub-services.
func NewBatchSubmitter(
	ctx context.Context,
	cancel func(),
	services []*Service,
) (*BatchSubmitter, error) {

	return &BatchSubmitter{
		ctx:      ctx,
		services: services,
		cancel:   cancel,
	}, nil
}

// Start starts all provided services.
func (b *BatchSubmitter) Start() error {
	for _, service := range b.services {
		if err := service.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops all provided services and blocks until shutdown.
func (b *BatchSubmitter) Stop() {
	b.cancel()
	for _, service := range b.services {
		_ = service.Stop()
	}
}
