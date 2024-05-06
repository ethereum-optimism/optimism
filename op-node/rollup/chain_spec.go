package rollup

// maxChannelBankSize is the amount of memory space, in number of bytes,
// till the bank is pruned by removing channels, starting with the oldest channel.
// It's value is changed with the Fjord network upgrade.
const (
	maxChannelBankSizeBedrock = 100_000_000
	maxChannelBankSizeFjord   = 1_000_000_000
)

type ChainSpec struct {
	config *Config
}

func NewChainSpec(config *Config) *ChainSpec {
	return &ChainSpec{config}
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
