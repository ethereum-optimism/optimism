package rollup

import (
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

// SafeMaxRLPBytesPerChannel is a limit of RLP Bytes per channel that is valid across every OP Stack chain.
// The limit on certain chains at certain times may be higher
// TODO(#10428) Remove this parameter
const SafeMaxRLPBytesPerChannel = maxRLPBytesPerChannelBedrock

// Fjord changes the max sequencer drift to a protocol constant. It was previously configurable via
// the rollup config.
// From Fjord, the max sequencer drift for a given block timestamp should be learned via the
// ChainSpec instead of reading the rollup configuration field directly.
const maxSequencerDriftFjord = 1800

type ForkName string

const (
	Regolith ForkName = "regolith"
	Canyon   ForkName = "canyon"
	Delta    ForkName = "delta"
	Ecotone  ForkName = "ecotone"
	Fjord    ForkName = "fjord"
	Interop  ForkName = "interop"
	None     ForkName = "none"
)

var nextFork = map[ForkName]ForkName{
	Regolith: Canyon,
	Canyon:   Delta,
	Delta:    Ecotone,
	Ecotone:  Fjord,
	Fjord:    Interop,
	Interop:  None,
}

type ChainSpec struct {
	config   *Config
	nextFork ForkName
}

func NewChainSpec(config *Config) *ChainSpec {
	return &ChainSpec{config: config}
}

// IsCanyon returns true if t >= canyon_time
func (s *ChainSpec) IsCanyon(t uint64) bool {
	return s.config.IsCanyon(t)
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
func (s *ChainSpec) ChannelTimeout() uint64 {
	return s.config.ChannelTimeout
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
	if s.nextFork == None {
		return
	}

	if s.nextFork == "" {
		// Initialize c.nextFork if it is not set yet
		if !s.config.IsRegolith(block.Time) {
			s.nextFork = Regolith
		} else if !s.config.IsCanyon(block.Time) {
			s.nextFork = Canyon
		} else if !s.config.IsDelta(block.Time) {
			s.nextFork = Delta
		} else if !s.config.IsEcotone(block.Time) {
			s.nextFork = Ecotone
		} else if !s.config.IsFjord(block.Time) {
			s.nextFork = Fjord
		} else if !s.config.IsInterop(block.Time) {
			s.nextFork = Interop
		} else {
			s.nextFork = None
			return
		}
	}

	foundActivationBlock := false

	switch s.nextFork {
	case Regolith:
		foundActivationBlock = s.config.IsRegolithActivationBlock(block.Time)
	case Canyon:
		foundActivationBlock = s.config.IsCanyonActivationBlock(block.Time)
	case Delta:
		foundActivationBlock = s.config.IsDeltaActivationBlock(block.Time)
	case Ecotone:
		foundActivationBlock = s.config.IsEcotoneActivationBlock(block.Time)
	case Fjord:
		foundActivationBlock = s.config.IsFjordActivationBlock(block.Time)
	case Interop:
		foundActivationBlock = s.config.IsInteropActivationBlock(block.Time)
	}

	if foundActivationBlock {
		log.Info("Detected hardfork activation block", "forkName", s.nextFork, "timestamp", block.Time, "blockNum", block.Number, "hash", block.Hash)
		s.nextFork = nextFork[s.nextFork]
	}
}
