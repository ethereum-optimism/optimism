package authority_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	opgenesis "github.com/ethereum-optimism/optimism/op-core/genesis"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// lagoonOffset is when interop activates, after the chains' genesis.
const lagoonOffset = 4

// peerChain is the dependency set's second chain of a one-chain system, which the test does not
// run.
const peerChain = 902_999

// logDirEnv names a directory to keep each test run's process logs and files in, in a directory
// named after the test, after the test. Without it they go to the test's temporary directory.
const logDirEnv = "KONA_AUTHORITY_TEST_LOG_DIR"

// system is interop chains on one action L1 miner. Go sequencers build each chain's blocks and Go
// batchers post them to L1, and per chain a kona-node with its op-reth derives the chain from that
// L1 while following one lokahi for its safe and finalized heads and its denied blocks.
//
// The test drives L1 and the batches by hand, so it chooses what each L1 block carries and can
// reorg L1. The processes poll the miner's RPC and the fake beacon like any L1.
type system struct {
	t   helpers.Testing
	log log.Logger
	dir string

	dp     *e2eutils.DeployParams
	depSet depset.DependencySet
	miner  *helpers.L1Miner
	beacon *fakebeacon.FakeBeacon
	chains []*chain

	jwtPath   string
	lokahiRPC string
	lokahi    *rpc.Client
}

// chain is one L2 chain of the system: its configuration, the producer whose chain its kona-node
// derives, and the kona-node with its op-reth.
type chain struct {
	s          *system
	name       string
	rollupCfg  *rollup.Config
	l2Genesis  *opgenesis.Genesis
	batcherKey *ecdsa.PrivateKey
	// producer builds the chain the kona-node derives.
	producer *producer

	// l1 is the kona-node's view of the action L1.
	l1 *rpcProxy
	// authority is the kona-node's connection to lokahi.
	authority *rpcProxy
	rethRPC   string
	rethAuth  string
	konaRPC   string
	reth      *ethclient.Client
	kona      *rpc.Client
}

// newSystem starts a system of chains interop chains, which activate interop together after
// genesis, so that the activation block deploys the interop predeploys, which a genesis does not.
func newSystem(t helpers.Testing, chains int) *system {
	logger := testlog.Logger(t, log.LevelInfo)
	dir := t.TempDir()
	if logDir := os.Getenv(logDirEnv); logDir != "" {
		require.NoError(t, os.MkdirAll(logDir, 0o755))
		var err error
		dir, err = os.MkdirTemp(logDir, t.Name()+"-")
		require.NoError(t, err)
	}

	tp := helpers.DefaultRollupTestParams()
	dp := e2eutils.MakeDeployParams(t, tp)
	dp.DeployConfig.ActivateForkAtOffset(forks.Lagoon, lagoonOffset)
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)

	miner := helpers.NewL1Miner(t, logger.New("role", "l1-miner"), sd.L1Cfg)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	beacon := fakebeacon.NewBeacon(logger.New("role", "beacon"), miner.BlobStore(), sd.L1Cfg.Timestamp, tp.L1BlockTime)
	require.NoError(t, beacon.Start("127.0.0.1:0"))
	t.Cleanup(func() { _ = beacon.Close() })

	s := &system{t: t, log: logger, dir: dir, dp: dp, miner: miner, beacon: beacon}
	s.chains = append(s.chains, &chain{
		s: s, name: "a", rollupCfg: sd.RollupCfg, l2Genesis: sd.L2Cfg, batcherKey: dp.Secrets.Batcher,
	})
	if chains > 1 {
		s.chains = append(s.chains, s.secondChain(sd))
	}
	require.Len(t, s.chains, chains, "the system runs one or two chains")

	// The activation block activates the interop contracts only for a dependency set of more than
	// one chain, so a one-chain system's set holds a peer chain the test does not run, as the
	// proofs action tests do (DependencySetPeers). No message of the test refers to it.
	deps := map[eth.ChainID]*depset.StaticConfigDependency{}
	for _, c := range s.chains {
		deps[c.chainID()] = &depset.StaticConfigDependency{}
	}
	if chains == 1 {
		deps[eth.ChainIDFromUInt64(peerChain)] = &depset.StaticConfigDependency{}
	}
	var err error
	s.depSet, err = depset.NewStaticConfigDependencySet(deps)
	require.NoError(t, err)

	jwt := filepath.Join(dir, "jwt.hex")
	secret := [32]byte{123}
	require.NoError(t, os.WriteFile(jwt, []byte(hexutil.Encode(secret[:])), 0o600))
	s.jwtPath = jwt
	s.writeJSON("depset.json", s.depSet)
	s.writeJSON("l1-chain-config.json", sd.L1Cfg.Config)

	lokahiProxy := tcpproxy.New(logger.New("proxy", "lokahi"))
	require.NoError(t, lokahiProxy.Start())
	t.Cleanup(func() { _ = lokahiProxy.Close() })
	s.lokahiRPC = "http://" + lokahiProxy.Addr()
	for _, c := range s.chains {
		c.producer = c.newProducer("producer")
		c.startOpReth()
		c.startKonaNode()
	}
	lokahiProxy.SetUpstream(strings.TrimPrefix(s.startLokahi(), "http://"))
	s.lokahi = s.dial(s.lokahiRPC)
	return s
}

