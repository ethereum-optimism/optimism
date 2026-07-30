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
	// The kona-sp1-proposer precondition from the old TODO(#21463) is met,
	// but this test's single-chain withdrawal preset cannot install the ZK
	// game type: the devstack bring-up fails in UpgradeOPChain.s.sol before
	// any proposer runs. Unskip once the single-chain runtime supports the
	// ZK game type (or the withdrawal helper gains a supernode preset).
	gt.Skip("single-chain withdrawal preset cannot install the ZK game type (UpgradeOPChain.s.sol reverts during bring-up)")
	withdrawal.TestWithdrawal(gt, gameTypes.ZKDisputeGameType)
}
