package silhouette

import (
	"context"
	"crypto/ecdsa"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	coreparams "github.com/ethereum-optimism/optimism/op-core/params"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// A synthetic L1 and a synthetic proof stream, so the acceptance rules, the forced-extension
// convention and the transcode can all be exercised without a chain.
//
// The L1 is deliberately a plain deterministic ladder: 12-second blocks, hash = keccak("l1", number),
// which makes every expectation in these tests a closed-form value rather than something read back
// out of the code under test.

const (
	l1BlockTime  = uint64(12)
	l2BlockTime  = uint64(2)
	l1GenesisNum = uint64(1000)
	l1GenesisT   = uint64(1_700_000_000)
	seqWindow    = uint64(60) // ~12 minutes of L1, small enough to expire inside a test
)

func l1Hash(num uint64) common.Hash {
	return crypto.Keccak256Hash([]byte("l1"), new(big.Int).SetUint64(num).Bytes())
}

func l1Time(num uint64) uint64 {
	return l1GenesisT + (num-l1GenesisNum)*l1BlockTime
}

// fakeL1 is a canonical L1 ladder, optionally reorged above a height.
type fakeL1 struct {
	head uint64
	// txsByBlock holds the proof-batch transactions planted in each L1 block.
	txsByBlock map[uint64]types.Transactions
	// reorgAbove, when non-zero, makes blocks strictly above it hash differently: the same ladder on
	// a different fork, which is what an L1 reorg looks like to this source.
	reorgAbove uint64
	// calls counts L1 reads, so a test can assert the hot path stays free of them.
	calls int
}

func (f *fakeL1) FetchReceipts(_ context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	num, ok := f.byHash(hash)
	if !ok {
		return nil, nil, ethereum.NotFound
	}
	return f.info(num), optypes.Receipts{}, nil
}

func (f *fakeL1) L1BlockRefByLabel(_ context.Context, _ eth.BlockLabel) (eth.L1BlockRef, error) {
	return f.ref(f.head), nil
}

func newFakeL1(head uint64) *fakeL1 {
	return &fakeL1{head: head, txsByBlock: map[uint64]types.Transactions{}}
}

func (f *fakeL1) hashOf(num uint64) common.Hash {
	if f.reorgAbove != 0 && num > f.reorgAbove {
		return crypto.Keccak256Hash([]byte("l1-fork"), new(big.Int).SetUint64(num).Bytes())
	}
	return l1Hash(num)
}

func (f *fakeL1) ref(num uint64) eth.L1BlockRef {
	r := eth.L1BlockRef{Hash: f.hashOf(num), Number: num, Time: l1Time(num)}
	if num > 0 {
		r.ParentHash = f.hashOf(num - 1)
	}
	return r
}

func (f *fakeL1) L1BlockRefByNumber(_ context.Context, num uint64) (eth.L1BlockRef, error) {
	f.calls++
	if num > f.head || num < l1GenesisNum {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return f.ref(num), nil
}

func (f *fakeL1) byHash(hash common.Hash) (uint64, bool) {
	for n := l1GenesisNum; n <= f.head; n++ {
		if f.hashOf(n) == hash {
			return n, true
		}
	}
	return 0, false
}

func (f *fakeL1) L1BlockRefByHash(_ context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	f.calls++
	num, ok := f.byHash(hash)
	if !ok {
		return eth.L1BlockRef{}, ethereum.NotFound
	}
	return f.ref(num), nil
}

func (f *fakeL1) InfoByHash(_ context.Context, hash common.Hash) (eth.BlockInfo, error) {
	f.calls++
	num, ok := f.byHash(hash)
	if !ok {
		return nil, ethereum.NotFound
	}
	return f.info(num), nil
}

func (f *fakeL1) InfoAndTxsByHash(_ context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	f.calls++
	num, ok := f.byHash(hash)
	if !ok {
		return nil, nil, ethereum.NotFound
	}
	return f.info(num), f.txsByBlock[num], nil
}

func (f *fakeL1) info(num uint64) eth.BlockInfo {
	beacon := crypto.Keccak256Hash([]byte("beacon"), new(big.Int).SetUint64(num).Bytes())
	zero := uint64(0)
	return &testBlockInfo{
		hash:      f.hashOf(num),
		parent:    f.hashOf(num - 1),
		number:    num,
		time:      l1Time(num),
		mixDigest: crypto.Keccak256Hash([]byte("randao"), new(big.Int).SetUint64(num).Bytes()),
		baseFee:   big.NewInt(7),
		beacon:    &beacon,
		excess:    &zero,
	}
}

// testBlockInfo is the minimum eth.BlockInfo the L1-info transaction builder reads.
type testBlockInfo struct {
	hash, parent common.Hash
	number, time uint64
	mixDigest    common.Hash
	baseFee      *big.Int
	beacon       *common.Hash
	excess       *uint64
}

func (b *testBlockInfo) Hash() common.Hash                        { return b.hash }
func (b *testBlockInfo) ParentHash() common.Hash                  { return b.parent }
func (b *testBlockInfo) Coinbase() common.Address                 { return common.Address{} }
func (b *testBlockInfo) Root() common.Hash                        { return common.Hash{} }
func (b *testBlockInfo) NumberU64() uint64                        { return b.number }
func (b *testBlockInfo) Time() uint64                             { return b.time }
func (b *testBlockInfo) MixDigest() common.Hash                   { return b.mixDigest }
func (b *testBlockInfo) BaseFee() *big.Int                        { return b.baseFee }
func (b *testBlockInfo) ExcessBlobGas() *uint64                   { return b.excess }
func (b *testBlockInfo) ReceiptHash() common.Hash                 { return types.EmptyReceiptsHash }
func (b *testBlockInfo) GasUsed() uint64                          { return 0 }
func (b *testBlockInfo) BlobGasUsed() *uint64                     { return b.excess }
func (b *testBlockInfo) GasLimit() uint64                         { return 30_000_000 }
func (b *testBlockInfo) ParentBeaconRoot() *common.Hash           { return b.beacon }
func (b *testBlockInfo) WithdrawalsRoot() *common.Hash            { return nil }
func (b *testBlockInfo) Extra() []byte                            { return nil }
func (b *testBlockInfo) BlobBaseFee(*params.ChainConfig) *big.Int { return big.NewInt(1) }

// fakeBlobs serves blobs by versioned hash, as planted by plantBatch.
type fakeBlobs struct {
	byHash map[common.Hash]*eth.Blob
	fail   error
}

func (f *fakeBlobs) GetBlobsByHash(_ context.Context, _ uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	if f.fail != nil {
		return nil, f.fail
	}
	out := make([]*eth.Blob, 0, len(hashes))
	for _, h := range hashes {
		blob, ok := f.byHash[h]
		if !ok {
			return nil, ethereum.NotFound
		}
		out = append(out, blob)
	}
	return out, nil
}

// testEnv wires a source over the fakes.
type testEnv struct {
	t      *testing.T
	l1     *fakeL1
	blobs  *fakeBlobs
	facts  *FactStore
	src    *DataSource
	cfg    *Config
	rollup *rollup.Config
	sysCfg eth.SystemConfig
	key    *ecdsaKey
}

// silhouetteRollupConfig is P's rollup config as these tests use it: every fork active at genesis,
// a FINITE sequencing window (DR-2), and epochs that advance. It is the same shape the generated
// artifact has — see RollupConfigFor.
func silhouetteRollupConfig() *rollup.Config {
	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: l1Hash(l1GenesisNum), Number: l1GenesisNum},
			L2:     eth.BlockID{Hash: crypto.Keccak256Hash([]byte("l2-genesis")), Number: 0},
			L2Time: l1GenesisT,
			SystemConfig: eth.SystemConfig{
				GasLimit:      30_000_000,
				EIP1559Params: eth.Bytes8{0, 0, 0, 250, 0, 0, 0, 6},
				MinBaseFee:    0,
			},
		},
		BlockTime:              l2BlockTime,
		MaxSequencerDrift:      1800,
		SeqWindowSize:          seqWindow,
		ChannelTimeoutBedrock:  300,
		L1ChainID:              big.NewInt(11155111),
		L2ChainID:              big.NewInt(424250),
		RegolithTime:           u64ptr(0),
		CanyonTime:             u64ptr(0),
		DeltaTime:              u64ptr(0),
		EcotoneTime:            u64ptr(0),
		FjordTime:              u64ptr(0),
		GraniteTime:            u64ptr(0),
		HoloceneTime:           u64ptr(0),
		IsthmusTime:            u64ptr(0),
		JovianTime:             u64ptr(0),
		BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000424250"),
		DepositContractAddress: common.HexToAddress("0x0000000000000000000000000000000000000dead"),
		L1SystemConfigAddress:  common.HexToAddress("0x0000000000000000000000000000000000005ca1"),
		ChainOpConfig: &coreparams.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       250,
			EIP1559DenominatorCanyon: u64ptr(250),
		},
	}
}

