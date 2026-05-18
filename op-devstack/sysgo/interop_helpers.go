package sysgo

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

func validateSimpleInteropPresetConfig(t devtest.T, cfg PresetConfig, l2Nets ...*L2Network) {
	require := t.Require()
	if cfg.MaxSequencingWindow != nil {
		for _, l2Net := range l2Nets {
			require.LessOrEqualf(
				l2Net.rollupCfg.SeqWindowSize,
				*cfg.MaxSequencingWindow,
				"sequencing window of chain %s must fit in max sequencing window size",
				l2Net.ChainID(),
			)
		}
	}
	if cfg.RequireInteropNotAtGen {
		for _, l2Net := range l2Nets {
			interopTime := l2Net.genesis.Config.InteropTime
			require.NotNilf(interopTime, "chain %s must have interop", l2Net.ChainID())
			require.NotZerof(*interopTime, "chain %s interop must not be at genesis", l2Net.ChainID())
		}
	}
}

func readJWTSecretFromPath(t devtest.T, jwtPath string) [32]byte {
	content, err := os.ReadFile(jwtPath)
	t.Require().NoError(err, "failed to read jwt path %s", jwtPath)
	raw, err := hexutil.Decode(strings.TrimSpace(string(content)))
	t.Require().NoError(err, "failed to decode jwt secret from %s", jwtPath)
	t.Require().Len(raw, 32, "invalid jwt secret length from %s", jwtPath)
	var secret [32]byte
	copy(secret[:], raw)
	return secret
}
