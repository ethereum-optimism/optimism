// Package interopbridge implements the canonical cross-chain ETH bridge used by
// the interop test tools (check-lagoon and interop-smoke): it sends ETH through
// the SuperchainETHBridge predeploy, relays the resulting message itself, and
// verifies the recipient was credited. Both tools share this one implementation.
package interopbridge

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// maxRelayRetries caps submission/inclusion retry attempts. The effective bound
// is the caller's context deadline; this is just large enough not to give up
// before that deadline ends the retries.
const maxRelayRetries = 1024

// sendETHFn is the SuperchainETHBridge.sendETH(to, chainId) entrypoint.
var sendETHFn = w3.MustNewFunc("sendETH(address,uint256)", "bytes32")

// SendETHTrigger initiates a cross-chain ETH transfer through the
// SuperchainETHBridge predeploy. The bridged amount is the tx value.
type SendETHTrigger struct {
	Recipient   common.Address
	Destination eth.ChainID
}

func (t *SendETHTrigger) To() (*common.Address, error) {
	addr := predeploys.SuperchainETHBridgeAddr
	return &addr, nil
}

func (t *SendETHTrigger) EncodeInput() ([]byte, error) {
	return sendETHFn.EncodeArgs(t.Recipient, t.Destination.ToBig())
}

func (t *SendETHTrigger) AccessList() (types.AccessList, error) {
	return nil, nil
}

// BridgePlan estimates gas and retries submission and inclusion to ride out
// cross-chain propagation; the overall wait is capped by the caller's context.
func BridgePlan(cl apis.EthClient, key *ecdsa.PrivateKey) txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(cl),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(cl),
		txplan.WithAgainstLatestBlock(cl),
		txplan.WithEstimator(cl, true),
		txplan.WithRetrySubmission(cl, maxRelayRetries, retry.Exponential()),
		txplan.WithRetryInclusion(cl, maxRelayRetries, retry.Exponential()),
		txplan.WithBlockInclusionInfo(cl),
	)
}

// BridgeETH sends amount of ETH from key's account on src to recipient on dst via
// the SuperchainETHBridge predeploy, relays the message itself, and asserts the
// recipient's dst-chain balance grew by exactly amount. The wait is bounded by ctx.
func BridgeETH(ctx context.Context, logger log.Logger, src, dst apis.EthClient, dstChainID eth.ChainID, key *ecdsa.PrivateKey, amount eth.ETH, recipient common.Address) error {
	before, err := dst.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("read recipient balance: %w", err)
	}

	send := txintent.NewIntent[*SendETHTrigger, *txintent.InteropOutput](BridgePlan(src, key), txplan.WithValue(amount))
	send.Content.Set(&SendETHTrigger{Recipient: recipient, Destination: dstChainID})
	if _, err := send.PlannedTx.Success.Eval(ctx); err != nil {
		return fmt.Errorf("send ETH: %w", err)
	}
	logger.Info("sent ETH", "tx", send.PlannedTx.Included.Value().TxHash)

	relay := txintent.NewIntent[*txintent.RelayTrigger, *txintent.InteropOutput](BridgePlan(dst, key))
	relay.Content.DependOn(&send.Result)
	relay.Content.Fn(txintent.RelayIndexed(predeploys.L2toL2CrossDomainMessengerAddr, &send.Result, &send.PlannedTx.Included, 1))
	if _, err := relay.PlannedTx.Success.Eval(ctx); err != nil {
		return fmt.Errorf("relay ETH: %w", err)
	}
	logger.Info("relayed ETH", "tx", relay.PlannedTx.Included.Value().TxHash)

	after, err := dst.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("read recipient balance: %w", err)
	}
	if credited := new(big.Int).Sub(after, before); credited.Cmp(amount.ToBig()) != 0 {
		return fmt.Errorf("recipient credited %s wei, want %s wei", credited, amount.ToBig())
	}
	logger.Info("bridged ETH", "amount", amount, "recipient", recipient)
	return nil
}
