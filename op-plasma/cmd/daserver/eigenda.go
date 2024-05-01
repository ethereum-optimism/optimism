package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/eigenda"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum/go-ethereum/rlp"
)

type EigenDAStore struct {
	client *eigenda.EigenDAClient
}

var _ plasma.PlasmaStore = EigenDAStore{}

func NewEigenDAStore(ctx context.Context, client *eigenda.EigenDAClient) (*EigenDAStore, error) {
	return &EigenDAStore{
		client: client,
	}, nil
}

// Get retrieves the given key if it's present in the key-value data store.
func (e EigenDAStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	var cert eigenda.Cert
	err := rlp.DecodeBytes(key, cert)
	if err != nil {
		return nil, fmt.Errorf("failed to encode DA cert to RLP format: %w", err)
	}
	blob, err := e.client.RetrieveBlob(ctx, cert.BatchHeaderHash, cert.BlobIndex)
	if err != nil {
		return nil, fmt.Errorf("EigenDA client failed to retrieve blob: %w", err)
	}
	return blob, nil
}

// PutWithCommitment attempts to insert the given key and value into the key-value data store
// and fails if the commitment does not match the
func (e EigenDAStore) PutWithComm(ctx context.Context, key []byte, value []byte) error {
	return fmt.Errorf("EigenDA plasma store does not support PutWithComm()")
}

// PutWithoutComm inserts the given value into the key-value data store and returns the corresponding commitment
func (e EigenDAStore) PutWithoutComm(ctx context.Context, value []byte) (comm []byte, err error) {
	cert, err := e.client.DisperseBlob(ctx, value)
	if err != nil {
		return nil, err
	}
	bytes, err := rlp.EncodeToBytes(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to encode DA cert to RLP format: %w", err)
	}
	return bytes, nil
}
