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
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// lagoonOffset is when interop activates, after the chain's genesis.
const lagoonOffset = 4

// peerChain is the dependency set's second chain, which the test does not run.
const peerChain = 902_999

// logDirEnv names a directory to keep each test run's process logs and files in, in a directory
// named after the test, after the test. Without it they go to the test's temporary directory.
const logDirEnv = "KONA_AUTHORITY_TEST_LOG_DIR"

// system is one interop chain whose blocks Go sequencers build and Go batchers post to the action
// L1 miner, and which a kona-node with its op-reth derives from that L1 while following lokahi for
// its safe and finalized heads and its denied blocks.
//
// The test drives L1 and the batches by hand, so it chooses what each L1 block carries and can
// reorg L1. The processes poll the miner's RPC and the fake beacon like any L1.
type system struct {
	t   helpers.Testing
	log log.Logger
	dir string

	dp     *e2eutils.DeployParams
	sd     *e2eutils.SetupData
	depSet depset.DependencySet
	miner  *helpers.L1Miner
	beacon *fakebeacon.FakeBeacon
	l1     *sources.L1Client
	// producer builds the chain the kona-node derives.
	producer *producer

	jwtPath   string
	rethRPC   string
	rethAuth  string
	konaRPC   string
	lokahiRPC string

	reth   *ethclient.Client
	kona   *rpc.Client
	lokahi *rpc.Client
}

func newSystem(t helpers.Testing) *system {
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
	// Interop activates after genesis, so that its activation block deploys the interop
	// predeploys, which a genesis does not.
	dp.DeployConfig.ActivateForkAtOffset(forks.Lagoon, lagoonOffset)
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	// The activation block activates the interop contracts only for a dependency set of more than
	// one chain, so the set holds a peer chain the test does not run, as the proofs action tests
	// do (DependencySetPeers). No message of the test refers to it.
	chainID := eth.ChainIDFromBig(sd.RollupCfg.L2ChainID)
	depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		chainID:                          {},
		eth.ChainIDFromUInt64(peerChain): {},
	})
	require.NoError(t, err)

	miner := helpers.NewL1Miner(t, logger.New("role", "l1-miner"), sd.L1Cfg)
	miner.ActL1SetFeeRecipient(common.Address{0xCA, 0xFE, 0xBA, 0xBE})
	beacon := fakebeacon.NewBeacon(logger.New("role", "beacon"), miner.BlobStore(), sd.L1Cfg.Timestamp, tp.L1BlockTime)
	require.NoError(t, beacon.Start("127.0.0.1:0"))
	t.Cleanup(func() { _ = beacon.Close() })

	l1Cl, err := sources.NewL1Client(miner.RPCClient(), logger, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false, sources.RPCKindStandard))
	require.NoError(t, err)

	s := &system{
		t: t, log: logger, dir: dir,
		dp: dp, sd: sd, depSet: depSet,
		miner: miner, beacon: beacon, l1: l1Cl,
	}
	s.producer = s.newProducer("producer")
	s.startOpReth()
	lokahiProxy := tcpproxy.New(logger.New("proxy", "lokahi"))
	require.NoError(t, lokahiProxy.Start())
	t.Cleanup(func() { _ = lokahiProxy.Close() })
	s.lokahiRPC = "http://" + lokahiProxy.Addr()
	s.startKonaNode()
	lokahiProxy.SetUpstream(strings.TrimPrefix(s.startLokahi(), "http://"))
	s.lokahi = s.dial(s.lokahiRPC)
	return s
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

