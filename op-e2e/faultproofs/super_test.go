package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum/go-ethereum/common"
)

func TestCreateSuperCannonGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()
	sys, disputeGameFactory, _ := StartInteropFaultDisputeSystem(t, WithAllocType(config.AllocTypeMTCannon))
	sys.L2IDs()
	game := disputeGameFactory.StartSuperCannonGame(ctx, common.Hash{0x01})
	game.LogGameData(ctx)
}
