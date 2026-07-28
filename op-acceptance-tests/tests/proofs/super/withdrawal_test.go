package super

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestSuperPermissionedWithdrawal(gt *testing.T) {
	withdrawal.TestWithdrawal(gt, gameTypes.SuperPermissionedGameType)
}

func TestSuperCannonKonaWithdrawal(gt *testing.T) {
	withdrawal.TestWithdrawal(gt, gameTypes.SuperCannonKonaGameType)
}

func TestZKDisputeGameWithdrawal(gt *testing.T) {
	// TODO(#21463): Implement once we have a working kona-sp1 proposer
	gt.Skip("Skipping until the above todo is fixed")
	withdrawal.TestWithdrawal(gt, gameTypes.ZKDisputeGameType)
}
