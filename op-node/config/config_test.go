package config

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// validConfig returns a node config that passes Check, so that a test can single out the effect of
// the one field it overrides.
func validConfig() *Config {
	return &Config{
		L1: &L1EndpointConfig{
			L1NodeAddr:     "http://localhost:8545",
			BatchSize:      10,
			MaxConcurrency: 10,
		},
		L2: &L2EndpointConfig{
			L2EngineAddr: "http://localhost:8551",
		},
		L1ChainConfig: params.MainnetChainConfig,
		Rollup: rollup.Config{
			Genesis: rollup.Genesis{
				L1:     eth.BlockID{Hash: common.Hash{0x11}, Number: 424242},
				L2:     eth.BlockID{Hash: common.Hash{0x22}, Number: 1337},
				L2Time: 1_700_000_000,
				SystemConfig: eth.SystemConfig{
					BatcherAddr: common.Address{0x33},
					Overhead:    eth.Bytes32{0x01},
					Scalar:      eth.Bytes32{0x01},
					GasLimit:    1234567,
				},
			},
			BlockTime:              2,
			MaxSequencerDrift:      100,
			SeqWindowSize:          2,
			ChannelTimeoutBedrock:  123,
			L1ChainID:              big.NewInt(900),
			L2ChainID:              big.NewInt(901),
			BatchInboxAddress:      common.Address{0x44},
			DepositContractAddress: common.Address{0x55},
			L1SystemConfigAddress:  common.Address{0x66},
			ChainOpConfig: &opparams.OptimismConfig{
				EIP1559Elasticity:        6,
				EIP1559Denominator:       50,
				EIP1559DenominatorCanyon: ptr.New(uint64(250)),
			},
		},
	}
}

func TestConfigCheck_Valid(t *testing.T) {
	require.NoError(t, validConfig().Check())
}

// op-node's derivation pipeline and sequencer both assume one L2 block per block time, so a
// multi-block chain must be refused at startup rather than silently mis-derived. The refusal is
// pinned in both roles: a verifier that cannot follow such a chain is as fatal as a sequencer that
// cannot produce one.
func TestConfigCheck_RejectsMultiBlock(t *testing.T) {
	for _, sequencer := range []bool{false, true} {
		name := "verifier"
		if sequencer {
			name = "sequencer"
		}
		t.Run(name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Driver.SequencerEnabled = sequencer
			cfg.Rollup.KarstTime = ptr.New(uint64(0))
			cfg.Rollup.MultiBlockTime = ptr.New(cfg.Rollup.Genesis.L2Time + 10*cfg.Rollup.BlockTime)

			err := cfg.Check()
			require.ErrorIs(t, err, ErrMultiBlockUnsupported)
			require.Contains(t, err.Error(), "kona-node")
		})
	}
}
