package wait

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ForGamePublished waits until a game is published on L1 for the given l2BlockNumber.
func ForGamePublished(ctx context.Context, client *ethclient.Client, optimismPortalAddr common.Address, disputeGameFactoryAddr common.Address, l2BlockNumber *big.Int, l2BlockTimestamp uint64) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	l2BlockNumber = new(big.Int).Set(l2BlockNumber) // Don't clobber caller owned l2BlockNumber
	l2SequenceNumber := l2BlockTimestamp

	optimismPortal2Contract, err := bindingspreview.NewOptimismPortal2Caller(optimismPortalAddr, client)
	if err != nil {
		return 0, err
	}

	disputeGameFactoryContract, err := bindings.NewDisputeGameFactoryCaller(disputeGameFactoryAddr, client)
	if err != nil {
		return 0, err
	}

	respectedGameType, err := optimismPortal2Contract.RespectedGameType(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("failed to get respected game type: %w", err)
	}

	getL2BlockFromLatestGame := func() (*big.Int, error) {
		latestGame, err := withdrawals.FindLatestGameForGameType(ctx, disputeGameFactoryContract, respectedGameType)
		if err != nil {
			return big.NewInt(-1), nil
		}

		var gameSequenceNumber *big.Int
		switch gameTypes.GameType(respectedGameType) {
		case gameTypes.CannonKonaGameType, gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.FastGameType:
			gameSequenceNumber = new(big.Int).SetBytes(latestGame.ExtraData[0:32])
		case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
			gameSequenceNumber = new(big.Int).SetBytes(latestGame.ExtraData[1:9])
		default:
			return nil, fmt.Errorf("unsupported game type: %v", respectedGameType)
		}
		return gameSequenceNumber, nil
	}
	outputBlockNum, err := AndGet(ctx, time.Second, getL2BlockFromLatestGame, func(latestSeqnum *big.Int) bool {
		switch gameTypes.GameType(respectedGameType) {
		case gameTypes.CannonKonaGameType, gameTypes.CannonGameType, gameTypes.PermissionedGameType, gameTypes.FastGameType:
			return latestSeqnum.Cmp(l2BlockNumber) >= 0
		case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType:
			return bigs.Uint64Strict(latestSeqnum) >= l2SequenceNumber
		default:
			panic("unreachable") // given above predicate asserting unsupported games return errors
		}
	})
	if err != nil {
		return 0, err
	}
	return bigs.Uint64Strict(outputBlockNum), nil
}

// ForWithdrawalCheck waits until the withdrawal check in the portal succeeds.
func ForWithdrawalCheck(ctx context.Context, client *ethclient.Client, withdrawal crossdomain.Withdrawal, optimismPortalAddr common.Address, proofSubmitter common.Address) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	opts := &bind.CallOpts{Context: ctx}
	portal, err := bindingspreview.NewOptimismPortal2Caller(optimismPortalAddr, client)
	if err != nil {
		return fmt.Errorf("create portal caller: %w", err)
	}

	return For(ctx, time.Second, func() (bool, error) {
		wdHash, err := withdrawal.Hash()
		if err != nil {
			return false, fmt.Errorf("hash withdrawal: %w", err)
		}

		err = portal.CheckWithdrawal(opts, wdHash, proofSubmitter)
		return err == nil, nil
	})
}
