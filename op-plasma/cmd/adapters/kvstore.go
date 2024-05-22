package adapters

import "context"

type KVStore interface {
	// Get retrieves the given key if it's present in the key-value data store.
	Get(ctx context.Context, key []byte) ([]byte, error)
	// Put inserts the given value into the key-value data store.
	Put(ctx context.Context, key []byte, value []byte) error
}

type KVStoreAdapter struct {
	KVStore KVStore
}

func (adapter KVStoreAdapter) Get(ctx context.Context, key []byte) ([]byte, error) {
	return adapter.KVStore.Get(ctx, key)
}

func (adapter KVStoreAdapter) Put(ctx context.Context, key []byte, value []byte) ([]byte, error) {
	err := adapter.KVStore.Put(ctx, key, value)
	if err != nil {
		return nil, err
	}
	return key, nil
}
