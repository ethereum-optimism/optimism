package consensus

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// --- latency-injecting stream layer ---

// tcpStreamLayer is a minimal raft.StreamLayer implementation over a TCP listener.
// hashicorp/raft exports TCPStreamLayer, but its constructor and fields are
// unexported, so we provide our own to inject a custom listener.
type tcpStreamLayer struct {
	listener  *net.TCPListener
	advertise net.Addr
}

func (t *tcpStreamLayer) Accept() (net.Conn, error) { return t.listener.Accept() }
func (t *tcpStreamLayer) Close() error              { return t.listener.Close() }
func (t *tcpStreamLayer) Addr() net.Addr {
	if t.advertise != nil {
		return t.advertise
	}
	return t.listener.Addr()
}

func (t *tcpStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("tcp", string(address), timeout)
}

// latencyStreamLayer wraps a StreamLayer and injects write delay on connections.
type latencyStreamLayer struct {
	raft.StreamLayer
	delay time.Duration
}

func (l *latencyStreamLayer) Accept() (net.Conn, error) {
	conn, err := l.StreamLayer.Accept()
	if err != nil {
		return nil, err
	}
	return &latencyConn{Conn: conn, delay: l.delay}, nil
}

func (l *latencyStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	conn, err := l.StreamLayer.Dial(address, timeout)
	if err != nil {
		return nil, err
	}
	return &latencyConn{Conn: conn, delay: l.delay}, nil
}

// latencyConn wraps net.Conn and injects delay on Write.
type latencyConn struct {
	net.Conn
	delay time.Duration
}

func (c *latencyConn) Write(b []byte) (int, error) {
	time.Sleep(c.delay)
	return c.Conn.Write(b)
}

// newLatencyTransport creates a raft.NetworkTransport with latency injection.
func newLatencyTransport(bindAddr string, delay time.Duration, logger hclog.Logger) (*raft.NetworkTransport, error) {
	addr, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	base := &tcpStreamLayer{listener: listener}
	var stream raft.StreamLayer = base
	if delay > 0 {
		stream = &latencyStreamLayer{StreamLayer: base, delay: delay}
	}
	transport := raft.NewNetworkTransportWithLogger(stream, 10, 5*time.Second, logger)
	return transport, nil
}

// --- benchMetrics ---

// benchMetrics implements ConsensusMetrics and collects timing data for benchmarks.
type benchMetrics struct {
	mu            sync.Mutex
	marshalDurs   []float64
	raftApplyDurs []float64
	fsmApplyDurs  []float64
	payloadSizes  []float64
}

var _ ConsensusMetrics = (*benchMetrics)(nil)

func (m *benchMetrics) RecordCommitDuration(marshalSec, raftApplySec float64) {
	m.mu.Lock()
	m.marshalDurs = append(m.marshalDurs, marshalSec)
	m.raftApplyDurs = append(m.raftApplyDurs, raftApplySec)
	m.mu.Unlock()
}

func (m *benchMetrics) RecordCommitPayloadSize(payloadBytes float64) {
	m.mu.Lock()
	m.payloadSizes = append(m.payloadSizes, payloadBytes)
	m.mu.Unlock()
}

func (m *benchMetrics) RecordFSMApplyDuration(seconds float64) {
	m.mu.Lock()
	m.fsmApplyDurs = append(m.fsmApplyDurs, seconds)
	m.mu.Unlock()
}

func (m *benchMetrics) reportTo(b *testing.B) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.marshalDurs) > 0 {
		b.ReportMetric(avg(m.marshalDurs)*1e9, "ns/marshal")
	}
	if len(m.raftApplyDurs) > 0 {
		b.ReportMetric(avg(m.raftApplyDurs)*1e9, "ns/raftApply")
	}
	if len(m.fsmApplyDurs) > 0 {
		b.ReportMetric(avg(m.fsmApplyDurs)*1e9, "ns/fsmApply")
	}
	if len(m.payloadSizes) > 0 {
		b.ReportMetric(avg(m.payloadSizes), "bytes/payload")
	}
}

func (m *benchMetrics) reset() {
	m.mu.Lock()
	m.marshalDurs = m.marshalDurs[:0]
	m.raftApplyDurs = m.raftApplyDurs[:0]
	m.fsmApplyDurs = m.fsmApplyDurs[:0]
	m.payloadSizes = m.payloadSizes[:0]
	m.mu.Unlock()
}

func avg(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		sum += v
	}
	return sum / float64(len(s))
}

// --- benchCluster ---

type benchCluster struct {
	nodes   []*RaftConsensus
	metrics *benchMetrics
	tempDir string
}