func u64ptr(v uint64) *uint64 { return &v }

// sepoliaChainConfig is the L1 chain config the L1-info transaction builder reads for its blob
// base-fee field. Sepolia because that is where Cove's live proof stream is, so the dark-launch gate
// and these tests read the same config.
func sepoliaChainConfig() *params.ChainConfig { return params.SepoliaChainConfig }

// sepoliaRandao is the mixDigest the fake L1 reports for a block, so tests can state the expected
// prevRandao in closed form rather than reading it back out of the code under test.
func sepoliaRandao(num uint64) common.Hash {
	return crypto.Keccak256Hash([]byte("randao"), new(big.Int).SetUint64(num).Bytes())
}

func newTestEnv(t *testing.T, head uint64) *testEnv {
	rollupCfg := silhouetteRollupConfig()
	key := newKey(t)
	cfg := &Config{
		L1ChainID:        11155111,
		Submitter:        key.addr,
		Inbox:            common.HexToAddress("0x00000000000000000000000000000000000b0000"),
		RollupConfigHash: common.HexToHash("0x1111"),
		DepSetHash:       common.HexToHash("0x2222"),
		L1StartBlock:     l1GenesisNum,
		ProofType:        ProofTypeAttested,
		Anchor: Anchor{
			OutputRoot:  common.HexToHash("0xa0"),
			BlockNumber: 0,
			BlockHash:   rollupCfg.Genesis.L2.Hash,
			Timestamp:   l1GenesisT,
			L1Origin:    eth.BlockID{Hash: l1Hash(l1GenesisNum), Number: l1GenesisNum},
		},
	}
	require.NoError(t, cfg.Check())
	env := &testEnv{
		t:      t,
		l1:     newFakeL1(head),
		blobs:  &fakeBlobs{byHash: map[common.Hash]*eth.Blob{}},
		facts:  &FactStore{},
		cfg:    cfg,
		rollup: rollupCfg,
		sysCfg: rollupCfg.Genesis.SystemConfig,
		key:    key,
	}
	verifier, err := cfg.NewVerifier()
	require.NoError(t, err)
	env.src = NewDataSource(testlog.Logger(t, 3), cfg, rollupCfg, params.SepoliaChainConfig,
		env.sysCfg, env.l1, env.blobs, verifier, env.facts)
	return env
}

