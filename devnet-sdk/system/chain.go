package system

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	coreTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

// this is to differentiate between op-geth and go-ethereum
type opBlock interface {
	WithdrawalsRoot() *common.Hash
}

var (
	// This will make sure that we implement the Chain interface
	_ Chain   = (*chain)(nil)
	_ L2Chain = (*l2Chain)(nil)

	// Make sure we're using op-geth in place of go-ethereum.
	// If you're wondering why this fails at compile time,
	// it's most likely because you're not using a "replace"
	// directive in your go.mod file.
	_ opBlock = (*coreTypes.Block)(nil)
)

// clientManager handles ethclient connections
type clientManager struct {
	mu          sync.RWMutex
	clients     map[string]*sources.EthClient
	gethClients map[string]*ethclient.Client
}

func newClientManager() *clientManager {
	return &clientManager{
		clients:     make(map[string]*sources.EthClient),
		gethClients: make(map[string]*ethclient.Client),
	}
}

func (m *clientManager) Client(rpcURL string) (*sources.EthClient, error) {
	m.mu.RLock()
	if client, ok := m.clients[rpcURL]; ok {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := m.clients[rpcURL]; ok {
		return client, nil
	}

	ethClCfg := sources.EthClientConfig{
		MaxRequestsPerBatch:   10,
		MaxConcurrentRequests: 10,
		ReceiptsCacheSize:     10,
		TransactionsCacheSize: 10,
		HeadersCacheSize:      10,
		PayloadsCacheSize:     10,
		BlockRefsCacheSize:    10,
		TrustRPC:              false,
		MustBePostMerge:       true,
		RPCProviderKind:       sources.RPCKindStandard,
		MethodResetDuration:   time.Minute,
	}
	rpcClient, err := rpc.DialContext(context.Background(), rpcURL)
	if err != nil {
		return nil, err
	}
	ethCl, err := sources.NewEthClient(client.NewBaseRPCClient(rpcClient), log.Root(), nil, &ethClCfg)
	if err != nil {
		return nil, err
	}
	m.clients[rpcURL] = ethCl
	return ethCl, nil
}

func (m *clientManager) GethClient(rpcURL string) (*ethclient.Client, error) {
	m.mu.RLock()
	if client, ok := m.gethClients[rpcURL]; ok {
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if client, ok := m.gethClients[rpcURL]; ok {
		return client, nil
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	m.gethClients[rpcURL] = client
	return client, nil
}

type chain struct {
	id          string
	wallets     WalletMap
	nodes       []Node
	chainConfig *params.ChainConfig
	addresses   AddressMap
}

func (c *chain) Nodes() []Node {
	return c.nodes
}

// Wallet returns the first wallet which meets all provided constraints, or an
// error.
// Typically this will be one of the pre-funded wallets associated with
// the deployed system.
func (c *chain) Wallets() WalletMap {
	return c.wallets
}

func (c *chain) ID() types.ChainID {
	if c.id == "" {
		return types.ChainID(big.NewInt(0))
	}
	id, ok := new(big.Int).SetString(c.id, 10)
	if !ok {
		return types.ChainID(big.NewInt(0))
	}
	return types.ChainID(id)
}

func (c *chain) Config() (*params.ChainConfig, error) {
	if c.chainConfig == nil {
		return nil, fmt.Errorf("chain config is nil")
	}
	return c.chainConfig, nil
}

func (c *chain) Addresses() AddressMap {
	return c.addresses
}

// SupportsEIP checks if the chain's first node supports the given EIP
func (c *chain) SupportsEIP(ctx context.Context, eip uint64) bool {
	if len(c.nodes) == 0 {
		return false
	}
	return c.nodes[0].SupportsEIP(ctx, eip)
}

func checkHeader(ctx context.Context, client *sources.EthClient, check func(eth.BlockInfo) bool) bool {
	info, err := client.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return false
	}
	return check(info)
}

func newNodesFromDescriptor(d *descriptors.Chain) []Node {
	clients := newClientManager()
	nodes := make([]Node, len(d.Nodes))
	for i, node := range d.Nodes {
		svc := node.Services["el"]
		name := svc.Name
		rpc := svc.Endpoints["rpc"]
		if rpc.Scheme == "" {
			rpc.Scheme = "http"
		}
		nodes[i] = newNode(fmt.Sprintf("%s://%s:%d", rpc.Scheme, rpc.Host, rpc.Port), name, clients)
	}
	return nodes
}

func newChainFromDescriptor(d *descriptors.Chain) (*chain, error) {
	// TODO: handle incorrect descriptors better. We could panic here.

	nodes := newNodesFromDescriptor(d)
	c := newChain(d.ID, nil, d.Config, AddressMap(d.Addresses), nodes) // Create chain first

	wallets, err := newWalletMapFromDescriptorWalletMap(d.Wallets, c)
	if err != nil {
		return nil, err
	}
	c.wallets = wallets

	return c, nil
}

func newChain(chainID string, wallets WalletMap, chainConfig *params.ChainConfig, addresses AddressMap, nodes []Node) *chain {
	chain := &chain{
		id:          chainID,
		wallets:     wallets,
		nodes:       nodes,
		chainConfig: chainConfig,
		addresses:   addresses,
	}
	return chain
}

func newL2ChainFromDescriptor(d *descriptors.L2Chain) (*l2Chain, error) {
	// TODO: handle incorrect descriptors better. We could panic here.

	nodes := newNodesFromDescriptor(&d.Chain)
	c := newL2Chain(d.ID, nil, nil, d.Config, AddressMap(d.L1Addresses), AddressMap(d.Addresses), nodes) // Create chain first

	l2Wallets, err := newWalletMapFromDescriptorWalletMap(d.Wallets, c)
	if err != nil {
		return nil, err
	}
	c.wallets = l2Wallets

	l1Wallets, err := newWalletMapFromDescriptorWalletMap(d.L1Wallets, c)
	if err != nil {
		return nil, err
	}
	c.l1Wallets = l1Wallets

	return c, nil
}

func newL2Chain(chainID string, l1Wallets WalletMap, l2Wallets WalletMap, chainConfig *params.ChainConfig, l1Addresses AddressMap, l2Addresses AddressMap, nodes []Node) *l2Chain {
	chain := &l2Chain{
		chain:       newChain(chainID, l2Wallets, chainConfig, l2Addresses, nodes),
		l1Addresses: l1Addresses,
		l1Wallets:   l1Wallets,
	}
	return chain
}

type l2Chain struct {
	*chain
	l1Addresses AddressMap
	l1Wallets   WalletMap
}

func (c *l2Chain) L1Addresses() AddressMap {
	return c.l1Addresses
}

func (c *l2Chain) L1Wallets() WalletMap {
	return c.l1Wallets
}
