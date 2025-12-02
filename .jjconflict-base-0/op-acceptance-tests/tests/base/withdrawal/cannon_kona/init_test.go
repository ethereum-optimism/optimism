package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestMain(m *testing.M) {
	withdrawal.InitWithGameType(m, gameTypes.CannonKonaGameType)
}
