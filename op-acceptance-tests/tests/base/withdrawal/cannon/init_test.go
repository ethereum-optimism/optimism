package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

func TestMain(m *testing.M) {
	withdrawal.InitWithGameType(m, types.CannonGameType)
}
