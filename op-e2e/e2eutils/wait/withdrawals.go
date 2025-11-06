package wait

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ForGamePublished waits until a game is published on L1 for the given l2BlockNumber.
func ForGamePublished(ctx context.Context, client *ethclient.Client, optimismPortalAddr common.Address, disputeGameFactoryAddr common.Address, l2BlockNumber *big.Int) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	l2BlockNumber = new(big.Int).Set(l2BlockNumber) // Don't clobber caller owned l2BlockNumber

	optimismPortal2Contract, err := bindingspreview.NewOptimismPortal2Caller(optimismPortalAddr, client)
	if err != nil {
		return 0, err
	}

	disputeGameFactoryContract, err := bindings.NewDisputeGameFactoryCaller(disputeGameFactoryAddr, client)
	if err != nil {
		return 0, err
	}

	getL2BlockFromLatestGame := func() (*big.Int, error) {
		latestGame, err := withdrawals.FindLatestGame(ctx, disputeGameFactoryContract, optimismPortal2Contract)
		if err != nil {
			return big.NewInt(-1), nil
		}

		gameBlockNumber := new(big.Int).SetBytes(latestGame.ExtraData[0:32])
		return gameBlockNumber, nil
	}
	outputBlockNum, err := AndGet(ctx, time.Second, getL2BlockFromLatestGame, func(latest *big.Int) bool {
		return latest.Cmp(l2BlockNumber) >= 0
	})
	if err != nil {
		return 0, err
	}
	return outputBlockNum.Uint64(), nil
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