// secondChain is a second chain of the same deployment on the same L1, with a chain ID, genesis,
// batch inbox and batcher of its own. It reads no deposits and system config updates from L1:
// the deployment's L1 contracts belong to the first chain.
func (s *system) secondChain(first *e2eutils.SetupData) *chain {
	cfg := s.dp.DeployConfig.Copy()
	cfg.L2ChainID = bigs.Uint64Strict(first.RollupCfg.L2ChainID) + 1
	cfg.BatchInboxAddress = common.Address{0xff, 0x02}
	cfg.BatchSenderAddress = s.dp.Addresses.Proposer
	l1Block := first.L1Cfg.ToBlock()
	allocs := config.L2Allocs(s.dp.AllocType, e2eutils.GetL2AllocsMode(cfg, l1Block.Time())).Copy()
	// The allocs were generated for the first chain, and Permit2 carries that chain's ID in an
	// immutable. The test never calls it.
	delete(allocs.Accounts, predeploys.Permit2Addr)
	l2Genesis, err := genesis.BuildL2Genesis(cfg, allocs, eth.BlockRefFromHeader(l1Block.Header()))
	require.NoError(s.t, err, "second chain's L2 genesis")
	for _, addr := range s.dp.Addresses.All() {
		l2Genesis.Alloc[addr] = types.Account{Balance: e2eutils.Ether(1e12)}
	}
	require.Equal(s.t, first.RollupCfg.Genesis.L2Time, l2Genesis.Timestamp, "both chains start together")
	rollupCfg := *first.RollupCfg
	rollupCfg.L2ChainID = new(big.Int).SetUint64(cfg.L2ChainID)
	rollupCfg.Genesis.L2.Hash = l2Genesis.ToBlock().Hash()
	rollupCfg.Genesis.SystemConfig.BatcherAddr = cfg.BatchSenderAddress
	rollupCfg.BatchInboxAddress = cfg.BatchInboxAddress
	rollupCfg.DepositContractAddress = common.Address{0xff, 0x03}
	rollupCfg.L1SystemConfigAddress = common.Address{0xff, 0x04}
	require.NoError(s.t, rollupCfg.Check())
	return &chain{s: s, name: "b", rollupCfg: &rollupCfg, l2Genesis: l2Genesis, batcherKey: s.dp.Secrets.Proposer}
}