// batchSpec describes a proof batch to plant on L1.
type batchSpec struct {
	prevRoot   common.Hash
	firstBlock uint64
	firstTime  uint64
	count      int
	l1Head     uint64
	carrier    uint64
	// mutate lets a test damage the batch after it is built but before it is posted.
	mutate func(*proofbatch.ProofBatch)
	proof  []byte
	// imports is the import list to put on the FIRST block of the batch. Named per-block rather than
	// per-batch because the interesting cases are about one block's dependencies.
	imports []proofbatch.ExecMsg
	// wireVersion frames the envelope at an explicit version; zero means the codec's current one.
	// It is what lets a test post to a verifier configured for the other version.
	wireVersion uint8
}

func (s batchSpec) version() uint8 {
	if s.wireVersion == 0 {
		return proofbatch.Version
	}
	return s.wireVersion
}

// buildBatch produces a well-formed batch: real-looking hashes and roots, contiguous blocks, exact
// block-time spacing.
func (e *testEnv) buildBatch(s batchSpec) *proofbatch.ProofBatch {
	b := &proofbatch.ProofBatch{
		PrevOutputRoot:   s.prevRoot,
		L1Head:           e.l1.hashOf(s.l1Head),
		RollupConfigHash: e.cfg.RollupConfigHash,
		DepSetHash:       e.cfg.DepSetHash,
		ExportPolicyHash: e.cfg.ExportPolicy(),
	}
	for i := 0; i < s.count; i++ {
		num := s.firstBlock + uint64(i)
		timestamp := s.firstTime + uint64(i)*l2BlockTime
		seed := new(big.Int).SetUint64(num).Bytes()
		block := proofbatch.BlockExport{
			Number:                   num,
			Timestamp:                timestamp,
			Hash:                     crypto.Keccak256Hash([]byte("l2"), seed),
			StateRoot:                crypto.Keccak256Hash([]byte("state"), seed),
			MessagePasserStorageRoot: crypto.Keccak256Hash([]byte("mp"), seed),
		}
		if proofbatch.VersionHasL1Origins(s.version()) {
			originNumber := l1GenesisNum + (timestamp-l1GenesisT)/l1BlockTime
			originNumber = min(originNumber, s.l1Head)
			block.L1Origin = eth.BlockID{Hash: e.l1.hashOf(originNumber), Number: originNumber}
			block.SequenceNumber = (timestamp - l1Time(originNumber)) / l2BlockTime
		}
		b.Blocks = append(b.Blocks, block)
	}
	if len(s.imports) > 0 {
		b.Blocks[0].ExecMsgs = s.imports
	}
	b.NewOutputRoot = b.Blocks[len(b.Blocks)-1].OutputRoot()
	if s.mutate != nil {
		s.mutate(b)
	}
	return b
}

