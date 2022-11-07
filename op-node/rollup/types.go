package rollup

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type Genesis struct {
	// The L1 block that the rollup starts *after* (no derived transactions)
	L1 eth.BlockID `json:"l1"`
	// The L2 block the rollup starts from (no transactions, pre-configured state)
	L2 eth.BlockID `json:"l2"`
	// Timestamp of L2 block
	L2Time uint64 `json:"l2_time"`
	// Initial system configuration values.
	// The L2 genesis block may not include transactions, and thus cannot encode the config values,
	// unlike later L2 blocks.
	SystemConfig eth.SystemConfig `json:"system_config"`
}

type Config struct {
	// Genesis anchor point of the rollup
	Genesis Genesis `json:"genesis"`
	// Seconds per L2 block
	BlockTime uint64 `json:"block_time"`
	// Sequencer batches may not be more than MaxSequencerDrift seconds after
	// the L1 timestamp of the sequencing window end.
	//
	// Note: When L1 has many 1 second consecutive blocks, and L2 grows at fixed 2 seconds,
	// the L2 time may still grow beyond this difference.
	MaxSequencerDrift uint64 `json:"max_sequencer_drift"`
	// Number of epochs (L1 blocks) per sequencing window, including the epoch L1 origin block itself
	SeqWindowSize uint64 `json:"seq_window_size"`
	// Number of L1 blocks between when a channel can be opened and when it must be closed by.
	ChannelTimeout uint64 `json:"channel_timeout"`
	// Required to verify L1 signatures
	L1ChainID *big.Int `json:"l1_chain_id"`
	// Required to identify the L2 network and create p2p signatures unique for this chain.
	L2ChainID *big.Int `json:"l2_chain_id"`

	// Address of the key the sequencer uses to sign blocks on the P2P layer
	P2PSequencerAddress common.Address `json:"p2p_sequencer_address"`

	// Note: below addresses are part of the block-derivation process,
	// and required to be the same network-wide to stay in consensus.

	// L1 address that batches are sent to.
	BatchInboxAddress common.Address `json:"batch_inbox_address"`
	// L1 Deposit Contract Address
	DepositContractAddress common.Address `json:"deposit_contract_address"`
	// L1 System Config Address
	L1SystemConfigAddress common.Address `json:"l1_system_config_address"`
}

// Check verifies that the given configuration makes sense
func (cfg *Config) Check() error {
	if cfg.BlockTime == 0 {
		return fmt.Errorf("block time cannot be 0, got %d", cfg.BlockTime)
	}
	if cfg.ChannelTimeout == 0 {
		return fmt.Errorf("channel timeout must be set, this should cover at least a L1 block time")
	}
	if cfg.SeqWindowSize < 2 {
		return fmt.Errorf("sequencing window size must at least be 2, got %d", cfg.SeqWindowSize)
	}
	if cfg.Genesis.L1.Hash == (common.Hash{}) {
		return errors.New("genesis l1 hash cannot be empty")
	}
	if cfg.Genesis.L2.Hash == (common.Hash{}) {
		return errors.New("genesis l2 hash cannot be empty")
	}
	if cfg.Genesis.L2.Hash == cfg.Genesis.L1.Hash {
		return errors.New("achievement get! rollup inception: L1 and L2 genesis cannot be the same")
	}
	if cfg.Genesis.L2Time == 0 {
		return errors.New("missing L2 genesis time")
	}
	if cfg.Genesis.SystemConfig.BatcherAddr == (common.Address{}) {
		return errors.New("missing genesis system config batcher address")
	}
	if cfg.Genesis.SystemConfig.Overhead == (eth.Bytes32{}) {
		return errors.New("missing genesis system config overhead")
	}
	if cfg.Genesis.SystemConfig.Scalar == (eth.Bytes32{}) {
		return errors.New("missing genesis system config scalar")
	}
	if cfg.Genesis.SystemConfig.GasLimit == 0 {
		return errors.New("missing genesis system config gas limit")
	}
	if cfg.P2PSequencerAddress == (common.Address{}) {
		return errors.New("missing p2p sequencer address")
	}
	if cfg.BatchInboxAddress == (common.Address{}) {
		return errors.New("missing batch inbox address")
	}
	if cfg.DepositContractAddress == (common.Address{}) {
		return errors.New("missing deposit contract address")
	}
	if cfg.L1ChainID == nil {
		return errors.New("l1 chain ID must not be nil")
	}
	if cfg.L2ChainID == nil {
		return errors.New("l2 chain ID must not be nil")
	}
	if cfg.L1ChainID.Cmp(cfg.L2ChainID) == 0 {
		return errors.New("l1 and l2 chain IDs must be different")
	}
	return nil
}

func (c *Config) L1Signer() types.Signer {
	return types.NewLondonSigner(c.L1ChainID)
}

type Epoch uint64