func (s *system) writeJSON(name string, v any) string {
	data, err := json.Marshal(v)
	require.NoError(s.t, err, "marshal %s", name)
	path := filepath.Join(s.dir, name)
	require.NoError(s.t, os.WriteFile(path, data, 0o644))
	return path
}

func (s *system) dial(url string) *rpc.Client {
	ctx, cancel := context.WithTimeout(s.t.Ctx(), 30*time.Second)
	defer cancel()
	cl, err := rpc.DialContext(ctx, url)
	require.NoError(s.t, err, "dial %s", url)
	s.t.Cleanup(cl.Close)
	return cl
}

func (c *chain) chainID() eth.ChainID {
	return eth.ChainIDFromBig(c.rollupCfg.L2ChainID)
}

// path is where the chain's file name lives.
func (c *chain) path(name string) string {
	return filepath.Join(c.s.dir, c.name+"-"+name)
}

// startOpReth starts the op-reth the chain's kona-node drives, from the chain's genesis.
func (c *chain) startOpReth() {
	s := c.s
	bin := resolveBinary(s.t, s.log, "rust", "op-reth", "op-reth")
	genesisPath := s.writeJSON(c.name+"-l2-genesis.json", c.l2Genesis)
	dataDir := c.path("op-reth")
	run(s.t, bin, "init", "--datadir="+dataDir, "--chain="+genesisPath)
	_, fields := startProcess(s.t, s.log, s.dir, c.name+"-op-reth", bin, []string{
		"node",
		"--addr=127.0.0.1",
		"--authrpc.addr=127.0.0.1",
		"--authrpc.jwtsecret=" + s.jwtPath,
		"--authrpc.port=0",
		"--chain=" + genesisPath,
		"--color=never",
		"--rollup.block-time=" + strconv.FormatUint(c.rollupCfg.BlockTime, 10),
		"--datadir=" + dataDir,
		"--disable-discovery",
		"--http",
		"--http.api=admin,debug,eth,net,txpool,web3,rpc",
		"--http.addr=127.0.0.1",
		"--http.port=0",
		"--ipcdisable",
		"--log.file.directory=" + c.path("op-reth-logs"),
		"--log.stdout.format=json",
		"--nat=none",
		"--port=0",
		"--with-unused-ports",
	}, nil, "RPC HTTP server started", "RPC auth server started")
	c.rethRPC = "http://" + fields["RPC HTTP server started"].FieldValue("url").(string)
	c.rethAuth = "http://" + fields["RPC auth server started"].FieldValue("url").(string)
	c.reth = ethclient.NewClient(s.dial(c.rethRPC))
}

// startKonaNode starts the kona-node that derives the chain from the action L1 into its op-reth
// and follows lokahi.
func (c *chain) startKonaNode() {
	s := c.s
	c.l1 = newRPCProxy(s.t, s.miner.HTTPEndpoint())
	c.authority = newRPCProxy(s.t, s.lokahiRPC)
	bin := resolveBinary(s.t, s.log, "rust/kona", "kona-node", "kona-node")
	_, fields := startProcess(s.t, s.log, s.dir, c.name+"-kona-node", bin, []string{"node"}, []string{
		"KONA_NODE_L1_ETH_RPC=" + c.l1.url(),
		"KONA_NODE_L1_BEACON=" + s.beacon.BeaconAddr(),
		"KONA_NODE_L2_ENGINE_RPC=" + c.rethAuth,
		"KONA_NODE_L2_ENGINE_AUTH=" + s.jwtPath,
		"KONA_NODE_ROLLUP_CONFIG=" + s.writeJSON(c.name+"-rollup.json", c.rollupCfg),
		"KONA_NODE_L1_CHAIN_CONFIG=" + filepath.Join(s.dir, "l1-chain-config.json"),
		"KONA_NODE_INTEROP_DEPENDENCY_SET=" + filepath.Join(s.dir, "depset.json"),
		"KONA_NODE_INTEROP_AUTHORITY_RPC=" + c.authority.url(),
		"KONA_NODE_SAFEDB_PATH=" + c.path("safedb"),
		"KONA_NODE_P2P_PRIV_PATH=" + c.path("p2pkey.txt"),
		"KONA_NODE_P2P_NO_DISCOVERY=true",
		"KONA_NODE_P2P_LISTEN_IP=127.0.0.1",
		"KONA_NODE_P2P_LISTEN_TCP_PORT=0",
		"KONA_NODE_P2P_LISTEN_UDP_PORT=0",
		"KONA_NODE_VERIFIER_L1_CONFS=0",
		"KONA_NODE_L1_EPOCH_POLL_INTERVAL=2",
		"KONA_NODE_MODE=Validator",
		"KONA_NODE_RPC_ADDR=127.0.0.1",
		"KONA_NODE_RPC_PORT=0",
		"KONA_LOG_LEVEL=3",
		"KONA_LOG_STDOUT_FORMAT=json",
	}, "RPC server bound to address")
	c.konaRPC = "http://" + fields["RPC server bound to address"].FieldValue("addr").(string)
	c.kona = s.dial(c.konaRPC)
}

