package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

func TestWithdrawal_CannonKona(gt *testing.T) {
	withdrawal.TestWithdrawal(gt, faultTypes.CannonKonaGameType)
}
