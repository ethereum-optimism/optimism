package system

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// This will make sure that we implement the Node interface
	_ Node = (*node)(nil)
)

type node struct {
	rpcUrl   string
	clients  *clientManager
	mu       sync.Mutex
	registry interfaces.ContractsRegistry
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

func (n *node) Client() (*sources.EthClient, error) {
	return n.clients.Client(n.rpcUrl)
}

func (n *node) GethClient() (*ethclient.Client, error) {
	return n.clients.GethClient(n.rpcUrl)
}

func (n *node) ContractsRegistry() interfaces.ContractsRegistry {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.registry != nil {
		return n.registry
	}
	client, err := n.clients.GethClient(n.rpcUrl)
	if err != nil {
		return contracts.NewEmptyRegistry()
	}

	n.registry = contracts.NewClientRegistry(client)
	return n.registry
}

func (n *node) RPCURL() string {
	return n.rpcUrl
}

func (n *node) SupportsEIP(ctx context.Context, eip uint64) bool {
	client, err := n.Client()
	if err != nil {
		return false
	}

	switch eip {
	case 1559:
		return checkHeader(ctx, client, func(h eth.BlockInfo) bool {
			return h.BaseFee() != nil
		})
	case 4844:
		return checkHeader(ctx, client, func(h eth.BlockInfo) bool {
			return h.ExcessBlobGas() != nil
		})
	}
	return false
}
