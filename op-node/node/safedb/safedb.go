package safedb

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type SafeDB struct {
	log log.Logger
}

func NewSafeDB(logger log.Logger, path string) *SafeDB {
	return &SafeDB{
		log: logger,
	}
}

func (d SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) {
	// TODO(client-pod#593): Write to a database
	d.log.Info("Update safe head", "l2", safeHead.ID(), "l1", l1Head)
}
