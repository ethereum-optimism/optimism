package transactions

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/delegatecallproxy"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// DelegateCallWithSetCode delegatecalls target with data in the context of the EOA of privateKey:
// an EIP-7702 SetCode transaction delegates the EOA to a DelegateCallProxy, so the EOA can act as
// an owner without a transfer of ownership.
func DelegateCallWithSetCode(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, target common.Address, data []byte) error {
	sender := crypto.PubkeyToAddress(privateKey.PublicKey)
	proxyAddr, err := deployDelegateCallProxy(ctx, client, privateKey, sender)
	if err != nil {
		return err
	}

	proxyABI, err := delegatecallproxy.DelegatecallproxyMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("failed to load the DelegateCallProxy ABI: %w", err)
	}
	calldata, err := proxyABI.Pack("executeDelegateCall", target, data)
	if err != nil {
		return fmt.Errorf("failed to encode executeDelegateCall: %w", err)
	}

	tx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(privateKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithType(types.SetCodeTxType),
		txplan.WithTo(&sender),
		txplan.WithAuthorizationTo(proxyAddr),
		txplan.WithData(calldata),
		// eth_estimateGas does not handle EIP-7702 authorizations, so use the EIP-7825 maximum.
		txplan.WithGasLimit(params.MaxTxGas),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(txplan.FromGethReceipts(client), 10, retry.Exponential()),
	)
	receipt, err := tx.Included.Eval(ctx)
	if err != nil {
		return fmt.Errorf("failed to include the SetCode transaction: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("delegatecall via SetCode failed in transaction %s", receipt.TxHash)
	}
	return nil
}

// deployDelegateCallProxy deploys the proxy that the EOA delegates to.
func deployDelegateCallProxy(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, owner common.Address) (common.Address, error) {
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get chain ID: %w", err)
	}
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create transactor: %w", err)
	}
	transactOpts.Context = ctx
	addr, tx, _, err := delegatecallproxy.DeployDelegatecallproxy(transactOpts, client, owner)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy DelegateCallProxy: %w", err)
	}
	if _, err := wait.ForReceiptOK(ctx, client, tx.Hash()); err != nil {
		return common.Address{}, fmt.Errorf("DelegateCallProxy deployment tx was not included successfully: %w", err)
	}
	return addr, nil
}
