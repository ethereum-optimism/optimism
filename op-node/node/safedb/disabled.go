package safedb

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type DisabledDB struct{}

var (
	Disabled      = &DisabledDB{}
	ErrNotEnabled = errors.New("safe head database not enabled")
)

func (d *DisabledDB) SafeHeadUpdated(_ eth.L2BlockRef, _ eth.BlockID) error {
	return nil
}

func (d *DisabledDB) SafeHeadAtL1(_ context.Context, _ uint64) (l1Hash common.Hash, safeHead eth.BlockID, err error) {
	err = ErrNotEnabled
	return
}

func (d *DisabledDB) SafeHeadReset(_ eth.L2BlockRef) error {
	return nil
}

func (d *DisabledDB) Close() error {
	return nil
}
