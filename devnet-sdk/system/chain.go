package system

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts"
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/interfaces"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// This will make sure that we implement the Chain interface
	_ Chain = (*chain)(nil)
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
	rpcUrl      string
	users       map[string]Wallet
	clients     *clientManager
	registry    interfaces.ContractsRegistry
	mu          sync.Mutex
	node        Node
	chainConfig *params.ChainConfig
	addresses   descriptors.AddressMap
}

func (c *chain) Node() Node {
	return c.node
}

func (c *chain) Client() (*sources.EthClient, error) {
	return c.clients.Client(c.rpcUrl)
}

func (c *chain) GethClient() (*ethclient.Client, error) {
	return c.clients.GethClient(c.rpcUrl)
}

func newChain(chainID string, rpcUrl string, users map[string]Wallet, chainConfig *params.ChainConfig, addresses descriptors.AddressMap) *chain {
	clients := newClientManager()
	chain := &chain{
		id:          chainID,
		rpcUrl:      rpcUrl,
		users:       users,
		clients:     clients,
		node:        newNode(rpcUrl, clients),
		chainConfig: chainConfig,
		addresses:   addresses,
	}
	return chain
}

func (c *chain) ContractsRegistry() interfaces.ContractsRegistry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.registry != nil {
		return c.registry
	}
	client, err := c.clients.GethClient(c.rpcUrl)
	if err != nil {
		return contracts.NewEmptyRegistry()
	}

	c.registry = contracts.NewClientRegistry(client)
	return c.registry
}

func (c *chain) RPCURL() string {
	return c.rpcUrl
}

// Wallet returns the first wallet which meets all provided constraints, or an
// error.
// Typically this will be one of the pre-funded wallets associated with
// the deployed system.
func (c *chain) Wallets(ctx context.Context) ([]Wallet, error) {
	wallets := []Wallet{}

	for _, user := range c.users {
		wallets = append(wallets, user)
	}

	return wallets, nil
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

func checkHeader(ctx context.Context, client *sources.EthClient, check func(eth.BlockInfo) bool) bool {
	info, err := client.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return false
	}
	return check(info)
}

func (c *chain) SupportsEIP(ctx context.Context, eip uint64) bool {
	client, err := c.Client()
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

func (c *chain) Config() (*params.ChainConfig, error) {
	if c.chainConfig == nil {
		return nil, fmt.Errorf("chain config not configured on L1 chains yet")
	}
	return c.chainConfig, nil
}

func (c *chain) Addresses() descriptors.AddressMap {
	return c.addresses
}

func chainFromDescriptor(d *descriptors.Chain) (Chain, error) {
	// TODO: handle incorrect descriptors better. We could panic here.
	firstNodeRPC := d.Nodes[0].Services["el"].Endpoints["rpc"]
	rpcURL := fmt.Sprintf("http://%s:%d", firstNodeRPC.Host, firstNodeRPC.Port)

	c := newChain(d.ID, rpcURL, nil, d.Config, d.Addresses) // Create chain first

	users := make(map[string]Wallet)
	for key, w := range d.Wallets {
		// TODO: The assumption that the wallet will necessarily be used on chain `d` may
		// be problematic if the L2 admin wallets are to be used to sign L1 transactions.
		// TBD on whether they belong somewhere other than `d.Wallets`.
		k, err := newWallet(w.PrivateKey, w.Address, c)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		users[key] = k
	}
	c.users = users // Set users after creation

	return c, nil
}
