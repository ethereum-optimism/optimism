package storage

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type Storage interface {
	// SaveBlobs saves a mapping of block hash : blob to storage
	SaveBlobs(common.Hash, []*eth.BlobAndMetadata) error
	GetLatestSavedBlockHash() (string, error)
}
