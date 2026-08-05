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
	// TODO(#22174): the single-chain withdrawal preset cannot install the
	// ZK game type; devstack bring-up fails in UpgradeOPChain.s.sol before
	// any proposer runs. Unskip once the single-chain runtime supports the
	// ZK game type (or the withdrawal helper gains a supernode preset).
	gt.Skip("single-chain withdrawal preset cannot install the ZK game type (#22174)")
	withdrawal.TestWithdrawal(gt, gameTypes.ZKDisputeGameType)
}