// startOpReth starts the op-reth the kona-node drives, from the chain's genesis.
func (s *system) startOpReth() {
	bin := resolveBinary(s.t, s.log, "rust", "op-reth", "op-reth")
	genesis := s.writeJSON("l2-genesis.json", s.sd.L2Cfg)
	jwt := filepath.Join(s.dir, "jwt.hex")
	secret := [32]byte{123}
	require.NoError(s.t, os.WriteFile(jwt, []byte(hexutil.Encode(secret[:])), 0o600))
	dataDir := filepath.Join(s.dir, "op-reth")
	run(s.t, bin, "init", "--datadir="+dataDir, "--chain="+genesis)
	_, fields := startProcess(s.t, s.log, s.dir, "op-reth", bin, []string{
		"node",
		"--addr=127.0.0.1",
		"--authrpc.addr=127.0.0.1",
		"--authrpc.jwtsecret=" + jwt,
		"--authrpc.port=0",
		"--chain=" + genesis,
		"--color=never",
		"--rollup.block-time=" + strconv.FormatUint(s.sd.RollupCfg.BlockTime, 10),
		"--datadir=" + dataDir,
		"--disable-discovery",
		"--http",
		"--http.api=admin,debug,eth,net,txpool,web3,rpc",
		"--http.addr=127.0.0.1",
		"--http.port=0",
		"--ipcdisable",
		"--log.file.directory=" + filepath.Join(s.dir, "op-reth-logs"),
		"--log.stdout.format=json",
		"--nat=none",
		"--port=0",
		"--with-unused-ports",
	}, nil, "RPC HTTP server started", "RPC auth server started")
	s.rethRPC = "http://" + fields["RPC HTTP server started"].FieldValue("url").(string)
	s.rethAuth = "http://" + fields["RPC auth server started"].FieldValue("url").(string)
	s.reth = ethclient.NewClient(s.dial(s.rethRPC))
	s.jwtPath = jwt
}

