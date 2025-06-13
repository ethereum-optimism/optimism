package proofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestChallengerPlaysGame(gt *testing.T) {
	// Setup
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	sys.L1Network.WaitForOnline()

	badClaim := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	attacker := fundAttackerWallet(t, sys, eth.OneEther.Mul(2))
	helper := proofs.HelperFromInteropPreset(t, sys, sys.L2ChainA)

	game := helper.StartSuperCannonGame(attacker, badClaim)

	claim := game.GetRootClaim()

	claim.WaitForCounterClaim()
}

func fundAttackerWallet(t devtest.T, sys *presets.SimpleInterop, fundingAmount eth.ETH) *dsl.EOA {
	wallet := sys.Wallet.NewEOA(sys.L1EL)
	initialBalance := sys.FunderL1.FundAtLeast(wallet, fundingAmount)
	require.GreaterOrEqual(t, initialBalance.ToBig().Int64(), fundingAmount.ToBig().Int64())

	return wallet
}

//func playGameToDepth(t devtest.T, attacker *dsl.EOA, game *bindings.FaultDisputeGame, depth int) {
//	// Wait for first counter-claim
//	latestClaim := waitForCounterClaim(t, game, big.NewInt(0))
//
//	// TOOD - clean this up with some helpers
//	for uint64(cTypes.NewPositionFromGIndex(latestClaim.Position).Depth()) < uint64(depth) {
//		// TODO - pass function that can generate the claim we want
//		attackClaim(t, attacker, game, latestClaim, badClaim)
//		//latestClaim = waitForCounterClaim(t, game,)
//	}
//
//}
//
//func attackClaim(t devtest.T, attacker *dsl.EOA, game *bindings.FaultDisputeGame, claimToCounter bindings.ClaimData, newClaim common.Hash) {
//	// TODO - rework claim object so that it holds a `Position`
//	newPosition := cTypes.NewPositionFromGIndex(claimToCounter.Position).Attack().ToGIndex()
//	requiredBond := contract.Read(game.GetRequiredBond(newPosition))
//
//	receipt := contract.Write(attacker, game.Attack(claimToCounter.Claim, claimToCounter.Position, newClaim), txplan.WithValue(requiredBond))
//	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
//}
//
//func waitForCounterClaim(t devtest.T, game *bindings.FaultDisputeGame, targetClaimIndex *big.Int) bindings.ClaimData {
//	var counterClaim bindings.ClaimData
//	lastCheckedClaim := int64(0)
//	require.Eventually(t, func() bool {
//		t.Logf("Waiting for counter to claim at index %d", targetClaimIndex)
//		lastClaimIndex := contract.Read(game.ClaimDataLen()).Int64() - 1
//		if lastClaimIndex <= lastCheckedClaim {
//			// Nothing new to check
//			return false
//		}
//
//		// Check new claims to see if they counter our target claim
//		t.Logf("Checking claims %d through %d", lastCheckedClaim+1, lastClaimIndex)
//		for i := lastCheckedClaim + 1; i <= lastClaimIndex; i++ {
//			claim := contract.Read(game.ClaimData(big.NewInt(i)))
//			if claim.ParentIndex.Cmp(targetClaimIndex) == 0 {
//				counterClaim = claim
//				return true
//			}
//		}
//
//		lastCheckedClaim = lastClaimIndex
//		return false
//
//	}, 5*time.Minute, 5*time.Second)
//
//	return counterClaim
//}
//
//
//
//func createNewGame(t devtest.T, sys *presets.SimpleInterop, attacker *dsl.EOA) *bindings.FaultDisputeGame {
//	l1Client := sys.L1EL.Escape().EthClient()
//	dgfAddr := sys.L2ChainA.Escape().Deployment().DisputeGameFactoryProxyAddr()
//	dgf := bindings.NewDisputeGameFactory(bindings.WithClient(l1Client), bindings.WithTo(dgfAddr), bindings.WithTest(t))
//
//	// Pull some metadata we need to construct a new game
//	gameType := uint32(cTypes.SuperCannonGameType)
//	anchorRoot := getAnchorRoot(t, sys)
//	requiredBonds := contract.Read(dgf.InitBonds(gameType))
//
//	l2SeqNum := big.NewInt(0).Add(anchorRoot.L2SeqNum, big.NewInt(10))
//	extraData := common.BigToHash(l2SeqNum).Bytes()
//	receipt := contract.Write(attacker, dgf.Create(gameType, badClaim, extraData), txplan.WithValue(requiredBonds.ToBig()))
//
//	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
//
//	// Extract new game contract from the logs
//	createLogs, err := dgf.DecodeDisputeGameCreatedLogs(receipt)
//	require.NoError(t, err)
//	require.Equal(t, 1, len(createLogs))
//
//	gameAddr := createLogs[0].DisputeProxy
//	return bindings.NewFaultDisputeGame(bindings.WithClient(l1Client), bindings.WithTo(gameAddr), bindings.WithTest(t))
//}
//
//func getAnchorRoot(t devtest.T, sys *presets.SimpleInterop) bindings.AnchorRoot {
//	l1Client := sys.L1EL.Escape().EthClient()
//	sysConfigAddr := sys.L2ChainA.Escape().Deployment().SystemConfigProxyAddr()
//	sysConfig := bindings.NewSystemConfig(bindings.WithClient(l1Client), bindings.WithTo(sysConfigAddr), bindings.WithTest(t))
//
//	portalAddr := contract.Read(sysConfig.OptimismPortal())
//	portal := bindings.NewPortal2(bindings.WithClient(l1Client), bindings.WithTo(portalAddr), bindings.WithTest(t))
//
//	asrAddr := contract.Read(portal.AnchorStateRegistry())
//	asr := bindings.NewAnchorStateRegistry(bindings.WithClient(l1Client), bindings.WithTo(asrAddr), bindings.WithTest(t))
//
//	return contract.Read(asr.GetAnchorRoot())
//}
