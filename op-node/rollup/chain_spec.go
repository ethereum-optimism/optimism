package rollup

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// maxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels, starting with the oldest channel.
// It's value is changed with the Fjord network upgrade.
const (
	maxChannelBankSizeBedrock = 100_000_000
	maxChannelBankSizeFjord   = 1_000_000_000
)

// MaxRLPBytesPerChannel is the maximum amount of bytes that will be read from
// a channel. This limit is set when decoding the RLP.
const (
	maxRLPBytesPerChannelBedrock = 10_000_000
	maxRLPBytesPerChannelFjord   = 100_000_000
)

// Fjord changes the max sequencer drift to a protocol constant. It was previously configurable via
// the rollup config.
// From Fjord, the max sequencer drift for a given block timestamp should be learned via the
// ChainSpec instead of reading the rollup configuration field directly.
const maxSequencerDriftFjord = 1800

// Legacy type alias kept temporarily for rollup internals; external code should use forks.Name directly.
type ForkName = forks.Name

type ChainSpec struct {
	config      *Config
	currentFork ForkName
}

func NewChainSpec(config *Config) *ChainSpec {
	return &ChainSpec{config: config}
}

// L2ChainID returns the chain ID of the L2 chain.
func (s *ChainSpec) L2ChainID() *big.Int {
	return s.config.L2ChainID
}

// L2GenesisTime returns the genesis time of the L2 chain.
func (s *ChainSpec) L2GenesisTime() uint64 {
	return s.config.Genesis.L2Time
}

// IsCanyon returns true if t >= canyon_time
func (s *ChainSpec) IsCanyon(t uint64) bool {
	return s.config.IsCanyon(t)
}

// IsHolocene returns true if t >= holocene_time
func (s *ChainSpec) IsHolocene(t uint64) bool {
	return s.config.IsHolocene(t)
}

// IsIsthmus returns true if t >= isthmus_time
func (s *ChainSpec) IsIsthmus(t uint64) bool {
	return s.config.IsIsthmus(t)
}

// IsJovian returns true if t >= jovian_time
func (s *ChainSpec) IsJovian(t uint64) bool {
	return s.config.IsJovian(t)
}

// MaxChannelBankSize returns the maximum number of bytes the can allocated inside the channel bank
// before pruning occurs at the given timestamp.
func (s *ChainSpec) MaxChannelBankSize(t uint64) uint64 {
	if s.config.IsFjord(t) {
		return maxChannelBankSizeFjord
	}
	return maxChannelBankSizeBedrock
}

// ChannelTimeout returns the channel timeout constant.
func (s *ChainSpec) ChannelTimeout(t uint64) uint64 {
	if s.config.IsGranite(t) {
		return params.ChannelTimeoutGranite
	}
	return s.config.ChannelTimeoutBedrock
}

// MaxRLPBytesPerChannel returns the maximum amount of bytes that will be read from
// a channel at a given timestamp.
func (s *ChainSpec) MaxRLPBytesPerChannel(t uint64) uint64 {
	if s.config.IsFjord(t) {
		return maxRLPBytesPerChannelFjord
	}
	return maxRLPBytesPerChannelBedrock
}

// IsFeatMaxSequencerDriftConstant specifies in which fork the max sequencer drift change to a
// constant will be performed.
func (s *ChainSpec) IsFeatMaxSequencerDriftConstant(t uint64) bool {
	return s.config.IsFjord(t)
}

// MaxSequencerDrift returns the maximum sequencer drift for the given block timestamp. Until Fjord,
// this was a rollup configuration parameter. Since Fjord, it is a constant, so its effective value
// should always be queried via the ChainSpec.
func (s *ChainSpec) MaxSequencerDrift(t uint64) uint64 {
	if s.IsFeatMaxSequencerDriftConstant(t) {
		return maxSequencerDriftFjord
	}
	return s.config.MaxSequencerDrift
}

func (s *ChainSpec) CheckForkActivation(log log.Logger, block eth.L2BlockRef) {
	if s.currentFork == "" {
		// Initialize currentFork if it is not set yet
		s.currentFork = forks.Bedrock
		if s.config.IsRegolith(block.Time) {
			s.currentFork = forks.Regolith
		}
		if s.config.IsCanyon(block.Time) {
			s.currentFork = forks.Canyon
		}
		if s.config.IsDelta(block.Time) {
			s.currentFork = forks.Delta
		}
		if s.config.IsEcotone(block.Time) {
			s.currentFork = forks.Ecotone
		}
		if s.config.IsFjord(block.Time) {
			s.currentFork = forks.Fjord
		}
		if s.config.IsGranite(block.Time) {
			s.currentFork = forks.Granite
		}
		if s.config.IsHolocene(block.Time) {
			s.currentFork = forks.Holocene
		}
		if s.config.IsIsthmus(block.Time) {
			s.currentFork = forks.Isthmus
		}
		if s.config.IsJovian(block.Time) {
			s.currentFork = forks.Jovian
		}
		if s.config.IsInterop(block.Time) {
			s.currentFork = forks.Interop
		}
		log.Info("Current hardfork version detected", "forkName", s.currentFork)
		return
	}

	foundActivationBlock := false

	switch forks.Next(s.currentFork) {
	case forks.Regolith:
		foundActivationBlock = s.config.IsRegolithActivationBlock(block.Time)
	case forks.Canyon:
		foundActivationBlock = s.config.IsCanyonActivationBlock(block.Time)
	case forks.Delta:
		foundActivationBlock = s.config.IsDeltaActivationBlock(block.Time)
	case forks.Ecotone:
		foundActivationBlock = s.config.IsEcotoneActivationBlock(block.Time)
	case forks.Fjord:
		foundActivationBlock = s.config.IsFjordActivationBlock(block.Time)
	case forks.Granite:
		foundActivationBlock = s.config.IsGraniteActivationBlock(block.Time)
	case forks.Holocene:
		foundActivationBlock = s.config.IsHoloceneActivationBlock(block.Time)
	case forks.Isthmus:
		foundActivationBlock = s.config.IsIsthmusActivationBlock(block.Time)
	case forks.Jovian:
		foundActivationBlock = s.config.IsJovianActivationBlock(block.Time)
	case forks.Interop:
		foundActivationBlock = s.config.IsInteropActivationBlock(block.Time)
	}

	if foundActivationBlock {
		s.currentFork = forks.Next(s.currentFork)
		log.Info("Detected hardfork activation block", "forkName", s.currentFork, "timestamp", block.Time, "blockNum", block.Number, "hash", block.Hash)
	}
}
