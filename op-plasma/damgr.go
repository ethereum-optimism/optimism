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
	Log     log.Logger
	Storage DAStorage
}

type Input struct {
	Data eth.Data
}

func NewPlasmaDA(log log.Logger, cfg CLIConfig) *DA {
	return &DA{
		Log:     log,
		Storage: cfg.NewDAClient(),
	}
}

// GetInput returns the input data for the given commitment bytes. blockNumber is required to lookup
// the challenge status in the DataAvailabilityChallenge L1 contract.
func (d *DA) GetInput(ctx context.Context, commitment []byte, blockNumber uint64) (Input, error) {
	data, err := d.Storage.GetInput(ctx, commitment)
	if err != nil {
		return Input{}, err
	}
	return Input{Data: data}, nil
}
