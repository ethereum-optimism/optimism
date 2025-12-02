package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestWithdrawal_Cannon(gt *testing.T) {
	withdrawal.TestWithdrawal(gt, gameTypes.CannonGameType)
}
