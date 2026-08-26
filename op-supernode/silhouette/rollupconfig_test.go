package silhouette

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func silhouetteParams() SilhouetteParams {
	return SilhouetteParams{
		L2ChainID:         big.NewInt(424250),
		L1ChainID:         big.NewInt(11155111),
		L1Genesis:         eth.BlockID{Hash: l1Hash(l1GenesisNum), Number: l1GenesisNum},
		L2Genesis:         eth.BlockID{Hash: common.HexToHash("0x1234"), Number: 0},
		L2Time:            l1GenesisT,
		BlockTime:         2,
		MaxSequencerDrift: 1800,
		GatedPortal:       common.HexToAddress("0x00000000000000000000000000000000000d0000"),
		BatchInbox:        common.HexToAddress("0xff00000000000000000000000000000000424250"),
		SystemConfigProxy: common.HexToAddress("0x00000000000000000000000000000000000c0000"),
		GasLimit:          30_000_000,
		EIP1559Params:     eth.Bytes8{0, 0, 0, 250, 0, 0, 0, 6},
		BatcherAddr:       common.HexToAddress("0x00000000000000000000000000000000000ba7c4"),
		BaseFeeScalar:     1368,
		BlobBaseFeeScalar: 810949,
	}
}

// TestRollupConfigIsSilhouetteShaped pins the three ways a silhouette chain's config differs from a
// stock one, since each is load-bearing somewhere far away.
func TestRollupConfigIsSilhouetteShaped(t *testing.T) {
	cfg, err := RollupConfigFor(silhouetteParams())
	require.NoError(t, err)

	// FINITE and short (DR-2): a dead prover must stop holding P hostage within an hour or two, not
	// the stock 12-hour default.
	require.Equal(t, uint64(DefaultSeqWindowSize), cfg.SeqWindowSize)
	require.Less(t, cfg.SeqWindowSize*12, uint64(2*3600))

	// The gated portal must be REAL: stock derivation reads deposit events from it and must find none.
	require.Equal(t, silhouetteParams().GatedPortal, cfg.DepositContractAddress)

	// Every fork at genesis, so there is no activation block to break a forced block's single-tx shape.
	for _, f := range []*uint64{cfg.RegolithTime, cfg.CanyonTime, cfg.DeltaTime, cfg.EcotoneTime,
		cfg.FjordTime, cfg.GraniteTime, cfg.HoloceneTime, cfg.IsthmusTime, cfg.JovianTime,
		cfg.KarstTime, cfg.LagoonTime} {
		require.NotNil(t, f)
		require.Equal(t, uint64(0), *f)
	}
	require.True(t, cfg.IsJovian(cfg.Genesis.L2Time), "Jovian must be live at genesis")

	// Karst and Lagoon at genesis keep the only upgrade bundles with non-zero upgrade gas out of
	// every post-genesis block, which is what preserves a forced block's single-transaction shape.
	require.True(t, cfg.IsKarst(cfg.Genesis.L2Time))

	// The generated artifact must round-trip as JSON: it is a file an operator hands to op-node.
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	var back struct {
		SeqWindowSize          uint64         `json:"seq_window_size"`
		DepositContractAddress common.Address `json:"deposit_contract_address"`
	}
	require.NoError(t, json.Unmarshal(raw, &back))
	require.Equal(t, cfg.SeqWindowSize, back.SeqWindowSize)
	require.Equal(t, cfg.DepositContractAddress, back.DepositContractAddress)
}

// TestRollupConfigRejectsUnsilhouetteShapes: the invariants are checked, not commented, because each
// violation breaks something in another lane.
func TestRollupConfigRejectsUnsilhouetteShapes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		break_ func(p *SilhouetteParams)
		msg    string
	}{
		{"window too long", func(p *SilhouetteParams) { p.SeqWindowSize = 3600 }, "longer than"},
		{"no base fee scalar", func(p *SilhouetteParams) { p.BaseFeeScalar = 0 }, "baseFeeScalar is required"},
		{"no gated portal", func(p *SilhouetteParams) { p.GatedPortal = common.Address{} }, "gatedPortal is required"},
		{"zero gas limit", func(p *SilhouetteParams) { p.GasLimit = 0 }, "gasLimit is required"},
		{
			// A zero denominator would make a forced block's extraData encode zeroes, and the next
			// block's base fee a division by zero (G2 D3/D7).
			"zero eip1559 denominator",
			func(p *SilhouetteParams) { p.EIP1559Params = eth.Bytes8{0, 0, 0, 0, 0, 0, 0, 6} },
			"non-zero denominator",
		},
		{"block time at L1 block time", func(p *SilhouetteParams) { p.BlockTime = 12 }, "well below"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := silhouetteParams()
			tc.break_(&p)
			_, err := RollupConfigFor(p)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.msg)
		})
	}
}