// startLokahi starts lokahi over the chains, reading each from its kona-node and op-reth, and
// returns its RPC address.
func (s *system) startLokahi() string {
	bin := resolveBinary(s.t, s.log, "rust", "lokahi", "lokahi")
	type chainFile struct {
		RollupConfig string `json:"rollup_config"`
		CLRPC        string `json:"cl_rpc"`
		ELRPC        string `json:"el_rpc"`
	}
	var chains []chainFile
	for _, c := range s.chains {
		chains = append(chains, chainFile{RollupConfig: c.path("rollup.json"), CLRPC: c.konaRPC, ELRPC: c.rethRPC})
	}
	config := s.writeJSON("lokahi.json", struct {
		L1RPC         string      `json:"l1_rpc"`
		RPCAddr       string      `json:"rpc_addr"`
		DependencySet string      `json:"dependency_set"`
		Chains        []chainFile `json:"chains"`
	}{
		L1RPC:         s.miner.HTTPEndpoint(),
		RPCAddr:       "127.0.0.1:0",
		DependencySet: filepath.Join(s.dir, "depset.json"),
		Chains:        chains,
	})
	_, fields := startProcess(s.t, s.log, s.dir, "lokahi", bin,
		[]string{"--config", config, "--datadir", filepath.Join(s.dir, "lokahi")},
		[]string{"RUST_LOG=info"}, "RPC server bound to address")
	return "http://" + fields["RPC server bound to address"].FieldValue("addr").(string)
}

// rollupClient is the chain's kona-node's rollup RPC.
func (c *chain) rollupClient() *sources.RollupClient {
	return sources.NewRollupClient(client.NewBaseRPCClient(c.kona))
}

// syncStatus is the chain's kona-node's sync status.
func (c *chain) syncStatus() *eth.SyncStatus {
	status, err := c.rollupClient().SyncStatus(c.s.t.Ctx())
	require.NoError(c.s.t, err, "chain %s: kona-node sync status", c.name)
	return status
}

// producer builds blocks of a chain with a Go sequencer on an engine of its own, and posts them to
// L1 with its batcher in calldata. Two producers that derived the same chain from L1 can build
// different blocks at the same height on it.
type producer struct {
	c         *chain
	engine    *helpers.L2Engine
	sequencer *helpers.L2Sequencer
	batcher   *helpers.L2Batcher
	l2        *sources.L2Client
}

