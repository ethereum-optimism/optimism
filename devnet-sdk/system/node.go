package system

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// This will make sure that we implement the Node interface
	_ Node = (*node)(nil)
)

type node struct {
	rpcUrl string

	clients *clientManager
}

func newNode(rpcUrl string, clients *clientManager) *node {
	return &node{rpcUrl: rpcUrl, clients: clients}
}

func (n *node) GasPrice(ctx context.Context) (*big.Int, error) {
	client, err := n.clients.Client(n.rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	return client.SuggestGasPrice(ctx)
}

func (n *node) GasLimit(ctx context.Context, tx TransactionData) (uint64, error) {
	client, err := n.clients.Client(n.rpcUrl)
	if err != nil {
		return 0, fmt.Errorf("failed to get client: %w", err)
	}

	msg := ethereum.CallMsg{
		From:       tx.From(),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
	}
	estimated, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas: %w", err)
	}

	return estimated, nil
}

func (n *node) PendingNonceAt(ctx context.Context, address common.Address) (uint64, error) {
	client, err := n.clients.Client(n.rpcUrl)
	if err != nil {
		return 0, fmt.Errorf("failed to get client: %w", err)
	}
	return client.PendingNonceAt(ctx, address)
}

func (n *node) BlockByNumber(ctx context.Context, number *big.Int) (eth.BlockInfo, error) {
	client, err := n.clients.Client(n.rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	var block eth.BlockInfo
	if number != nil {
		block, err = client.InfoByNumber(ctx, number.Uint64())
	} else {
		block, err = client.InfoByLabel(ctx, eth.Unsafe)
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}
