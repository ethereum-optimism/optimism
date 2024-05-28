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

// ForOutputRootPublished waits until there is an output published for an L2 block number larger than the supplied l2BlockNumber
// This function polls and can block for a very long time if used on mainnet.
// This returns the block number to use for proof generation.
func ForOutputRootPublished(ctx context.Context, client *ethclient.Client, l2OutputOracleAddr common.Address, l2BlockNumber *big.Int) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	l2BlockNumber = new(big.Int).Set(l2BlockNumber) // Don't clobber caller owned l2BlockNumber
	opts := &bind.CallOpts{Context: ctx}

	l2OO, err := bindings.NewL2OutputOracleCaller(l2OutputOracleAddr, client)
	if err != nil {
		return 0, err
	}

	getL2BlockFromLatestOutput := func() (*big.Int, error) { return l2OO.LatestBlockNumber(opts) }
	outputBlockNum, err := AndGet(ctx, time.Second, getL2BlockFromLatestOutput, func(latest *big.Int) bool {
		return latest.Cmp(l2BlockNumber) >= 0
	})
	if err != nil {
		return 0, err
	}
	return outputBlockNum.Uint64(), nil
}

// ForFinalizationPeriod waits until the L1 chain has progressed far enough that l1ProvingBlockNum has completed
// the finalization period.
// This functions polls and can block for a very long time if used on mainnet.
func ForFinalizationPeriod(ctx context.Context, client *ethclient.Client, l1ProvingBlockNum *big.Int, l2OutputOracleAddr common.Address) error {
	l1ProvingBlockNum = new(big.Int).Set(l1ProvingBlockNum) // Don't clobber caller owned l1ProvingBlockNum
	opts := &bind.CallOpts{Context: ctx}

	// Load finalization period
	l2OO, err := bindings.NewL2OutputOracleCaller(l2OutputOracleAddr, client)
	if err != nil {
		return fmt.Errorf("create L2OOCaller: %w", err)
	}
	finalizationPeriod, err := l2OO.FINALIZATIONPERIODSECONDS(opts)
	if err != nil {
		return fmt.Errorf("get finalization period: %w", err)
	}

	provingHeader, err := client.HeaderByNumber(ctx, l1ProvingBlockNum)
	if err != nil {
		return fmt.Errorf("retrieve proving header: %w", err)
	}

	targetTimestamp := new(big.Int).Add(new(big.Int).SetUint64(provingHeader.Time), finalizationPeriod)
	targetTime := time.Unix(targetTimestamp.Int64(), 0)
	// Assume clock is relatively correct
	time.Sleep(time.Until(targetTime))
	// Poll for L1 Block to have a time greater than the target time
	return For(ctx, time.Second, func() (bool, error) {
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return false, fmt.Errorf("retrieve latest header: %w", err)
		}
		return header.Time > targetTimestamp.Uint64(), nil
	})
}

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
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
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