func (c *chain) newProducer(name string) *producer {
	s := c.s
	logger := s.log.New("role", c.name+"-"+name)
	engine := helpers.NewL2Engine(s.t, logger.New("component", "engine"), c.l2Genesis)
	engineCl, err := sources.NewEngineClient(engine.RPCClient(), s.log, nil, sources.EngineClientDefaultConfig(c.rollupCfg))
	require.NoError(s.t, err)
	l1, err := sources.NewL1Client(s.miner.RPCClient(), s.log, nil, sources.L1ClientDefaultConfig(c.rollupCfg, false, sources.RPCKindStandard))
	require.NoError(s.t, err)
	sequencer := helpers.NewL2Sequencer(s.t, logger, l1, s.miner.BlobStore(), altda.Disabled,
		engineCl, c.rollupCfg, s.miner.L1Chain().Config(), s.depSet, 0)
	sequencer.ActL2PipelineFull(s.t)
	batcher := helpers.NewL2Batcher(logger.New("component", "batcher"), c.rollupCfg, &helpers.BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		DataAvailabilityType: batcherFlags.CalldataType,
		BatcherKey:           c.batcherKey,
	}, sequencer.RollupClient(), s.miner.EthClient(), engine.EngineClient(s.t, c.rollupCfg))
	l2, err := sources.NewL2Client(engine.RPCClient(), s.log, nil, sources.L2ClientDefaultConfig(c.rollupCfg, false))
	require.NoError(s.t, err)
	return &producer{c: c, engine: engine, sequencer: sequencer, batcher: batcher, l2: l2}
}

// buildToL1Head has the producer build blocks up to the L1 head's time.
func (p *producer) buildToL1Head() eth.L2BlockRef {
	p.sequencer.ActL1HeadSignal(p.c.s.t)
	return p.sequencer.ActBuildToL1Head(p.c.s.t)
}

// derive has the producer derive the chain from L1 up to the L1 head.
func (p *producer) derive() eth.L2BlockRef {
	p.sequencer.ActL1HeadSignal(p.c.s.t)
	p.sequencer.ActL2PipelineFull(p.c.s.t)
	return p.sequencer.SyncStatus().SafeL2
}

// batch posts the producer's unsafe blocks past its safe head to L1 in one transaction, which it
// returns. The transaction takes the batcher's nonce on the L1 chain as it is, not counting a
// transaction of another producer the L1 pool may still hold, and outbids that one.
func (p *producer) batch() *types.Transaction {
	t := p.c.s.t
	nonce, err := p.c.s.miner.EthClient().NonceAt(t.Ctx(), p.batcher.BatcherAddr, nil)
	require.NoError(t, err)
	p.batcher.ActBufferAll(t)
	p.batcher.ActL2ChannelClose(t)
	p.batcher.ActL2BatchSubmit(t, func(tx *types.DynamicFeeTx) {
		tx.Nonce = nonce
		tx.GasTipCap = new(big.Int).Mul(tx.GasTipCap, big.NewInt(3))
		tx.GasFeeCap = new(big.Int).Mul(tx.GasFeeCap, big.NewInt(3))
	})
	return p.batcher.LastSubmitted
}

// payload is the producer's block number as an execution payload.
func (p *producer) payload(number uint64) *eth.ExecutionPayloadEnvelope {
	payload, err := p.l2.PayloadByNumber(p.c.s.t.Ctx(), number)
	require.NoError(p.c.s.t, err, "payload of block %d", number)
	return payload
}

// build has the producer build its next block with txs in it.
func (p *producer) build(txs ...*types.Transaction) eth.L2BlockRef {
	t := p.c.s.t
	p.sequencer.ActL2StartBlock(t)
	for _, tx := range txs {
		require.NoError(t, p.engine.IncludeTxErr(t, tx, common.Address{}), "include tx %s", tx.Hash())
	}
	return p.sequencer.ActL2EndBlock(t)
}

// receipt is the receipt of tx on the producer's chain, which must have succeeded.
func (p *producer) receipt(tx *types.Transaction) *types.Receipt {
	receipt, err := p.engine.EthClient().TransactionReceipt(p.c.s.t.Ctx(), tx.Hash())
	require.NoError(p.c.s.t, err, "receipt of %s", tx.Hash())
	require.Equal(p.c.s.t, types.ReceiptStatusSuccessful, receipt.Status, "tx %s must succeed", tx.Hash())
	return receipt
}

