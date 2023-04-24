package kvstore

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-program/host/config"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestLocalPreimageSource(t *testing.T) {
	cfg := &config.Config{
		Rollup:             &chaincfg.Goerli,
		L1Head:             common.HexToHash("0x1111"),
		L2Head:             common.HexToHash("0x2222"),
		L2Claim:            common.HexToHash("0x3333"),
		L2ClaimBlockNumber: 1234,
		L2ChainConfig:      params.GoerliChainConfig,
	}
	source := NewLocalPreimageSource(cfg)
	tests := []struct {
		name     string
		key      common.Hash
		expected []byte
	}{
		{"L1Head", L1HeadKey, cfg.L1Head.Bytes()},
		{"L2Head", L2HeadKey, cfg.L2Head.Bytes()},
		{"L2Claim", L2ClaimKey, cfg.L2Claim.Bytes()},
		{"L2ClaimBlockNumber", L2ClaimBlockNumberKey, binary.BigEndian.AppendUint64(nil, cfg.L2ClaimBlockNumber)},
		{"Rollup", RollupKey, asJson(t, cfg.Rollup)},
		{"ChainConfig", L2ChainConfigKey, asJson(t, cfg.L2ChainConfig)},
		{"Unknown", preimage.LocalIndexKey(1000).PreimageKey(), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := source.Get(test.key)
			if test.expected == nil {
				require.ErrorIs(t, err, ErrNotFound)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.expected, result)
		})
	}
}

func asJson(t *testing.T, v any) []byte {
	d, err := json.Marshal(v)
	require.NoError(t, err)
	return d
}
