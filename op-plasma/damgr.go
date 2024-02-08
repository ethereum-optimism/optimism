package plasma

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DAStorage interface {
	GetInput(ctx context.Context, key []byte) ([]byte, error)
	SetInput(ctx context.Context, img []byte) ([]byte, error)
}

type DA struct {
	log     log.Logger
	storage DAStorage
}

type Input struct {
	Data eth.Data
}

// NewPlasmaDA creates a new PlasmaDA instance with the given log and CLIConfig.
func NewPlasmaDA(log log.Logger, cfg CLIConfig) *DA {
	return &DA{
		log:     log,
		storage: cfg.NewDAClient(),
	}
}

// NewPlasmaDAWithStorage creates a new PlasmaDA instance with the given log and DAStorage interface.
func NewPlasmaDAWithStorage(log log.Logger, storage DAStorage) *DA {
	return &DA{
		log:     log,
		storage: storage,
	}
}

// GetInput returns the input data for the given commitment bytes. blockNumber is required to lookup
// the challenge status in the DataAvailabilityChallenge L1 contract.
func (d *DA) GetInput(ctx context.Context, commitment []byte, blockNumber uint64) (Input, error) {
	data, err := d.storage.GetInput(ctx, commitment)
	if err != nil {
		return Input{}, err
	}
	return Input{Data: data}, nil
}
