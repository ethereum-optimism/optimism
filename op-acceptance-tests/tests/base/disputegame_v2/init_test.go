package disputegame_v2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	// TODO(#17810): Use the new v2 dispute game flag via presets.WithDisputeGameV2()
	//presets.DoMain(m, presets.WithMinimal(), presets.WithDisputeGameV2())
	presets.DoMain(m, presets.WithMinimal())
}
