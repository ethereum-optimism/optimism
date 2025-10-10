package disputegame_v2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal(), presets.WithDisputeGameV2())
}