// plant encodes a batch, packs it into blobs and puts a signed blob transaction from the submitter
// into the carrying L1 block.
func (e *testEnv) plant(b *proofbatch.ProofBatch, s batchSpec) {
	raw, err := proofbatch.EncodeAs(b, s.proof, s.version())
	require.NoError(e.t, err)
	blobs, err := proofbatch.ToBlobs(raw)
	require.NoError(e.t, err)
	hashes := make([]common.Hash, 0, len(blobs))
	for _, blob := range blobs {
		commit, err := blob.ComputeKZGCommitment()
		require.NoError(e.t, err)
		h := eth.KZGToVersionedHash(commit)
		e.blobs.byHash[h] = blob
		hashes = append(hashes, h)
	}
	e.l1.txsByBlock[s.carrier] = append(e.l1.txsByBlock[s.carrier], e.key.blobTx(e.t, e.cfg.Inbox, hashes))
	if s.carrier > e.l1.head {
		e.l1.head = s.carrier
	}
}

// derive runs the source over one L1 block and returns the payloads it emitted.
func (e *testEnv) derive(l1Number uint64) [][]byte {
	e.t.Helper()
	it, err := e.src.OpenData(context.Background(), e.l1.ref(l1Number), common.Address{})
	require.NoError(e.t, err)
	var out [][]byte
	for {
		data, err := it.Next(context.Background())
		if err != nil {
			require.ErrorIs(e.t, err, io.EOF, "iterator must drain with io.EOF")
			break
		}
		out = append(out, data)
	}
	return out
}

// ecdsaKey is the proof-batch submitter key: the authenticity rule for the whole envelope stream is
// "from this address, to that inbox", so a test that wants a batch accepted has to sign like it.
type ecdsaKey struct {
	priv *ecdsa.PrivateKey
	addr common.Address
}

func newKey(t *testing.T) *ecdsaKey {
	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	return &ecdsaKey{priv: priv, addr: crypto.PubkeyToAddress(priv.PublicKey)}
}

// blobTx builds a signed type-3 transaction carrying the given blob hashes.
func (k *ecdsaKey) blobTx(t *testing.T, to common.Address, hashes []common.Hash) *types.Transaction {
	signer := types.LatestSignerForChainID(big.NewInt(11155111))
	tx, err := types.SignNewTx(k.priv, signer, &types.BlobTx{
		ChainID:    uint256.NewInt(11155111),
		Nonce:      0,
		GasTipCap:  uint256.NewInt(1),
		GasFeeCap:  uint256.NewInt(1000),
		Gas:        21000,
		To:         to,
		BlobHashes: hashes,
		BlobFeeCap: uint256.NewInt(1000),
	})
	require.NoError(t, err)
	return tx
}

// otherKey is a signer that is NOT the configured submitter.
func otherTx(t *testing.T, to common.Address, hashes []common.Hash) *types.Transaction {
	return newKey(t).blobTx(t, to, hashes)
}
