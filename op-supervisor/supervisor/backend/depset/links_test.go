package depset

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type mockLinkCfg struct {
	activationTimes map[eth.ChainID]uint64
	window          uint64
}

func (m *mockLinkCfg) Chains() (out []eth.ChainID) {
	for id := range m.activationTimes {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Cmp(out[j]) < 0
	})
	return
}

func (m *mockLinkCfg) HasChain(chainID eth.ChainID) bool {
	_, ok := m.activationTimes[chainID]
	return ok
}

func (m *mockLinkCfg) IsInterop(chainID eth.ChainID, ts uint64) bool {
	v, ok := m.activationTimes[chainID]
	if !ok {
		return false
	}
	return ts >= v
}

func (m *mockLinkCfg) IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool {
	v, ok := m.activationTimes[chainID]
	if !ok {
		return false
	}
	return ts == v
}

func (m *mockLinkCfg) MessageExpiryWindow() uint64 {
	return m.window
}

var _ LinkerConfig = (*mockLinkCfg)(nil)

func TestLinkChecker(t *testing.T) {
	chainA := eth.ChainIDFromUInt64(900)
	chainB := eth.ChainIDFromUInt64(901)
	chainUnknown := eth.ChainIDFromUInt64(700)

	cfg := &mockLinkCfg{
		activationTimes: map[eth.ChainID]uint64{
			chainA: 1000,
			chainB: 900,
		},
		window: 400,
	}
	linker := LinkerFromConfig(cfg)
	req := require.New(t)

	req.False(linker.CanExecute(chainA, 999, chainB, 950), "cannot exec pre-interop")
	req.False(linker.CanExecute(chainA, 1050, chainB, 899), "cannot init pre-interop")

	req.False(linker.CanExecute(chainUnknown, 2050, chainB, 2000), "cannot execute on unknown chain")
	req.False(linker.CanExecute(chainA, 2050, chainUnknown, 2000), "cannot initiate on unknown chain")
	req.False(linker.CanExecute(chainUnknown, 2050, chainUnknown, 2000), "cannot both be on unknown chain")

	req.False(linker.CanExecute(chainA, 1050, chainB, 1051), "cannot init after exec")

	req.True(linker.CanExecute(chainA, 2000, chainB, 2000), "simple same timestamp")
	req.True(linker.CanExecute(chainA, 2050, chainB, 2000), "simple diff timestamp")
	req.True(linker.CanExecute(chainA, 2399, chainB, 2000), "near expiry")
	req.True(linker.CanExecute(chainA, 2400, chainB, 2000), "at expiry")
	req.False(linker.CanExecute(chainA, 2401, chainB, 2000), "expired")
	req.False(linker.CanExecute(chainA, (^uint64(0))-200, chainB, (^uint64(0))-300), "claimed init msg causes overflow")

	req.True(linker.CanExecute(chainA, 1001, chainA, 1001), "with self")
	req.False(linker.CanExecute(chainA, 1001, chainA, 1000), "no init at activation")
	req.False(linker.CanExecute(chainA, 1000, chainA, 1001), "no exec at activation")
	req.True(linker.CanExecute(chainA, 1001, chainB, 950), "other inits pre-activation of self")
}
