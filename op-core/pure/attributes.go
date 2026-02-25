package pure

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

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
//  3. Network upgrade transactions (at fork activation blocks)
//  4. Batch transactions from the sequencer
func buildAttributes(
	batch *derive.SingularBatch,
	l1Block *L1Input,
	cursor l2Cursor,
	sysConfig eth.SystemConfig,
	cfg *rollup.Config,
) (*DerivedBlock, error) {
	epochChanged := uint64(batch.EpochNum) != cursor.L1Origin.Number

	var seqNumber uint64
	if epochChanged {
		seqNumber = 0
	} else {
		seqNumber = cursor.SequenceNumber + 1
	}

	l2Timestamp := batch.Timestamp

	l1InfoTx, err := derive.L1InfoDeposit(cfg, nil, sysConfig, seqNumber, l1Block.blockInfo(), l2Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 info deposit tx: %w", err)
	}

	encodedL1Info, err := types.NewTx(l1InfoTx).MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode L1 info deposit tx: %w", err)
	}

	// Network upgrade transactions (NUTs). Only forks from Jovian onward are
	// included; earlier forks (Ecotone, Fjord, Isthmus) cannot be activation
	// blocks since PureDerive requires Karst to already be active.
	var upgradeTxs []hexutil.Bytes

	if cfg.IsJovianActivationBlock(l2Timestamp) {
		jovianTxs, err := derive.JovianNetworkUpgradeTransactions()
		if err != nil {
			return nil, fmt.Errorf("failed to build Jovian network upgrade txs: %w", err)
		}
		upgradeTxs = append(upgradeTxs, jovianTxs...)
	}

	// TODO: Add Karst NUTs here once KarstNetworkUpgradeTransactions() exists.
	// Karst currently has no network upgrade transactions.

	txCount := 1 + len(upgradeTxs) + len(batch.Transactions)
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

	txs = append(txs, upgradeTxs...)
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