// startKonaNode starts the kona-node that derives the chain from the action L1 into op-reth and
// follows lokahi.
func (s *system) startKonaNode() {
	bin := resolveBinary(s.t, s.log, "rust/kona", "kona-node", "kona-node")
	_, fields := startProcess(s.t, s.log, s.dir, "kona-node", bin, []string{"node"}, []string{
		"KONA_NODE_L1_ETH_RPC=" + s.miner.HTTPEndpoint(),
		"KONA_NODE_L1_BEACON=" + s.beacon.BeaconAddr(),
		"KONA_NODE_L2_ENGINE_RPC=" + s.rethAuth,
		"KONA_NODE_L2_ENGINE_AUTH=" + s.jwtPath,
		"KONA_NODE_ROLLUP_CONFIG=" + s.writeJSON("rollup.json", s.sd.RollupCfg),
		"KONA_NODE_L1_CHAIN_CONFIG=" + s.writeJSON("l1-chain-config.json", s.sd.L1Cfg.Config),
		"KONA_NODE_INTEROP_DEPENDENCY_SET=" + s.writeJSON("depset.json", s.depSet),
		"KONA_NODE_INTEROP_AUTHORITY_RPC=" + s.lokahiRPC,
		"KONA_NODE_SAFEDB_PATH=" + filepath.Join(s.dir, "safedb"),
		"KONA_NODE_P2P_PRIV_PATH=" + filepath.Join(s.dir, "p2pkey.txt"),
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
	s.konaRPC = "http://" + fields["RPC server bound to address"].FieldValue("addr").(string)
	s.kona = s.dial(s.konaRPC)
}

// startLokahi starts lokahi over the chain, reading it from the kona-node and its op-reth, and
// returns its RPC address.
func (s *system) startLokahi() string {
	bin := resolveBinary(s.t, s.log, "rust", "lokahi", "lokahi")
	type chainFile struct {
		RollupConfig string `json:"rollup_config"`
		CLRPC        string `json:"cl_rpc"`
		ELRPC        string `json:"el_rpc"`
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
		Chains: []chainFile{{
			RollupConfig: filepath.Join(s.dir, "rollup.json"),
			CLRPC:        s.konaRPC,
			ELRPC:        s.rethRPC,
		}},
	})
	_, fields := startProcess(s.t, s.log, s.dir, "lokahi", bin,
		[]string{"--config", config, "--datadir", filepath.Join(s.dir, "lokahi")},
		[]string{"RUST_LOG=info"}, "RPC server bound to address")
	return "http://" + fields["RPC server bound to address"].FieldValue("addr").(string)
}

// rollupClient is the kona-node's rollup RPC.
func (s *system) rollupClient() *sources.RollupClient {
	return sources.NewRollupClient(client.NewBaseRPCClient(s.kona))
}

// syncStatus is the kona-node's sync status.
func (s *system) syncStatus() *eth.SyncStatus {
	status, err := s.rollupClient().SyncStatus(s.t.Ctx())
	require.NoError(s.t, err, "kona-node sync status")
	return status
}

// producer builds blocks of the chain with a Go sequencer on an engine of its own, and posts them
// to L1 with its batcher in calldata. Two producers that derived the same chain from L1 can build
// different blocks at the same height on it.
type producer struct {
	s         *system
	engine    *helpers.L2Engine
	sequencer *helpers.L2Sequencer
	batcher   *helpers.L2Batcher
	l2        *sources.L2Client
}

func (s *system) newProducer(name string) *producer {
	engine := helpers.NewL2Engine(s.t, s.log.New("role", name+"-engine"), s.sd.L2Cfg)
	engineCl, err := sources.NewEngineClient(engine.RPCClient(), s.log, nil, sources.EngineClientDefaultConfig(s.sd.RollupCfg))
	require.NoError(s.t, err)
	sequencer := helpers.NewL2Sequencer(s.t, s.log.New("role", name), s.l1, s.miner.BlobStore(), altda.Disabled,
		engineCl, s.sd.RollupCfg, s.sd.L1Cfg.Config, s.depSet, 0)
	sequencer.ActL2PipelineFull(s.t)
	batcher := helpers.NewL2Batcher(s.log.New("role", name+"-batcher"), s.sd.RollupCfg, &helpers.BatcherCfg{
		MinL1TxSize:          0,
		MaxL1TxSize:          128_000,
		DataAvailabilityType: batcherFlags.CalldataType,
		BatcherKey:           s.dp.Secrets.Batcher,
	}, sequencer.RollupClient(), s.miner.EthClient(), engine.EngineClient(s.t, s.sd.RollupCfg))
	l2, err := sources.NewL2Client(engine.RPCClient(), s.log, nil, sources.L2ClientDefaultConfig(s.sd.RollupCfg, false))
	require.NoError(s.t, err)
	return &producer{s: s, engine: engine, sequencer: sequencer, batcher: batcher, l2: l2}
}

// derive has the producer derive the chain from L1 up to the L1 head.
func (p *producer) derive() eth.L2BlockRef {
	p.sequencer.ActL1HeadSignal(p.s.t)
	p.sequencer.ActL2PipelineFull(p.s.t)
	return p.sequencer.SyncStatus().SafeL2
}

// batch posts the producer's unsafe blocks past its safe head to L1 in one transaction, which it
// returns. The transaction takes the batcher's nonce on the L1 chain as it is, not counting a
// transaction of another producer the L1 pool may still hold, and outbids that one.
func (p *producer) batch() *types.Transaction {
	nonce, err := p.s.miner.EthClient().NonceAt(p.s.t.Ctx(), p.batcher.BatcherAddr, nil)
	require.NoError(p.s.t, err)
	p.batcher.ActBufferAll(p.s.t)
	p.batcher.ActL2ChannelClose(p.s.t)
	p.batcher.ActL2BatchSubmit(p.s.t, func(tx *types.DynamicFeeTx) {
		tx.Nonce = nonce
		tx.GasTipCap = new(big.Int).Mul(tx.GasTipCap, big.NewInt(3))
		tx.GasFeeCap = new(big.Int).Mul(tx.GasFeeCap, big.NewInt(3))
	})
	return p.batcher.LastSubmitted
}

// payload is the producer's block number as an execution payload.
func (p *producer) payload(number uint64) *eth.ExecutionPayloadEnvelope {
	payload, err := p.l2.PayloadByNumber(p.s.t.Ctx(), number)
	require.NoError(p.s.t, err, "payload of block %d", number)
	return payload
}

// mine mines an L1 block that includes txs.
func (s *system) mine(txs ...*types.Transaction) *types.Block {
	s.miner.ActL1StartBlock(12)(s.t)
	for _, tx := range txs {
		s.miner.ActL1IncludeTxByHash(tx.Hash())(s.t)
	}
	return s.miner.ActL1EndBlock(s.t)
}

// postUnsafe hands the kona-node a block, as gossip would.
func (s *system) postUnsafe(payload *eth.ExecutionPayloadEnvelope) {
	require.NoError(s.t, s.rollupClient().PostUnsafePayload(s.t.Ctx(), payload),
		"post block %d", payload.ExecutionPayload.BlockNumber)
}

// awaitLocalSafe waits until the kona-node's local-safe head is at least number.
func (s *system) awaitLocalSafe(number uint64) *eth.SyncStatus {
	var status *eth.SyncStatus
	require.Eventually(s.t, func() bool {
		status = s.syncStatus()
		return status.LocalSafeL2.Number >= number
	}, 2*time.Minute, 500*time.Millisecond, "kona-node must derive to block %d", number)
	return status
}

// build has the producer build its next block with tx in it.
func (p *producer) build(tx *types.Transaction) eth.L2BlockRef {
	p.sequencer.ActL2StartBlock(p.s.t)
	require.NoError(p.s.t, p.engine.IncludeTxErr(p.s.t, tx, common.Address{}), "include tx %s", tx.Hash())
	return p.sequencer.ActL2EndBlock(p.s.t)
}

// signL2 signs a transaction of the chain from key, at its next nonce on the producer's chain.
func (s *system) signL2(p *producer, key *ecdsa.PrivateKey, to common.Address, gas uint64, data []byte, accessList types.AccessList) *types.Transaction {
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := p.engine.EthClient().NonceAt(s.t.Ctx(), from, nil)
	require.NoError(s.t, err)
	head, err := p.engine.EthClient().HeaderByNumber(s.t.Ctx(), nil)
	require.NoError(s.t, err)
	tip := big.NewInt(params.GWei)
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(s.sd.RollupCfg.L2ChainID), &types.DynamicFeeTx{
		ChainID:    s.sd.RollupCfg.L2ChainID,
		Nonce:      nonce,
		GasTipCap:  tip,
		GasFeeCap:  new(big.Int).Add(tip, new(big.Int).Mul(head.BaseFee, big.NewInt(2))),
		Gas:        gas,
		To:         &to,
		Value:      big.NewInt(0),
		Data:       data,
		AccessList: accessList,
	})
	require.NoError(s.t, err)
	return tx
}

