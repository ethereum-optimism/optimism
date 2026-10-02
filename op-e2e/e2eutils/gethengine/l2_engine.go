package gethengine

import (
	"errors"
	"os"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/gethengine/engineapi"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	geth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// L2Engine is an in-process op-geth L2 execution engine serving a custom Engine API that lets
// tests choose the transactions of each block, without support for snap-sync.
type L2Engine struct {
	log log.Logger

	node *node.Node
	Eth  *geth.Ethereum

	// L2 evm / chain
	l2Chain *core.BlockChain

	EngineApi *engineapi.L2EngineAPI
}

func NewL2Engine(t helpers.Testing, log log.Logger, genesis *core.Genesis, jwtPath string) *L2Engine {
	n, ethBackend, apiBackend := newBackend(t, genesis, jwtPath)
	engineApi := engineapi.NewL2EngineAPI(log, apiBackend, ethBackend.Downloader())
	chain := ethBackend.BlockChain()
	eng := &L2Engine{
		log:       log,
		node:      n,
		Eth:       ethBackend,
		l2Chain:   chain,
		EngineApi: engineApi,
	}
	// register the custom engine API, so we can serve engine requests while having more control
	// over sequencing of individual txs.
	n.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       eng.EngineApi,
			Authenticated: true,
		},
	})
	require.NoError(t, n.Start(), "failed to start L2 op-geth node")

	return eng
}

func newBackend(t e2eutils.TestingBase, genesis *core.Genesis, jwtPath string) (*node.Node, *geth.Ethereum, *engineApiBackend) {
	ethCfg := &ethconfig.Config{
		NetworkId:   bigs.Uint64Strict(genesis.Config.ChainID),
		Genesis:     genesis,
		StateScheme: rawdb.HashScheme,
		NoPruning:   true,
		// Record trie-key preimages when generating pre-fork state artifacts, so
		// the post-activation state can be enumerated and dumped. Off otherwise to
		// avoid the recording overhead in normal test runs.
		Preimages: os.Getenv("OP_E2E_GEN_PREFORK_STATE") != "",
	}
	nodeCfg := &node.Config{
		Name:        "l2-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		HTTPHost:    "127.0.0.1",
		HTTPPort:    0,
		AuthAddr:    "127.0.0.1",
		AuthPort:    0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		JWTSecret:   jwtPath,
	}
	n, err := node.New(nodeCfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = n.Close()
	})
	backend, err := geth.New(n, ethCfg)
	require.NoError(t, err)
	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	chain := backend.BlockChain()
	db := backend.ChainDb()
	apiBackend := &engineApiBackend{
		BlockChain: chain,
		db:         db,
		genesis:    genesis,
	}
	return n, backend, apiBackend
}

type engineApiBackend struct {
	*core.BlockChain
	db      ethdb.Database
	genesis *core.Genesis
}

func (e *engineApiBackend) Database() ethdb.Database {
	return e.db
}

func (e *engineApiBackend) Genesis() *core.Genesis {
	return e.genesis
}

func (s *L2Engine) L2Chain() *core.BlockChain {
	return s.l2Chain
}

func (s *L2Engine) HTTPEndpoint() string {
	return s.node.HTTPEndpoint()
}

func (s *L2Engine) EthClient() *ethclient.Client {
	cl := s.node.Attach()
	return ethclient.NewClient(cl)
}

func (s *L2Engine) GethClient() *gethclient.Client {
	cl := s.node.Attach()
	return gethclient.New(cl)
}

func (e *L2Engine) RPCClient() client.RPC {
	return client.NewBaseRPCClient(e.node.Attach())
}

func (e *L2Engine) EngineClient(t helpers.Testing, cfg *rollup.Config) *sources.EngineClient {
	l2Cl, err := sources.NewEngineClient(e.RPCClient(), e.log, nil, sources.EngineClientDefaultConfig(cfg))
	require.NoError(t, err)
	return l2Cl
}

// ActL2IncludeTxIgnoreForcedEmpty includes the next transaction from the given address in the block that is being built,
// skipping the usual check for e.EngineApi.ForcedEmpty()
func (e *L2Engine) ActL2IncludeTxIgnoreForcedEmpty(from common.Address) helpers.Action {
	return func(t helpers.Testing) {
		if e.EngineApi.ForcedEmpty() {
			e.log.Info("Ignoring e.L2ForceEmpty=true")
		}

		require.NoError(t, e.Eth.TxPool().Sync(), "must sync tx-pool to get accurate pending txs")
		tx := helpers.FirstValidTx(t, from, e.EngineApi.PendingIndices, e.Eth.TxPool().ContentFrom, e.EthClient().NonceAt)
		prevState := e.EngineApi.ForcedEmpty()
		e.EngineApi.SetForceEmpty(false) // ensure the engine API can include it
		_, err := e.EngineApi.IncludeTx(tx, from)
		e.EngineApi.SetForceEmpty(prevState)
		if errors.Is(err, engineapi.ErrNotBuildingBlock) {
			t.InvalidAction(err.Error())
		} else if errors.Is(err, engineapi.ErrUsesTooMuchGas) {
			t.InvalidAction("included tx uses too much gas: %v", err)
		} else if err != nil {
			require.NoError(t, err, "include tx")
		}
	}
}

// ActL2IncludeTx includes the next transaction from the given address in the block that is being built
func (e *L2Engine) ActL2IncludeTx(from common.Address) helpers.Action {
	return func(t helpers.Testing) {

		if e.EngineApi.ForcedEmpty() {
			e.log.Info("Skipping including a transaction because e.L2ForceEmpty is true")
			return
		}

		require.NoError(t, e.Eth.TxPool().Sync(), "must sync tx-pool to get accurate pending txs")
		tx := helpers.FirstValidTx(t, from, e.EngineApi.PendingIndices, e.Eth.TxPool().ContentFrom, e.EthClient().NonceAt)
		_, err := e.EngineApi.IncludeTx(tx, from)
		if errors.Is(err, engineapi.ErrNotBuildingBlock) {
			t.InvalidAction(err.Error())
		} else if errors.Is(err, engineapi.ErrUsesTooMuchGas) {
			t.InvalidAction("included tx uses too much gas: %v", err)
		} else if err != nil {
			require.NoError(t, err, "include tx")
		}
	}
}
