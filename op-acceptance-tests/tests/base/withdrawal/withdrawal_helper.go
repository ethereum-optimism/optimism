package withdrawal

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// ForGamePublished waits until a game is published on L1 for the given l2BlockNumber.
func ForGamePublished(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient, optimismPortalAddr common.Address, disputeGameFactoryAddr common.Address, l2BlockNumber *big.Int) (uint64, error) {
	_, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	var outputBlockNum *big.Int
	require.Eventually(t, func() bool {
		latestGame, err := utils.FindLatestGame(t, l2Chain, l1Client)
		if err != nil {
			return false
		}
		outputBlockNum = new(big.Int).SetBytes(latestGame.ExtraData[0:32])
		return outputBlockNum.Cmp(l2BlockNumber) >= 0
	}, 30*time.Second, 100*time.Millisecond, "latest game not found")
	return outputBlockNum.Uint64(), nil
}

func ProveWithdrawal(t devtest.T, portal *bindings.OptimismPortal2, optimismPortalAddr common.Address, sys *presets.Minimal, l1User *dsl.EOA, l2WithdrawalReceipt *types.Receipt) (withdrawals.ProvenWithdrawalParameters, *types.Receipt) {
	l1Client := sys.L1EL.Escape().EthClient()
	l2Client := sys.L2EL.Escape().EthClient()

	// Wait for another block to be mined so that the timestamp increases. Otherwise,
	// proveWithdrawalTransaction gas estimation may fail because the current timestamp is the same
	// as the dispute game creation timestamp.
	sys.L1Network.WaitForBlock()

	var proveReceipt *types.Receipt
	var params withdrawals.ProvenWithdrawalParameters

	t.Logf("proveWithdrawal: proving withdrawal...")
	require.Eventually(t, func() bool {
		dgfaddr := contract.Read(portal.DisputeGameFactoryAddr())
		_, err := ForGamePublished(t, sys.L2Networks()[0], l1Client, optimismPortalAddr, dgfaddr, l2WithdrawalReceipt.BlockNumber)
		require.NoError(t, err)
		params, err = ProveWithdrawalParameters(t, sys.L2Chain, l1Client, l2Client, l2WithdrawalReceipt)
		require.NoError(t, err)
		tx := bindings.WithdrawalTransaction{
			Nonce:    params.Nonce,
			Sender:   params.Sender,
			Target:   params.Target,
			Value:    params.Value,
			GasLimit: params.GasLimit,
			Data:     params.Data,
		}
		proof := bindings.OutputRootProof{
			Version:                  [32]byte{},
			StateRoot:                params.OutputRootProof.StateRoot,
			MessagePasserStorageRoot: params.OutputRootProof.MessagePasserStorageRoot,
			LatestBlockhash:          params.OutputRootProof.LatestBlockhash,
		}
		call := portal.ProveWithdrawalTransaction(tx, params.L2OutputIndex, proof, params.WithdrawalProof)
		proveReceipt, err = contractio.Write(call, t.Ctx(), l1User.Plan())
		if err != nil {
			return false
		}
		return proveReceipt.Status == types.ReceiptStatusSuccessful
	}, 120*time.Second, 100*time.Millisecond, "withdrawal proof failed")
	require.Equal(t, 2, len(proveReceipt.Logs)) // emit WithdrawalProven, WithdrawalProvenExtension1

	return params, proveReceipt
}

func ProveWithdrawalParameters(t devtest.T, l2Chain *dsl.L2Network, l1Client apis.EthClient, l2Client apis.EthClient, l2WithdrawalReceipt *types.Receipt) (withdrawals.ProvenWithdrawalParameters, error) {
	return utils.ProveWithdrawalParameters(t, l2Chain, l1Client, l2Client, l2WithdrawalReceipt)
}

