package validators

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/constraints"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
)

type WalletGetter = func(context.Context) system.Wallet

func walletFundsValidator(chain system.Chain, minFunds types.Balance, userMarker interface{}) systest.PreconditionValidator {
	constraint := constraints.WithBalance(minFunds)
	return func(t systest.T, sys system.System) (context.Context, error) {
		wallets := chain.Wallets()

		for _, wallet := range wallets {
			if constraint.CheckWallet(wallet) {
				return context.WithValue(t.Context(), userMarker, wallet), nil
			}
		}

		return nil, fmt.Errorf("no available wallet with balance of at least of %s", minFunds)
	}
}

func AcquireL2WalletWithFunds(chainIndex uint64, minFunds types.Balance) (WalletGetter, systest.PreconditionValidator) {
	walletMarker := new(byte)
	return func(ctx context.Context) system.Wallet {
			return ctx.Value(walletMarker).(system.Wallet)
		}, func(t systest.T, sys system.System) (context.Context, error) {
			if len(sys.L2s()) <= int(chainIndex) {
				return nil, fmt.Errorf("chain index %d out of range, only %d L2 chains available", chainIndex, len(sys.L2s()))
			}
			chain := sys.L2s()[chainIndex]
			return walletFundsValidator(chain, minFunds, walletMarker)(t, sys)
		}
}

func AcquireL1WalletWithFunds(minFunds types.Balance) (WalletGetter, systest.PreconditionValidator) {
	walletMarker := new(byte)
	return func(ctx context.Context) system.Wallet {
			return ctx.Value(walletMarker).(system.Wallet)
		}, func(t systest.T, sys system.System) (context.Context, error) {
			chain := sys.L1()
			return walletFundsValidator(chain, minFunds, walletMarker)(t, sys)
		}
}