// mine mines an L1 block that includes txs.
func (s *system) mine(txs ...*types.Transaction) *types.Block {
	s.miner.ActL1StartBlock(12)(s.t)
	for _, tx := range txs {
		s.miner.ActL1IncludeTxByHash(tx.Hash())(s.t)
	}
	return s.miner.ActL1EndBlock(s.t)
}

// postUnsafe hands the chain's kona-node a block, as gossip would.
func (c *chain) postUnsafe(payload *eth.ExecutionPayloadEnvelope) {
	require.NoError(c.s.t, c.rollupClient().PostUnsafePayload(c.s.t.Ctx(), payload),
		"chain %s: post block %d", c.name, payload.ExecutionPayload.BlockNumber)
}

// awaitLocalSafe waits until the chain's kona-node's local-safe head is at least number.
func (c *chain) awaitLocalSafe(number uint64) *eth.SyncStatus {
	var status *eth.SyncStatus
	require.Eventually(c.s.t, func() bool {
		status = c.syncStatus()
		return status.LocalSafeL2.Number >= number
	}, 2*time.Minute, 500*time.Millisecond, "chain %s: kona-node must derive to block %d", c.name, number)
	return status
}

// signL2 signs a transaction of the chain from key, at its next nonce on the producer's chain.
func (p *producer) signL2(key *ecdsa.PrivateKey, to common.Address, value *big.Int, gas uint64, data []byte, accessList types.AccessList) *types.Transaction {
	t := p.c.s.t
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := p.engine.EthClient().NonceAt(t.Ctx(), from, nil)
	require.NoError(t, err)
	head, err := p.engine.EthClient().HeaderByNumber(t.Ctx(), nil)
	require.NoError(t, err)
	tip := big.NewInt(params.GWei)
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(p.c.rollupCfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:    p.c.rollupCfg.L2ChainID,
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  new(big.Int).Add(tip, new(big.Int).Mul(head.BaseFee, big.NewInt(2))),
		Gas:        gas,
		To:         &to,
		Value:      value,
		Data:       data,
		AccessList: accessList,
	})
	require.NoError(t, err)
	return tx
}

// executingMessageTx is a transaction that executes msg.
func (p *producer) executingMessageTx(msg messages.Message) *types.Transaction {
	trigger := &txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: msg}
	data, err := trigger.EncodeInput()
	require.NoError(p.c.s.t, err)
	accessList, err := trigger.AccessList()
	require.NoError(p.c.s.t, err)
	return p.signL2(p.c.s.dp.Secrets.Alice, predeploys.CrossL2InboxAddr, big.NewInt(0), 200_000, data, accessList)
}

// missingMessage is a message of the chain's first block after interop's activation that no log
// of it initiates.
func (c *chain) missingMessage() messages.Message {
	initiatingBlock := c.rollupCfg.Genesis.L2.Number + lagoonOffset/c.rollupCfg.BlockTime + 1
	return messages.Message{
		Identifier: messages.Identifier{
			Origin:      common.Address{0xde, 0xad},
			BlockNumber: initiatingBlock,
			LogIndex:    0,
			Timestamp:   c.rollupCfg.TimestampForBlock(initiatingBlock),
			ChainID:     c.chainID(),
		},
		PayloadHash: crypto.Keccak256Hash([]byte("no such message")),
	}
}

// initiatingTx is a transaction whose log can initiate a message: it wraps ether into WETH, which
// logs the deposit.
func (p *producer) initiatingTx() *types.Transaction {
	deposit := crypto.Keccak256([]byte("deposit()"))[:4]
	return p.signL2(p.c.s.dp.Secrets.Bob, predeploys.WETHAddr, big.NewInt(1), 100_000, deposit, nil)
}

