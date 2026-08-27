package rollup

import (
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// konaRollupConfigFixture is parsed by kona-genesis in `test_op_node_rollup_config_parses`
// (rust/kona/crates/protocol/genesis/src/rollup.rs). kona's `RollupConfig` rejects unknown keys, so
// a `Config` field that kona does not model turns a rollup.json op-node accepts into one kona-node
// refuses to start on.
const konaRollupConfigFixture = "../../rust/kona/crates/protocol/genesis/tests/fixtures/op_node_rollup_config.json"

// updateFixtureEnvVar set to "1" rewrites the fixture instead of asserting against it.
const updateFixtureEnvVar = "UPDATE_KONA_ROLLUP_CONFIG_FIXTURE"

// fullyPopulatedConfig returns a Config with every field non-zero, so marshalling it emits every
// key op-node can write to a rollup.json — `omitempty` drops nothing. Adding a Config field means
// updating this, `fullyPopulatedChainConfig` (op-node/superchain), and the fixture.
//
// Values are distinct wherever two fields share a type or a packed encoding, so a decoder that
// reads the wrong half of a packed field, or crosses two fields, fails on the kona side.
func fullyPopulatedConfig() *Config {
	return &Config{
		Genesis: Genesis{
			L1:     eth.BlockID{Hash: common.HexToHash("0xaa"), Number: 100},
			L2:     eth.BlockID{Hash: common.HexToHash("0xbb"), Number: 1},
			L2Time: 1725557164,
			SystemConfig: eth.SystemConfig{
				BatcherAddr:          common.HexToAddress("0x1111111111111111111111111111111111111111"),
				Overhead:             eth.Bytes32(common.HexToHash("0xbc")),
				Scalar:               eth.Bytes32(common.HexToHash("0xa6fe0")),
				GasLimit:             30_000_000,
				EIP1559Params:        eth.Bytes8{0, 0, 0, 101, 0, 0, 0, 7},
				OperatorFeeParams:    eth.EncodeOperatorFeeParams(eth.OperatorFeeParams{Scalar: 23, Constant: 42}),
				MinBaseFee:           1_000_000,
				DAFootprintGasScalar: 400,
			},
		},
		BlockTime:              2,
		MaxSequencerDrift:      600,
		SeqWindowSize:          3600,
		ChannelTimeoutBedrock:  300,
		L1ChainID:              big.NewInt(1),
		L2ChainID:              big.NewInt(10),
		RegolithTime:           ptr.New(uint64(1)),
		CanyonTime:             ptr.New(uint64(2)),
		DeltaTime:              ptr.New(uint64(3)),
		EcotoneTime:            ptr.New(uint64(4)),
		FjordTime:              ptr.New(uint64(5)),
		GraniteTime:            ptr.New(uint64(6)),
		HoloceneTime:           ptr.New(uint64(7)),
		IsthmusTime:            ptr.New(uint64(8)),
		JovianTime:             ptr.New(uint64(9)),
		KarstTime:              ptr.New(uint64(10)),
		KeepKarstUpgradeGas:    true,
		LagoonTime:             ptr.New(uint64(11)),
		BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000010"),
		DepositContractAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		L1SystemConfigAddress:  common.HexToAddress("0x3333333333333333333333333333333333333333"),
		ChainOpConfig: &opparams.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New(uint64(250)),
		},
		AltDAConfig: &AltDAConfig{
			DAChallengeAddress: common.HexToAddress("0x4444444444444444444444444444444444444444"),
			CommitmentType:     "KeccakCommitment",
			MaxInputSize:       ptr.New(uint64(130_672)),
			DAChallengeWindow:  160,
			DAResolveWindow:    180,
		},
		PectraBlobScheduleTime: ptr.New(uint64(12)),
	}
}

// requireAllJSONFieldsSet asserts every marshalled field of v is non-zero, so `omitempty` drops
// nothing. Fields tagged `json:"-"` are never emitted and are exempt.
func requireAllJSONFieldsSet(t *testing.T, v any) {
	rv := reflect.ValueOf(v)
	typ := rv.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Tag.Get("json") == "-" {
			continue
		}
		require.Falsef(t, rv.Field(i).IsZero(),
			"%s field %q is zero, so marshalling omits it: give it a value in fullyPopulatedConfig",
			typ.Name(), field.Name)
	}
}

// TestKonaRollupConfigFixture keeps the rollup.json kona-genesis parses in sync with what op-node
// emits. It fails when Config gains a field, which is the point: the new key has to be modelled in
// kona's RollupConfig before a rollup.json carrying it will load in kona-node.
func TestKonaRollupConfigFixture(t *testing.T) {
	cfg := fullyPopulatedConfig()

	// Every struct that reaches the JSON, so a new `omitempty` field nested inside one of them
	// cannot stay unemitted — an unemitted key leaves the fixture unchanged and the drift unseen.
	requireAllJSONFieldsSet(t, *cfg)
	requireAllJSONFieldsSet(t, cfg.Genesis)
	requireAllJSONFieldsSet(t, cfg.Genesis.SystemConfig)
	requireAllJSONFieldsSet(t, *cfg.ChainOpConfig)
	requireAllJSONFieldsSet(t, *cfg.AltDAConfig)

	require.NoError(t, cfg.Check(), "the fixture must be a config op-node itself accepts")

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')

	if os.Getenv(updateFixtureEnvVar) == "1" {
		require.NoError(t, os.WriteFile(konaRollupConfigFixture, data, 0o644))
		t.Skip("rewrote " + konaRollupConfigFixture)
	}

	fixture, err := os.ReadFile(konaRollupConfigFixture)
	require.NoError(t, err)
	require.Equal(t, string(fixture), string(data),
		"%s is stale: model any new key in kona's RollupConfig, then regenerate with %s=1 go test ./op-node/rollup -run TestKonaRollupConfigFixture",
		konaRollupConfigFixture, updateFixtureEnvVar)
}