func newBenchCluster(tb testing.TB, nodeCount int, latency time.Duration, opts ...func(*RaftConsensusConfig)) *benchCluster {
	tb.Helper()
	require := require.New(tb)

	tmpDir := tb.TempDir()
	logger := testlog.Logger(tb, log.LevelWarn)
	hcLogger := hclog.NewNullLogger()

	now := uint64(time.Now().Unix())
	rollupCfg := &rollup.Config{CanyonTime: &now}

	bm := &benchMetrics{}
	c := &benchCluster{
		metrics: bm,
		tempDir: tmpDir,
	}

	// Create all nodes
	for i := 0; i < nodeCount; i++ {
		transport, err := newLatencyTransport("127.0.0.1:0", latency, hcLogger)
		require.NoError(err)

		cfg := &RaftConsensusConfig{
			ServerID:           fmt.Sprintf("node-%d", i),
			ListenAddr:         "127.0.0.1",
			ListenPort:         0,
			StorageDir:         tmpDir,
			Bootstrap:          i == 0,
			RollupCfg:          rollupCfg,
			SnapshotInterval:   120 * time.Second,
			SnapshotThreshold:  8192,
			TrailingLogs:       10240,
			HeartbeatTimeout:   1000 * time.Millisecond,
			LeaderLeaseTimeout: 500 * time.Millisecond,
			Metrics:            bm,
			Transport:          transport,
		}
		for _, opt := range opts {
			opt(cfg)
		}

		node, err := NewRaftConsensus(logger, cfg)
		require.NoError(err)
		c.nodes = append(c.nodes, node)
	}

	// Wait for node 0 to become leader
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case <-c.nodes[0].LeaderCh():
	case <-timer.C:
		tb.Fatal("timed out waiting for leader election")
	}

	// Small delay to let leadership stabilize
	time.Sleep(100 * time.Millisecond)

	// Add remaining nodes as voters
	for i := 1; i < nodeCount; i++ {
		addr := string(c.nodes[i].transport.LocalAddr())
		err := c.nodes[0].AddVoter(fmt.Sprintf("node-%d", i), addr, 0)
		require.NoError(err)
	}

	// Wait for cluster to stabilize
	if nodeCount > 1 {
		time.Sleep(500 * time.Millisecond)
	}

	return c
}

func (c *benchCluster) leader() *RaftConsensus {
	for _, n := range c.nodes {
		if n.Leader() {
			return n
		}
	}
	return c.nodes[0]
}

func (c *benchCluster) shutdown() {
	for _, n := range c.nodes {
		_ = n.Shutdown()
	}
}

// --- payload generator ---