func FinalizeWithdrawal(t devtest.T, portal *bindings.OptimismPortal2, sys *presets.Minimal, l1User *dsl.EOA, l2WithdrawalReceipt *types.Receipt, params withdrawals.ProvenWithdrawalParameters) (*types.Receipt, *types.Receipt, *types.Receipt) {
	wd := crossdomain.Withdrawal{
		Nonce:    params.Nonce,
		Sender:   &params.Sender,
		Target:   &params.Target,
		Value:    params.Value,
		GasLimit: params.GasLimit,
		Data:     params.Data,
	}

	l1Client := sys.L1EL.Escape().EthClient()
	wdHash, err := wd.Hash()
	require.NoError(t, err)

	game := contract.Read(portal.ProvenWithdrawals(wdHash, l1User.Address()))
	// basic sanity check for unix timestamp
	require.Greater(t, game.Timestamp, uint64(1700000000))

	gameContract, err := contracts.NewFaultDisputeGameContract(t.Ctx(), metrics.NoopContractMetrics, game.DisputeGameProxy, l1Client.NewMultiCaller(batching.DefaultBatchSize))
	require.NoError(t, err)

	timedCtx, cancel := context.WithTimeout(t.Ctx(), 120*time.Second)
	defer cancel()
	require.NoError(t, wait.For(timedCtx, time.Second, func() (bool, error) {
		// First check if the game is in a resolvable state
		status, err := gameContract.GetStatus(t.Ctx())
		if err != nil {
			return false, err
		}
		if status != gameTypes.GameStatusInProgress {
			return false, fmt.Errorf("game is not in progress: %v", status)
		}
		// Try to resolve the claim
		err = gameContract.CallResolveClaim(t.Ctx(), 0)
		if err != nil {
			t.Logf("Could not resolve dispute game claim: %v", err)
			return false, nil
		}
		return true, nil
	}))

	t.Logf("FinalizeWithdrawal: resolveClaim...")
	tx, err := gameContract.ResolveClaimTx(0)
	require.NoError(t, err)
	resolveClaimReceipt := l1User.Transact(
		l1User.Plan(),
		txplan.WithTo(tx.To),
		txplan.WithValue(tx.Value),
		txplan.WithGasLimit(tx.GasLimit),
		txplan.WithData(tx.TxData),
	)

	t.Logf("FinalizeWithdrawal: resolve...")
	tx, err = gameContract.ResolveTx()
	require.NoError(t, err)

	resolveReceipt := l1User.Transact(
		l1User.Plan(),
		txplan.WithTo(tx.To),
		txplan.WithValue(tx.Value),
		txplan.WithGasLimit(tx.GasLimit),
		txplan.WithData(tx.TxData),
	)

	receipt := resolveReceipt.Included.Value()
	require.Equal(t, 1, len(receipt.Logs)) // emit Resolved

	if resolveReceipt.Included.Value().Status == types.ReceiptStatusFailed {
		t.Logf("resolve failed (tx: %s)! But game may have resolved already. Checking now...", resolveReceipt.Included.Value().TxHash)
		// it may have failed because someone else front-ran this by calling `resolve()` first.
		status, err := gameContract.GetStatus(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, gameTypes.GameStatusDefenderWon, status, "game must have resolved with defender won")
		t.Logf("resolve was not needed, the game was already resolved")
	}

	// Finalize withdrawal
	t.Logf("FinalizeWithdrawal: finalizing withdrawal...")
	var finalizeWithdrawalReceipt *types.Receipt
	require.Eventually(t, func() bool {
		finalizeWithdrawalReceipt, err = contractio.Write(portal.FinalizeWithdrawalTransaction(wd.WithdrawalTransaction()), t.Ctx(), l1User.Plan())
		if err != nil {
			return false
		}
		return types.ReceiptStatusSuccessful == finalizeWithdrawalReceipt.Status
	}, 60*time.Second, 100*time.Millisecond, "finalize withdrawal failed")

	return finalizeWithdrawalReceipt, resolveClaimReceipt.Included.Value(), resolveReceipt.Included.Value()
}