// messageOf is the message the only log of receipt initiates, in the block at time timestamp.
func (c *chain) messageOf(receipt *types.Receipt, timestamp uint64) messages.Message {
	require.Len(c.s.t, receipt.Logs, 1, "the initiating tx logs one message")
	l := receipt.Logs[0]
	return messages.Message{
		Identifier: messages.Identifier{
			Origin:      l.Address,
			BlockNumber: l.BlockNumber,
			LogIndex:    uint32(l.Index),
			Timestamp:   timestamp,
			ChainID:     c.chainID(),
		},
		PayloadHash: crypto.Keccak256Hash(messages.LogToMessagePayload(l)),
	}
}

// transferTx is a transaction that sends no ether from Bob to Alice.
func (p *producer) transferTx() *types.Transaction {
	return p.signL2(p.c.s.dp.Secrets.Bob, p.c.s.dp.Addresses.Alice, big.NewInt(0), params.TxGas, nil, nil)
}

// verifierHead is a head in lokahi's answer to interop_chainStatus.
type verifierHead struct {
	Block     eth.BlockID `json:"block"`
	Timestamp uint64      `json:"timestamp"`
	Source    string      `json:"source"`
	L1        eth.BlockID `json:"l1"`
}

// lokahiStatus is lokahi's answer to interop_chainStatus: its heads are nil until it caught up
// with its sources.
type lokahiStatus struct {
	Epoch     common.Hash   `json:"epoch"`
	Verified  *verifierHead `json:"verified"`
	Finalized *verifierHead `json:"finalized"`
	Denied    []eth.BlockID `json:"denied"`
}

// lokahiStatus asks lokahi for the chain's status, with every denied block.
func (c *chain) lokahiStatus() lokahiStatus {
	var status lokahiStatus
	chainID := bigs.Uint64Strict(c.rollupCfg.L2ChainID)
	require.NoError(c.s.t, c.s.lokahi.CallContext(c.s.t.Ctx(), &status, "interop_chainStatus", chainID, 0))
	return status
}

// appliedEpoch is the lokahi epoch the chain's kona-node derives under, if any.
func (c *chain) appliedEpoch() *common.Hash {
	var applied *common.Hash
	require.NoError(c.s.t, c.kona.CallContext(c.s.t.Ctx(), &applied, "interop_appliedEpoch"))
	return applied
}

// nodeBlockAt is the block at number on the chain's kona-node's op-reth.
func (c *chain) nodeBlockAt(number uint64) eth.BlockID {
	header, err := c.reth.HeaderByNumber(c.s.t.Ctx(), new(big.Int).SetUint64(number))
	require.NoError(c.s.t, err, "chain %s: block %d on the kona-node's op-reth", c.name, number)
	return eth.BlockID{Hash: header.Hash(), Number: number}
}

// konaLog is the path of the chain's kona-node log.
func (c *chain) konaLog() string {
	return filepath.Join(c.s.dir, c.name+"-kona-node.log")
}

// logSize is the size of the chain's kona-node log, to read what it logs after.
func (c *chain) logSize() int64 {
	info, err := os.Stat(c.konaLog())
	require.NoError(c.s.t, err)
	return info.Size()
}

// konaLogEntry is a line of a kona-node's JSON log.
type konaLogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Fields    map[string]any `json:"fields"`
}

func (e konaLogEntry) message() string {
	msg, _ := e.Fields["message"].(string)
	return msg
}

// logSince returns the chain's kona-node log entries after offset whose message contains any of
// words.
func (c *chain) logSince(offset int64, words ...string) []konaLogEntry {
	data, err := os.ReadFile(c.konaLog())
	require.NoError(c.s.t, err)
	var entries []konaLogEntry
	for _, line := range strings.Split(string(data[offset:]), "\n") {
		var e konaLogEntry
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		for _, word := range words {
			if strings.Contains(e.message(), word) {
				entries = append(entries, e)
				break
			}
		}
	}
	return entries
}
