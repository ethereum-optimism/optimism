package pure

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// buildAttributes constructs a DerivedBlock (PayloadAttributes + metadata) from
// a validated singular batch, its L1 origin, the current derivation cursor,
// and the active system config.
//
// Transaction ordering follows the OP Stack derivation spec:
//  1. L1 info deposit transaction (always first)
//  2. User deposit transactions (only at epoch boundaries)
//  3. Batch transactions from the sequencer
//
// Network upgrade transactions (NUTs) are not included because all pre-Karst
// forks are already active (PureDerive requires Karst), and Karst itself has
// no NUTs. Future forks with NUTs must be added here.
func buildAttributes(
	batch *derive.SingularBatch,
	l1Block *L1Input,
	cursor l2Cursor,
	sysConfig eth.SystemConfig,
	cfg *rollup.Config,
	l1ChainConfig *params.ChainConfig,
) (*DerivedBlock, error) {
	epochChanged := uint64(batch.EpochNum) != cursor.L1Origin.Number

	var seqNumber uint64
	if epochChanged {
		seqNumber = 0
	} else {
		seqNumber = cursor.SequenceNumber + 1
	}

	l2Timestamp := batch.Timestamp

	l1InfoTx, err := derive.L1InfoDeposit(cfg, l1ChainConfig, sysConfig, seqNumber, eth.HeaderBlockInfo(l1Block.Header), l2Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 info deposit tx: %w", err)
	}

	encodedL1Info, err := types.NewTx(l1InfoTx).MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode L1 info deposit tx: %w", err)
	}

	txCount := 1 + len(batch.Transactions)
	if epochChanged {
		txCount += len(l1Block.Deposits)
	}
	txs := make([]hexutil.Bytes, 0, txCount)
	txs = append(txs, encodedL1Info)

	if epochChanged {
		for _, dep := range l1Block.Deposits {
			encoded, err := types.NewTx(dep).MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("failed to encode user deposit tx: %w", err)
			}
			txs = append(txs, encoded)
		}
	}

	txs = append(txs, batch.Transactions...)

	gasLimit := sysConfig.GasLimit

	var withdrawals *types.Withdrawals
	if cfg.IsCanyon(l2Timestamp) {
		withdrawals = &types.Withdrawals{}
	}

	var parentBeaconRoot *common.Hash
	if cfg.IsEcotone(l2Timestamp) {
		parentBeaconRoot = new(common.Hash)
	}

	attrs := &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(l2Timestamp),
		PrevRandao:            eth.Bytes32(l1Block.Header.MixDigest),
		SuggestedFeeRecipient: predeploys.SequencerFeeVaultAddr,
		Transactions:          txs,
		NoTxPool:              true,
		GasLimit:              (*eth.Uint64Quantity)(&gasLimit),
		Withdrawals:           withdrawals,
		ParentBeaconBlockRoot: parentBeaconRoot,
	}

	if cfg.IsHolocene(l2Timestamp) {
		attrs.EIP1559Params = new(eth.Bytes8)
		*attrs.EIP1559Params = sysConfig.EIP1559Params
	}

	if cfg.IsJovian(l2Timestamp) {
		attrs.MinBaseFee = &sysConfig.MinBaseFee
	}

	return &DerivedBlock{
		Attributes:         attrs,
		ExpectedParentHash: batch.ParentHash,
		DerivedFrom:        l1Block.BlockRef(),
	}, nil
}