// invalidExecutingMessageTx is a transaction that executes a message of the chain's first block
// after interop's activation that no log of it initiates.
func (s *system) invalidExecutingMessageTx(p *producer) *types.Transaction {
	initiatingBlock := s.sd.RollupCfg.Genesis.L2.Number + lagoonOffset/s.sd.RollupCfg.BlockTime + 1
	trigger := &txintent.ExecTrigger{
		Executor: predeploys.CrossL2InboxAddr,
		Msg: messages.Message{
			Identifier: messages.Identifier{
				Origin:      common.Address{0xde, 0xad},
				BlockNumber: initiatingBlock,
				LogIndex:    0,
				Timestamp:   s.sd.RollupCfg.TimestampForBlock(initiatingBlock),
				ChainID:     eth.ChainIDFromBig(s.sd.RollupCfg.L2ChainID),
			},
			PayloadHash: crypto.Keccak256Hash([]byte("no such message")),
		},
	}
	data, err := trigger.EncodeInput()
	require.NoError(s.t, err)
	accessList, err := trigger.AccessList()
	require.NoError(s.t, err)
	return s.signL2(p, s.dp.Secrets.Alice, predeploys.CrossL2InboxAddr, 200_000, data, accessList)
}

// transferTx is a transaction that sends no ether from Bob to Alice.
func (s *system) transferTx(p *producer) *types.Transaction {
	return s.signL2(p, s.dp.Secrets.Bob, s.dp.Addresses.Alice, params.TxGas, nil, nil)
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
func (s *system) lokahiStatus() lokahiStatus {
	var status lokahiStatus
	chainID := bigs.Uint64Strict(s.sd.RollupCfg.L2ChainID)
	require.NoError(s.t, s.lokahi.CallContext(s.t.Ctx(), &status, "interop_chainStatus", chainID, 0))
	return status
}

// appliedEpoch is the lokahi epoch the kona-node derives under, if any.
func (s *system) appliedEpoch() *common.Hash {
	var applied *common.Hash
	require.NoError(s.t, s.kona.CallContext(s.t.Ctx(), &applied, "interop_appliedEpoch"))
	return applied
}

// nodeBlockAt is the block at number on the kona-node's op-reth.
func (s *system) nodeBlockAt(number uint64) eth.BlockID {
	header, err := s.reth.HeaderByNumber(s.t.Ctx(), new(big.Int).SetUint64(number))
	require.NoError(s.t, err, "block %d on the kona-node's op-reth", number)
	return eth.BlockID{Hash: header.Hash(), Number: number}
}
