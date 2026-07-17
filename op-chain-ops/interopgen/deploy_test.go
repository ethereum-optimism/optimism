package interopgen

import (
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestInitialDisputeAbsolutePrestates(t *testing.T) {
	cannonPrestate := common.HexToHash("0x01")
	cannonKonaPrestate := common.HexToHash("0x02")
	tests := []struct {
		name             string
		gameType         gameTypes.GameType
		selectedPrestate common.Hash
	}{
		{
			name:             "permissioned cannon selects cannon prestate",
			gameType:         gameTypes.PermissionedGameType,
			selectedPrestate: cannonPrestate,
		},
		{
			name:             "cannon kona selects kona prestate",
			gameType:         gameTypes.CannonKonaGameType,
			selectedPrestate: cannonKonaPrestate,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &L2Config{
				DisputeGameType:             uint32(test.gameType),
				DisputeAbsolutePrestate:     cannonPrestate,
				DisputeKonaAbsolutePrestate: cannonKonaPrestate,
			}
			selectedPrestate, fallbackPrestate := initialDisputeAbsolutePrestates(cfg)
			require.Equal(t, test.selectedPrestate, selectedPrestate)
			require.Equal(t, cannonPrestate, fallbackPrestate)
			if test.gameType == gameTypes.CannonKonaGameType {
				require.NotEqual(t, selectedPrestate, fallbackPrestate)
			}
		})
	}
}

func TestInteropAtGenesis(t *testing.T) {
	zero := hexutil.Uint64(0)
	nonzero := hexutil.Uint64(24)

	tests := []struct {
		name   string
		offset *hexutil.Uint64
		want   bool
	}{
		{"nil offset: Lagoon not scheduled", nil, false},
		{"zero offset: Lagoon activates interop at genesis", &zero, true},
		{"non-zero offset: Lagoon delayed activation", &nonzero, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, lagoonAtGenesis(tt.offset))
		})
	}
}

// devFeatureBitmapForL2Genesis sets the OptimismPortalInteropFlag when interop is enabled and the L2CMFlag when L2CM
// is enabled.
func TestDevFeatureBitmapForL2Genesis(t *testing.T) {
	interopOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.OptimismPortalInteropFlag)
	l2cmOnly := devfeatures.EnableDevFeature(common.Hash{}, devfeatures.L2CMFlag)
	both := devfeatures.EnableDevFeature(interopOnly, devfeatures.L2CMFlag)

	tests := []struct {
		name          string
		enableInterop bool
		useL2CM       bool
		want          common.Hash
	}{
		{"both disabled", false, false, common.Hash{}},
		{"interop only", true, false, interopOnly},
		{"L2CM only", false, true, l2cmOnly},
		{"both enabled", true, true, both},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, devFeatureBitmapForL2Genesis(tt.enableInterop, tt.useL2CM))
		})
	}
}
