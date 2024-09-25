package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
)

func IsSupportedStateVersion(version int) bool {
	return version == 1
}

func Init(ctx context.Context, env *Env, artifactsFS foundry.StatDirFs, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "init")
	lgr.Info("initializing pipeline")

	// Ensure the state version is supported.
	if !IsSupportedStateVersion(st.Version) {
		return fmt.Errorf("unsupported state version: %d", st.Version)
	}

	if st.Create2Salt == (common.Hash{}) {
		_, err := rand.Read(st.Create2Salt[:])
		if err != nil {
			return fmt.Errorf("failed to generate CREATE2 salt: %w", err)
		}
	}

	// If the state has never been applied, we don't need to perform
	// any additional checks.
	if st.AppliedIntent == nil {
		return nil
	}

	// If the state has been applied, we need to check if any immutable
	// fields have changed.
	if st.AppliedIntent.L1ChainID != intent.L1ChainID {
		return immutableErr("L1ChainID", st.AppliedIntent.L1ChainID, intent.L1ChainID)
	}

	if st.AppliedIntent.UseFaultProofs != intent.UseFaultProofs {
		return immutableErr("useFaultProofs", st.AppliedIntent.UseFaultProofs, intent.UseFaultProofs)
	}

	if st.AppliedIntent.UseAltDA != intent.UseAltDA {
		return immutableErr("useAltDA", st.AppliedIntent.UseAltDA, intent.UseAltDA)
	}

	if st.AppliedIntent.FundDevAccounts != intent.FundDevAccounts {
		return immutableErr("fundDevAccounts", st.AppliedIntent.FundDevAccounts, intent.FundDevAccounts)
	}

	l1ChainID, err := env.L1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get L1 chain ID: %w", err)
	}

	if l1ChainID.Cmp(intent.L1ChainIDBig()) != 0 {
		return fmt.Errorf("L1 chain ID mismatch: got %d, expected %d", l1ChainID, intent.L1ChainID)
	}

	deployerCode, err := env.L1Client.CodeAt(ctx, script.DeterministicDeployerAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get deployer code: %w", err)
	}
	if len(deployerCode) == 0 {
		return fmt.Errorf("deterministic deployer is not deployed on this chain - please deploy it first")
	}

	// TODO: validate individual

	return nil
}

func immutableErr(field string, was, is any) error {
	return fmt.Errorf("%s is immutable: was %v, is %v", field, was, is)
}