// generateRealisticTx builds a byte slice resembling an RLP-encoded EIP-1559
// smart contract interaction. Real OP Stack transactions have highly structured,
// repetitive fields (chain ID, gas params, zero-padded ABI arguments, common
// function selectors) that compress well, plus a 65-byte random ECDSA signature
// that does not. With ~65 bytes random out of ~200 bytes structured, payloads
// compress at roughly 3:1, matching observed production ratios.
//
// Layout (total 350 bytes):
//
//	[0]:       type prefix (0x02)
//	[1-2]:     RLP header
//	[3-10]:    chain ID + nonce (low entropy)
//	[11-30]:   gas fields (identical across txs)
//	[31-50]:   to-address (few unique targets, zero-padded)
//	[51-54]:   value (zero for contract calls)
//	[55-58]:   calldata length + selector (repeated)
//	[59-282]:  ABI-encoded args (multi-param call with nested data, mostly zero-padded)
//	[283-284]: access list (empty)
//	[285]:     v (0 or 1)
//	[286-317]: r signature (RANDOM - incompressible)
//	[318-349]: s signature (RANDOM - incompressible)
//
// Real Uniswap/multicall txs commonly have 6-7 ABI-encoded uint256/address
// params (each 32 bytes, mostly zeros), making 350 bytes realistic for
// DeFi swap or multicall interactions. The 64 random signature bytes in
// 350 total gives ~18% incompressible content, producing ~3:1 compression
// with S2.
func generateRealisticTx(rng *rand.Rand, txIndex int) []byte {
	tx := make([]byte, 350) // starts all-zero, which is our compressible baseline

	// --- Fixed/repeated fields (compressible) ---

	tx[0] = 0x02 // EIP-1559 type
	tx[1] = 0xf8 // RLP header
	tx[2] = 0xc5

	tx[3] = 0x0a // chain ID = 10 (OP Mainnet)

	// Nonce: low-entropy, sequential
	tx[8] = byte(txIndex >> 8)
	tx[9] = byte(txIndex)

	// Gas fields: identical across all txs in a block (very compressible)
	tx[11] = 0x59 // maxPriorityFeePerGas
	tx[12] = 0x68
	tx[13] = 0x20
	tx[21] = 0x02 // maxFeePerGas
	tx[22] = 0x54
	tx[23] = 0x0b
	tx[24] = 0xe4
	tx[25] = 0x00
	tx[26] = 0x01 // gasLimit
	tx[27] = 0x00
	tx[28] = 0x00

	// To-address: small set of popular contracts (e.g., Uniswap, USDC)
	addrSeed := rng.Intn(4)
	tx[31] = 0x42 // common prefix
	tx[32] = 0x00
	tx[33] = byte(addrSeed)
	tx[34] = 0xA0
	tx[35] = 0xb8
	tx[36] = 0x6a
	// bytes 37-50 stay zero (address zero-padding)

	// Calldata: function selector (repeated across txs) + ABI-encoded args
	tx[55] = 0xb9 // RLP data length (2-byte length)
	tx[56] = 0x00
	tx[57] = 0xa4 // 164 bytes of calldata
	tx[58] = 0xa9 // swap(address,uint256,uint256,address,uint256) selector
	tx[59] = 0x05
	tx[60] = 0x9c
	tx[61] = 0xbb

	// ABI arg1: address padded to 32 bytes (bytes 62-93)
	tx[91] = byte(rng.Intn(256))
	tx[92] = byte(rng.Intn(256))

	// ABI arg2: uint256 amount padded to 32 bytes (bytes 94-125)
	tx[123] = byte(rng.Intn(256))
	tx[124] = byte(rng.Intn(256))
	tx[125] = byte(rng.Intn(256))

	// ABI arg3: uint256 minOut padded to 32 bytes (bytes 126-157)
	tx[155] = byte(rng.Intn(256))
	tx[156] = byte(rng.Intn(256))

	// ABI arg4: address recipient padded to 32 bytes (bytes 158-189)
	tx[187] = byte(rng.Intn(256))
	tx[188] = byte(rng.Intn(256))

	// ABI arg5: uint256 deadline padded to 32 bytes (bytes 190-221)
	// Deadlines are usually current timestamp + small offset, very repetitive
	tx[218] = 0x67
	tx[219] = 0x5e
	tx[220] = 0xab
	tx[221] = 0xc0

	// ABI arg6: bytes offset (uint256, mostly zeros) (bytes 222-253)
	tx[251] = 0x01
	tx[252] = 0x00

	// ABI arg7: bytes length + data (bytes 254-281)
	tx[278] = 0x20 // length = 32
	tx[279] = byte(rng.Intn(256))

	tx[283] = 0xc0              // empty access list
	tx[284] = byte(rng.Intn(2)) // v recovery bit

	// --- ECDSA signature (incompressible, 64 random bytes) ---
	rng.Read(tx[286:350])

	return tx
}

// generatePayloadEnvelope creates a BlockV3 ExecutionPayloadEnvelope with
// realistic transaction data. The transactions are structured to compress at
// roughly 3:1, matching observed production compression ratios.
func generatePayloadEnvelope(rng *rand.Rand, blockNum uint64, targetSSZSize int) *eth.ExecutionPayloadEnvelope {
	hash := common.Hash{}
	rng.Read(hash[:])
	parentHash := common.Hash{}
	rng.Read(parentHash[:])
	beaconRoot := common.Hash{}
	rng.Read(beaconRoot[:])

	one := hexutil.Uint64(1)

	// Each realistic tx is 350 bytes.
	txSize := 350
	// Estimate overhead: SSZ envelope is ~600 bytes without transactions
	overhead := 600
	txDataTarget := targetSSZSize - overhead
	if txDataTarget < 0 {
		txDataTarget = 0
	}

	numTxs := txDataTarget / txSize
	if numTxs < 1 && txDataTarget > 0 {
		numTxs = 1
	}

	txs := make([]eth.Data, numTxs)
	for i := 0; i < numTxs; i++ {
		txs[i] = generateRealisticTx(rng, i)
	}

	return &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &beaconRoot,
		ExecutionPayload: &eth.ExecutionPayload{
			ParentHash:    parentHash,
			BlockHash:     hash,
			BlockNumber:   eth.Uint64Quantity(blockNum),
			Timestamp:     hexutil.Uint64(time.Now().Unix()),
			Transactions:  txs,
			ExtraData:     []byte("benchmark"),
			Withdrawals:   &types.Withdrawals{},
			ExcessBlobGas: &one,
			BlobGasUsed:   &one,
		},
	}
}
