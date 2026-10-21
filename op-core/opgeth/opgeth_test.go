package opgeth

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/params/opgethtest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func schedules() map[string]forkstest.Schedule {
	out := map[string]forkstest.Schedule{"staggered": forkstest.Staggered()}
	for _, fork := range forkstest.TimestampELForks {
		out["through "+string(fork)] = forkstest.AtGenesisThrough(fork)
	}
	return out
}

func opChainConfig(s forkstest.Schedule) *opparams.ChainConfig {
	cfg := &opparams.ChainConfig{
		ChainID: big.NewInt(901),
		Optimism: &opparams.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New[uint64](250),
		},
		BedrockBlock: big.NewInt(0),
	}
	for _, fork := range forkstest.TimestampELForks {
		cfg.SetActivationTime(fork, s(fork))
	}
	return cfg
}

// TestChainConfigMatchesOpGeth checks ChainConfig against op-geth's OP Stack test config
// with the same fork schedule, built independently of op-core.
func TestChainConfigMatchesOpGeth(t *testing.T) {
	for name, s := range schedules() {
		t.Run(name, func(t *testing.T) {
			got := ChainConfig(opChainConfig(s))
			require.True(t, got.IsOptimism())
			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			wantJSON, err := json.Marshal(opgethtest.ChainConfig(t, 901, s))
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}

func TestChainConfigNilOptimism(t *testing.T) {
	cfg := opChainConfig(forkstest.Staggered())
	cfg.Optimism = nil
	require.Nil(t, ChainConfig(cfg).Optimism)
}

func fixture(s forkstest.Schedule) *opgenesis.Genesis {
	return &opgenesis.Genesis{
		Config:     opChainConfig(s),
		Nonce:      7,
		Timestamp:  1234,
		ExtraData:  eip1559.EncodeJovianExtraData(250, 6, 0),
		GasLimit:   30_000_000,
		Difficulty: big.NewInt(0),
		Mixhash:    common.Hash{0xaa},
		Coinbase:   predeploys.SequencerFeeVaultAddr,
		Alloc: types.GenesisAlloc{
			predeploys.L2ToL1MessagePasserAddr: {
				Code:    []byte{0x60, 0x00, 0x60, 0x00, 0xf3},
				Storage: map[common.Hash]common.Hash{{0x01}: {0x02}},
				Balance: big.NewInt(0),
				Nonce:   1,
			},
		},
		BaseFee:       big.NewInt(1_000_000_000),
		ExcessBlobGas: ptr.New[uint64](0),
		BlobGasUsed:   ptr.New[uint64](0),
	}
}

// TestGenesisToBlockMatchesOpGeth checks that op-geth, started on Genesis(g), computes
// the genesis block that g.ToBlock() computes.
func TestGenesisToBlockMatchesOpGeth(t *testing.T) {
	for name, s := range schedules() {
		t.Run(name, func(t *testing.T) {
			g := fixture(s)
			gg := Genesis(g)
			require.True(t, gg.Config.IsOptimism())
			want := gg.ToBlock()
			got := g.ToBlock()
			require.Equal(t, want.Header(), got.Header())
			require.Equal(t, want.Hash(), got.Hash())
			require.Equal(t, want.Withdrawals(), got.Withdrawals())
		})
	}
}

func TestGenesisNilConfig(t *testing.T) {
	g := fixture(forkstest.Staggered())
	g.Config = nil
	require.Nil(t, Genesis(g).Config)
}

// TestFromGethReceiptMirrorsDepositFields checks that the wrapper carries the deposit
// fields to the authoritative outer copies, where wrapping by hand leaves them nil.
func TestFromGethReceiptMirrorsDepositFields(t *testing.T) {
	gr := types.Receipt{
		Type:                  types.DepositTxType,
		Status:                types.ReceiptStatusSuccessful,
		CumulativeGasUsed:     42_000,
		Logs:                  []*types.Log{},
		DepositNonce:          ptr.New[uint64](7),
		DepositReceiptVersion: ptr.New(types.CanyonDepositReceiptVersion),
	}

	wrapped := FromGethReceipt(&gr)
	require.Equal(t, gr.DepositNonce, wrapped.DepositNonce)
	require.Equal(t, gr.DepositReceiptVersion, wrapped.DepositReceiptVersion)

	var wrappedBuf, handBuiltBuf, theirBuf bytes.Buffer
	optypes.Receipts{wrapped}.EncodeIndex(0, &wrappedBuf)
	optypes.Receipts{{Receipt: gr}}.EncodeIndex(0, &handBuiltBuf)
	types.Receipts{&gr}.EncodeIndex(0, &theirBuf)
	require.Equal(t, theirBuf.Bytes(), wrappedBuf.Bytes())
	require.NotEqual(t, handBuiltBuf.Bytes(), wrappedBuf.Bytes())

	var listBuf bytes.Buffer
	FromGethReceipts(types.Receipts{&gr}).EncodeIndex(0, &listBuf)
	require.Equal(t, wrappedBuf.Bytes(), listBuf.Bytes())
}
