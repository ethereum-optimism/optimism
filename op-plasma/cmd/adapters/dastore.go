package adapters

import "context"

type DAService interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Put(ctx context.Context, value []byte) ([]byte, error) // returns the key back
}

type DAServiceAdapter struct {
	DAService DAService
}

func (adapter DAServiceAdapter) Get(ctx context.Context, key []byte) ([]byte, error) {
	return adapter.DAService.Get(ctx, key)
}

func (adapter DAServiceAdapter) Put(ctx context.Context, key []byte, value []byte) ([]byte, error) {
	return adapter.DAService.Put(ctx, value)
}
