package cannon

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// cannonUpdater is a [types.OracleUpdater] that exposes a method
// to update onchain cannon oracles with required data.
type cannonUpdater struct {
	logger log.Logger
	txMgr  txmgr.TxManager

	fdgAbi  abi.ABI
	fdgAddr common.Address

	preimageOracleAbi  abi.ABI
	preimageOracleAddr common.Address
}

// NewOracleUpdater returns a new updater.
func NewOracleUpdater(
	logger log.Logger,
	txMgr txmgr.TxManager,
	fdgAddr common.Address,
	preimageOracleAddr common.Address,
) (*cannonUpdater, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	preimageOracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	return &cannonUpdater{
		logger: logger,
		txMgr:  txMgr,

		fdgAbi:  *fdgAbi,
		fdgAddr: fdgAddr,

		preimageOracleAbi:  *preimageOracleAbi,
		preimageOracleAddr: preimageOracleAddr,
	}, nil
}

// UpdateOracle updates the oracle with the given data.
func (u *cannonUpdater) UpdateOracle(ctx context.Context, data types.PreimageOracleData) error {
	panic("oracle updates not implemented")
}
