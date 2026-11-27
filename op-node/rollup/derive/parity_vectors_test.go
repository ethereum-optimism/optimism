package derive

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestParityVectors pins op-node's outputs for inputs to derivation steps that kona implements
// too. kona's derivation crate reads the same file and compares its own outputs with op-node's
// (rust/kona/crates/protocol/derive/tests/parity.rs), so every input on which the two clients
// differ is either pinned there as a known difference or fails.
//
// The test runs op-node on the committed inputs, so it fails when op-node's behaviour changes, and
// checks that the committed inputs are the ones the cases below build. After changing op-node or
// adding or changing a case, rewrite the file and update kona's pinned differences:
//
//	UPDATE_PARITY_VECTORS=1 go test ./op-node/rollup/derive -run TestParityVectors
func TestParityVectors(t *testing.T) {
	generated := buildParityVectors(t)
	if os.Getenv("UPDATE_PARITY_VECTORS") != "" {
		generated.run(t, true)
		data, err := json.MarshalIndent(generated, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(parityVectorsPath, append(data, '\n'), 0o644))
		return
	}

	data, err := os.ReadFile(parityVectorsPath)
	require.NoError(t, err)
	var committed parityVectors
	require.NoError(t, json.Unmarshal(data, &committed))
	require.Equal(t, generated.caseNames(), committed.caseNames(),
		"the committed vectors are out of date, rewrite them with UPDATE_PARITY_VECTORS=1")
	generated.requireSameInputs(t, &committed)
	committed.run(t, false)
}

// parityVectorsPath is where kona's derivation crate reads the vectors from.
const parityVectorsPath = "../../../rust/kona/crates/protocol/derive/testdata/parity_vectors.json"

// parityVectors holds the inputs and op-node's outputs for each derivation step both clients
// implement. Cases name their configs, which are stored once.
type parityVectors struct {
	RollupConfigs   map[string]*rollup.Config                                    `json:"rollupConfigs"`
	L1ChainConfigs  map[string]*params.ChainConfig                               `json:"l1ChainConfigs"`
	ChannelReader   []parityCase[parityChannelInput, parityChannelOutput]        `json:"channelReader"`
	L1InfoDeposit   []parityCase[parityL1InfoDepositInput, parityBytesOutput]    `json:"l1InfoDeposit"`
	L1InfoDecode    []parityCase[parityL1InfoDecodeInput, parityBytesOutput]     `json:"l1InfoDecode"`
	DepositLog      []parityCase[parityDepositLogInput, parityBytesOutput]       `json:"depositLog"`
	SystemConfigLog []parityCase[paritySystemConfigLogInput, paritySystemConfig] `json:"systemConfigLog"`
	// SystemConfigFromBlock rebuilds the system config from an L2 block, as a pipeline reset does.
	SystemConfigFromBlock []parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput] `json:"systemConfigFromBlock"`
}

type parityCase[I, O any] struct {
	Name   string `json:"name"`
	Input  I      `json:"input"`
	OpNode O      `json:"opNode"`
}

// parityChannelInput is one channel, as the channel bank hands it to the channel reader, read at
// an L1 origin with the given timestamp.
type parityChannelInput struct {
	RollupConfig string        `json:"rollupConfig"`
	OriginTime   uint64        `json:"originTime"`
	Channel      hexutil.Bytes `json:"channel"`
}

// parityChannelOutput lists the batches the channel reader passes on, in order.
type parityChannelOutput struct {
	Batches []parityBatch `json:"batches"`
	// Error is set when the reader returns an error that is not temporary. op-node's reader drops a
	// channel that fails to decompress or decode without returning an error, so for op-node this
	// only happens on a critical error.
	Error bool `json:"error,omitempty"`
}

type parityBatch struct {
	Singular *paritySingularBatch `json:"singular,omitempty"`
	Span     *paritySpanBatch     `json:"span,omitempty"`
}

type paritySingularBatch struct {
	ParentHash   common.Hash     `json:"parentHash"`
	EpochNum     uint64          `json:"epochNum"`
	EpochHash    common.Hash     `json:"epochHash"`
	Timestamp    uint64          `json:"timestamp"`
	Transactions []hexutil.Bytes `json:"transactions"`
}

type paritySpanBatch struct {
	ParentCheck   hexutil.Bytes          `json:"parentCheck"`
	L1OriginCheck hexutil.Bytes          `json:"l1OriginCheck"`
	Blocks        []paritySpanBatchBlock `json:"blocks"`
}

type paritySpanBatchBlock struct {
	EpochNum     uint64          `json:"epochNum"`
	Timestamp    uint64          `json:"timestamp"`
	Transactions []hexutil.Bytes `json:"transactions"`
}

// parityBytesOutput is an encoding, or an error flag when the step fails.
type parityBytesOutput struct {
	Bytes hexutil.Bytes `json:"bytes,omitempty"`
	Error bool          `json:"error,omitempty"`
}

type parityL1InfoDepositInput struct {
	RollupConfig   string           `json:"rollupConfig"`
	L1ChainConfig  string           `json:"l1ChainConfig"`
	SystemConfig   eth.SystemConfig `json:"systemConfig"`
	SequenceNumber uint64           `json:"sequenceNumber"`
	// L1Header is the RLP encoding of the L1 origin's header.
	L1Header    hexutil.Bytes `json:"l1Header"`
	L2Timestamp uint64        `json:"l2Timestamp"`
}

// parityL1InfoDecodeInput is the calldata of an L2 block's L1-info deposit. The output is the
// decoded value encoded again in the same layout, so it is equal for both clients exactly when
// they decode the same fields.
type parityL1InfoDecodeInput struct {
	RollupConfig string        `json:"rollupConfig"`
	L2Timestamp  uint64        `json:"l2Timestamp"`
	Calldata     hexutil.Bytes `json:"calldata"`
}

type parityLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

type parityDepositLogInput struct {
	Log       parityLog   `json:"log"`
	BlockHash common.Hash `json:"blockHash"`
	LogIndex  uint64      `json:"logIndex"`
}

// paritySystemConfigLogInput is a successful L1 receipt with one log, applied to a system config.
type paritySystemConfigLogInput struct {
	RollupConfig string           `json:"rollupConfig"`
	L1Time       uint64           `json:"l1Time"`
	SystemConfig eth.SystemConfig `json:"systemConfig"`
	Log          parityLog        `json:"log"`
}

// paritySystemConfig is a system config by the values derivation reads from it, so that layouts
// that carry the same values compare equal.
type paritySystemConfig struct {
	BatcherAddr          common.Address `json:"batcherAddr"`
	Overhead             eth.Bytes32    `json:"overhead"`
	Scalar               eth.Bytes32    `json:"scalar"`
	GasLimit             uint64         `json:"gasLimit"`
	EIP1559Denominator   uint32         `json:"eip1559Denominator"`
	EIP1559Elasticity    uint32         `json:"eip1559Elasticity"`
	OperatorFeeScalar    uint32         `json:"operatorFeeScalar"`
	OperatorFeeConstant  uint64         `json:"operatorFeeConstant"`
	MinBaseFee           uint64         `json:"minBaseFee"`
	DAFootprintGasScalar uint16         `json:"daFootprintGasScalar"`
}

func newParitySystemConfig(sysCfg eth.SystemConfig) paritySystemConfig {
	operatorFee := sysCfg.OperatorFee()
	daFootprintGasScalar := sysCfg.DAFootprintGasScalar
	if daFootprintGasScalar == 0 {
		// Derivation reads a zero scalar as the default, and kona stores it that way.
		daFootprintGasScalar = DAFootprintGasScalarDefault
	}
	return paritySystemConfig{
		BatcherAddr:          sysCfg.BatcherAddr,
		Overhead:             sysCfg.Overhead,
		Scalar:               sysCfg.Scalar,
		GasLimit:             sysCfg.GasLimit,
		EIP1559Denominator:   binary.BigEndian.Uint32(sysCfg.EIP1559Params[:4]),
		EIP1559Elasticity:    binary.BigEndian.Uint32(sysCfg.EIP1559Params[4:]),
		OperatorFeeScalar:    operatorFee.Scalar,
		OperatorFeeConstant:  operatorFee.Constant,
		MinBaseFee:           sysCfg.MinBaseFee,
		DAFootprintGasScalar: daFootprintGasScalar,
	}
}

// paritySystemConfigFromBlockInput is an L2 block with only the fields the rebuild reads: the
// header fields below and the L1-info deposit as its only transaction.
type paritySystemConfigFromBlockInput struct {
	RollupConfig string        `json:"rollupConfig"`
	Number       uint64        `json:"number"`
	Timestamp    uint64        `json:"timestamp"`
	GasLimit     uint64        `json:"gasLimit"`
	ExtraData    hexutil.Bytes `json:"extraData"`
	L1InfoTx     hexutil.Bytes `json:"l1InfoTx"`
}

type paritySystemConfigOutput struct {
	SystemConfig *paritySystemConfig `json:"systemConfig,omitempty"`
	Error        bool                `json:"error,omitempty"`
}

func (v *parityVectors) caseNames() map[string][]string {
	return map[string][]string{
		"channelReader":         parityCaseNames(v.ChannelReader),
		"l1InfoDeposit":         parityCaseNames(v.L1InfoDeposit),
		"l1InfoDecode":          parityCaseNames(v.L1InfoDecode),
		"depositLog":            parityCaseNames(v.DepositLog),
		"systemConfigLog":       parityCaseNames(v.SystemConfigLog),
		"systemConfigFromBlock": parityCaseNames(v.SystemConfigFromBlock),
	}
}

func parityCaseNames[I, O any](cases []parityCase[I, O]) []string {
	names := make([]string, len(cases))
	for i, c := range cases {
		names[i] = c.Name
	}
	return names
}

// run runs op-node on every case. With update set it records op-node's outputs, otherwise it checks
// them against the recorded ones.
func (v *parityVectors) run(t *testing.T, update bool) {
	runParityCases(t, update, "channelReader", v.ChannelReader, v.runChannelReader)
	runParityCases(t, update, "l1InfoDeposit", v.L1InfoDeposit, v.runL1InfoDeposit)
	runParityCases(t, update, "l1InfoDecode", v.L1InfoDecode, v.runL1InfoDecode)
	runParityCases(t, update, "depositLog", v.DepositLog, runParityDepositLog)
	runParityCases(t, update, "systemConfigLog", v.SystemConfigLog, v.runSystemConfigLog)
	runParityCases(t, update, "systemConfigFromBlock", v.SystemConfigFromBlock, v.runSystemConfigFromBlock)
}

func runParityCases[I, O any](t *testing.T, update bool, group string, cases []parityCase[I, O], run func(*testing.T, I) O) {
	for i := range cases {
		t.Run(group+"/"+cases[i].Name, func(t *testing.T) {
			out := run(t, cases[i].Input)
			if update {
				cases[i].OpNode = out
				return
			}
			require.Equal(t, cases[i].OpNode, out, "op-node's output changed, rewrite the vectors with UPDATE_PARITY_VECTORS=1")
		})
	}
}

// requireSameInputs checks that the committed configs and inputs are the ones the cases build.
// Channels are compared by everything but their bytes, which a compressor upgrade changes.
func (v *parityVectors) requireSameInputs(t *testing.T, committed *parityVectors) {
	const msg = "%s changed, rewrite the vectors with UPDATE_PARITY_VECTORS=1"
	require.JSONEqf(t, parityJSON(t, v.RollupConfigs), parityJSON(t, committed.RollupConfigs), msg, "a rollup config")
	require.JSONEqf(t, parityJSON(t, v.L1ChainConfigs), parityJSON(t, committed.L1ChainConfigs), msg, "an L1 chain config")
	for i, c := range v.ChannelReader {
		in := committed.ChannelReader[i].Input
		require.Equalf(t, []any{c.Input.RollupConfig, c.Input.OriginTime}, []any{in.RollupConfig, in.OriginTime},
			msg, "the input of channelReader case "+c.Name)
	}
	requireSameParityInputs(t, "l1InfoDeposit", v.L1InfoDeposit, committed.L1InfoDeposit)
	requireSameParityInputs(t, "l1InfoDecode", v.L1InfoDecode, committed.L1InfoDecode)
	requireSameParityInputs(t, "depositLog", v.DepositLog, committed.DepositLog)
	requireSameParityInputs(t, "systemConfigLog", v.SystemConfigLog, committed.SystemConfigLog)
	requireSameParityInputs(t, "systemConfigFromBlock", v.SystemConfigFromBlock, committed.SystemConfigFromBlock)
}

func requireSameParityInputs[I, O any](t *testing.T, group string, generated, committed []parityCase[I, O]) {
	for i := range generated {
		require.JSONEqf(t, parityJSON(t, generated[i].Input), parityJSON(t, committed[i].Input),
			"the input of %s case %q changed, rewrite the vectors with UPDATE_PARITY_VECTORS=1", group, generated[i].Name)
	}
}

func parityJSON(t *testing.T, value any) string {
	data, err := json.Marshal(value)
	require.NoError(t, err)
	return string(data)
}

func (v *parityVectors) rollupConfig(t *testing.T, name string) *rollup.Config {
	cfg, ok := v.RollupConfigs[name]
	require.Truef(t, ok, "unknown rollup config %q", name)
	return cfg
}

// parityChannelSource hands the channel reader a single channel.
type parityChannelSource struct {
	origin  eth.L1BlockRef
	channel []byte
}

func (s *parityChannelSource) Origin() eth.L1BlockRef { return s.origin }

func (s *parityChannelSource) NextRawChannel(context.Context) ([]byte, error) {
	if s.channel == nil {
		return nil, io.EOF
	}
	channel := s.channel
	s.channel = nil
	return channel, nil
}

func (s *parityChannelSource) Reset(context.Context, eth.L1BlockRef, eth.SystemConfig) error {
	return io.EOF
}

func (s *parityChannelSource) FlushChannel() {}

func (v *parityVectors) runChannelReader(t *testing.T, in parityChannelInput) parityChannelOutput {
	cfg := v.rollupConfig(t, in.RollupConfig)
	source := &parityChannelSource{origin: eth.L1BlockRef{Time: in.OriginTime}, channel: in.Channel}
	reader := NewChannelInReader(cfg, testlog.Logger(t, log.LevelCrit), source, &testutils.TestDerivationMetrics{})
	out := parityChannelOutput{Batches: []parityBatch{}}
	for range 1000 {
		batch, err := reader.NextBatch(t.Context())
		switch {
		case errors.Is(err, io.EOF):
			return out
		case errors.Is(err, NotEnoughData), errors.Is(err, ErrTemporary):
			continue
		case err != nil:
			out.Error = true
			return out
		}
		out.Batches = append(out.Batches, newParityBatch(batch))
	}
	t.Fatal("the channel reader did not run out of data")
	return out
}

func newParityBatch(batch Batch) parityBatch {
	if singular, ok := batch.AsSingularBatch(); ok {
		return parityBatch{Singular: &paritySingularBatch{
			ParentHash:   singular.ParentHash,
			EpochNum:     uint64(singular.EpochNum),
			EpochHash:    singular.EpochHash,
			Timestamp:    singular.Timestamp,
			Transactions: nonNilTxs(singular.Transactions),
		}}
	}
	span, _ := batch.AsSpanBatch()
	out := &paritySpanBatch{
		ParentCheck:   span.ParentCheck[:],
		L1OriginCheck: span.L1OriginCheck[:],
		Blocks:        []paritySpanBatchBlock{},
	}
	for _, element := range span.Batches {
		out.Blocks = append(out.Blocks, paritySpanBatchBlock{
			EpochNum:     uint64(element.EpochNum),
			Timestamp:    element.Timestamp,
			Transactions: nonNilTxs(element.Transactions),
		})
	}
	return parityBatch{Span: out}
}

func nonNilTxs(txs []hexutil.Bytes) []hexutil.Bytes {
	if txs == nil {
		return []hexutil.Bytes{}
	}
	return txs
}

func (v *parityVectors) runL1InfoDeposit(t *testing.T, in parityL1InfoDepositInput) parityBytesOutput {
	l1Cfg, ok := v.L1ChainConfigs[in.L1ChainConfig]
	require.Truef(t, ok, "unknown L1 chain config %q", in.L1ChainConfig)
	var header types.Header
	require.NoError(t, rlp.DecodeBytes(in.L1Header, &header))
	data, err := L1InfoDepositBytes(v.rollupConfig(t, in.RollupConfig), l1Cfg, in.SystemConfig, in.SequenceNumber,
		eth.HeaderBlockInfo(&header), in.L2Timestamp)
	if err != nil {
		return parityBytesOutput{Error: true}
	}
	return parityBytesOutput{Bytes: data}
}

func (v *parityVectors) runL1InfoDecode(t *testing.T, in parityL1InfoDecodeInput) parityBytesOutput {
	cfg := v.rollupConfig(t, in.RollupConfig)
	info, err := L1BlockInfoFromBytes(cfg, in.L2Timestamp, in.Calldata)
	if err != nil {
		return parityBytesOutput{Error: true}
	}
	// Encode the value again in the layout it was decoded from, which L1BlockInfoFromBytes picks
	// by the block's fork.
	var data []byte
	switch {
	case isJovianButNotFirstBlock(cfg, in.L2Timestamp):
		data, err = info.marshalBinaryJovian()
	case isIsthmusButNotFirstBlock(cfg, in.L2Timestamp):
		data, err = info.marshalBinaryIsthmus()
	case isEcotoneButNotFirstBlock(cfg, in.L2Timestamp):
		data, err = info.marshalBinaryEcotone()
	default:
		data, err = info.marshalBinaryBedrock()
	}
	require.NoError(t, err)
	return parityBytesOutput{Bytes: data}
}

func runParityDepositLog(_ *testing.T, in parityDepositLogInput) parityBytesOutput {
	deposit, err := UnmarshalDepositLogEvent(&types.Log{
		Address:   in.Log.Address,
		Topics:    in.Log.Topics,
		Data:      in.Log.Data,
		BlockHash: in.BlockHash,
		Index:     uint(in.LogIndex),
	})
	if err != nil {
		return parityBytesOutput{Error: true}
	}
	data, err := deposit.MarshalBinary()
	if err != nil {
		return parityBytesOutput{Error: true}
	}
	return parityBytesOutput{Bytes: data}
}

func (v *parityVectors) runSystemConfigLog(t *testing.T, in paritySystemConfigLogInput) paritySystemConfig {
	sysCfg := in.SystemConfig
	receipt := &types.Receipt{
		Status: types.ReceiptStatusSuccessful,
		Logs:   []*types.Log{{Address: in.Log.Address, Topics: in.Log.Topics, Data: in.Log.Data}},
	}
	// A malformed update is skipped and reported, and derivation goes on with the config as it
	// stands, so the config is the outcome to compare.
	_ = UpdateSystemConfigWithL1Receipts(&sysCfg, []*types.Receipt{receipt}, v.rollupConfig(t, in.RollupConfig), in.L1Time)
	return newParitySystemConfig(sysCfg)
}

func (v *parityVectors) runSystemConfigFromBlock(t *testing.T, in paritySystemConfigFromBlockInput) paritySystemConfigOutput {
	sysCfg, err := PayloadToSystemConfig(v.rollupConfig(t, in.RollupConfig), &eth.ExecutionPayload{
		BlockNumber:  eth.Uint64Quantity(in.Number),
		Timestamp:    eth.Uint64Quantity(in.Timestamp),
		GasLimit:     eth.Uint64Quantity(in.GasLimit),
		ExtraData:    eth.BytesMax32(in.ExtraData),
		Transactions: []eth.Data{eth.Data(in.L1InfoTx)},
	})
	if err != nil {
		return paritySystemConfigOutput{Error: true}
	}
	out := newParitySystemConfig(sysCfg)
	return paritySystemConfigOutput{SystemConfig: &out}
}

// The parity rollup config activates forks at these L2 timestamps, so cases can sit on either
// side of each activation. The channel cases compare these with the L1 origin's timestamp, as the
// channel reader does.
const (
	parityDeltaTime     = 1000
	parityEcotoneTime   = 2000
	parityFjordTime     = 3000 // Granite and Holocene too
	parityIsthmusTime   = 4000
	parityJovianTime    = 5000
	parityKarstTime     = 6000
	parityPreDelta      = 500
	parityDeltaOrigin   = 1500
	parityEcotoneBlock  = parityEcotoneTime + 2
	parityHoloceneBlock = parityFjordTime + 2
	parityFjordOrigin   = 3500
	parityIsthmusBlock  = parityIsthmusTime + 2
	parityJovianBlock   = parityJovianTime + 2
)

var (
	parityBatcher    = common.HexToAddress("0x6887246668a3b87F54DeB3b94Ba47a6f63F32985")
	parityRecipient  = common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	parityBlockHash  = common.HexToHash("0x9c3e9d5e5f8c1ae0c2f6a2d6e3b1b0e8c4a7d2f1e0b9c8a7d6e5f4a3b2c1d0e9")
	parityParentHash = common.HexToHash("0x1d2c3b4a59687766554433221100ffeeddccbbaa99887766554433221100ffee")
	parityEpochHash  = common.HexToHash("0x0a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20212223242526272829")
	// parityKey signs the transactions in the channel cases. Signatures are deterministic, so the
	// vectors are reproducible.
	parityKey, _ = crypto.HexToECDSA("8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63")
	twoTo128     = new(big.Int).Lsh(big.NewInt(1), 128)
)

func parityRollupConfig() *rollup.Config {
	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.HexToHash("0x01"), Number: 0},
			L2:     eth.BlockID{Hash: common.HexToHash("0x02"), Number: 0},
			L2Time: 0,
			SystemConfig: eth.SystemConfig{
				BatcherAddr: parityBatcher,
				Overhead:    eth.Bytes32{31: 0xbc},
				Scalar:      eth.Bytes32{29: 0x0a, 30: 0x6f, 31: 0xe0},
				GasLimit:    30_000_000,
			},
		},
		BlockTime:              2,
		MaxSequencerDrift:      600,
		SeqWindowSize:          3600,
		ChannelTimeoutBedrock:  300,
		L1ChainID:              big.NewInt(1),
		L2ChainID:              big.NewInt(10),
		RegolithTime:           ptr.New(uint64(0)),
		CanyonTime:             ptr.New(uint64(0)),
		DeltaTime:              ptr.New(uint64(parityDeltaTime)),
		EcotoneTime:            ptr.New(uint64(parityEcotoneTime)),
		FjordTime:              ptr.New(uint64(parityFjordTime)),
		GraniteTime:            ptr.New(uint64(parityFjordTime)),
		HoloceneTime:           ptr.New(uint64(parityFjordTime)),
		IsthmusTime:            ptr.New(uint64(parityIsthmusTime)),
		JovianTime:             ptr.New(uint64(parityJovianTime)),
		KarstTime:              ptr.New(uint64(parityKarstTime)),
		BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000010"),
		DepositContractAddress: common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
		L1SystemConfigAddress:  common.HexToAddress("0x229047fed2591dbec1eF1118d64F7aF3dB9EB290"),
		ChainOpConfig: &opparams.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New(uint64(250)),
		},
	}
}

func buildParityVectors(t *testing.T) *parityVectors {
	mainnet := params.MainnetChainConfig
	customBlobs := *mainnet
	customBlobs.BlobScheduleConfig = &params.BlobScheduleConfig{
		Cancun: &params.BlobConfig{Target: 3, Max: 6, UpdateFraction: 2 * params.DefaultCancunBlobConfig.UpdateFraction},
		Prague: &params.BlobConfig{Target: 6, Max: 9, UpdateFraction: 3 * params.DefaultPragueBlobConfig.UpdateFraction},
		BPO1:   mainnet.BlobScheduleConfig.BPO1,
		BPO2:   mainnet.BlobScheduleConfig.BPO2,
	}
	lateBlobSchedule := parityRollupConfig()
	lateBlobSchedule.PectraBlobScheduleTime = ptr.New(*mainnet.OsakaTime + 86_400)

	v := &parityVectors{
		RollupConfigs: map[string]*rollup.Config{
			"staggered": parityRollupConfig(),
			// Forces the Cancun blob fee formula until a day after L1 activates Osaka.
			"late-pectra-blob-schedule": lateBlobSchedule,
		},
		L1ChainConfigs: map[string]*params.ChainConfig{
			"mainnet": mainnet,
			// Cancun and Prague blob fee update fractions that differ from mainnet's.
			"custom-blob-schedule": &customBlobs,
		},
	}
	v.ChannelReader = parityChannelCases(t, v.RollupConfigs["staggered"])
	v.L1InfoDeposit = parityL1InfoDepositCases(t, mainnet)
	v.L1InfoDecode = parityL1InfoDecodeCases(t)
	v.DepositLog = parityDepositLogCases(t, v.RollupConfigs["staggered"].DepositContractAddress)
	v.SystemConfigLog = paritySystemConfigLogCases(v.RollupConfigs["staggered"].L1SystemConfigAddress)
	v.SystemConfigFromBlock = paritySystemConfigFromBlockCases(t, v.RollupConfigs["staggered"], mainnet)
	return v
}

// parityEcotoneSystemConfig is a system config with a value for every field derivation reads
// after Ecotone.
func parityEcotoneSystemConfig() eth.SystemConfig {
	return eth.SystemConfig{
		BatcherAddr:          parityBatcher,
		Scalar:               eth.EncodeScalar(eth.EcotoneScalars{BlobBaseFeeScalar: 810_949, BaseFeeScalar: 1_368}),
		GasLimit:             30_000_000,
		EIP1559Params:        eth.Bytes8{0, 0, 0, 250, 0, 0, 0, 6},
		OperatorFeeParams:    eth.Bytes32{23: 7, 31: 11},
		MinBaseFee:           1_000_000,
		DAFootprintGasScalar: 400,
	}
}

func parityChannelCases(t *testing.T, cfg *rollup.Config) []parityCase[parityChannelInput, parityChannelOutput] {
	signer := types.LatestSignerForChainID(cfg.L2ChainID)
	dynamicFee := func(nonce uint64) *types.Transaction {
		return types.MustSignNewTx(parityKey, signer, &types.DynamicFeeTx{
			ChainID: cfg.L2ChainID, Nonce: nonce, GasTipCap: big.NewInt(1e9), GasFeeCap: big.NewInt(2e9),
			Gas: 21_000, To: &parityRecipient, Value: big.NewInt(1),
		})
	}
	legacy := types.MustSignNewTx(parityKey, signer, &types.LegacyTx{
		Nonce: 10, GasPrice: big.NewInt(1e9), Gas: 21_000, To: &parityRecipient, Value: big.NewInt(1),
	})
	accessList := types.MustSignNewTx(parityKey, signer, &types.AccessListTx{
		ChainID: cfg.L2ChainID, Nonce: 11, GasPrice: big.NewInt(1e9), Gas: 30_000, To: &parityRecipient,
		AccessList: types.AccessList{{Address: parityRecipient, StorageKeys: []common.Hash{{0x01}}}},
	})
	auth, err := types.SignSetCode(parityKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(cfg.L2ChainID), Address: parityRecipient, Nonce: 13,
	})
	require.NoError(t, err)
	setCode := types.MustSignNewTx(parityKey, signer, &types.SetCodeTx{
		ChainID: uint256.MustFromBig(cfg.L2ChainID), Nonce: 12, GasTipCap: uint256.NewInt(1e9),
		GasFeeCap: uint256.NewInt(2e9), Gas: 60_000, To: parityRecipient, Value: uint256.NewInt(1),
		AuthList: []types.SetCodeAuthorization{auth},
	})

	singular := func(timestamp uint64, txs ...*types.Transaction) *SingularBatch {
		batch := &SingularBatch{ParentHash: parityParentHash, EpochNum: 7, EpochHash: parityEpochHash, Timestamp: timestamp}
		for _, tx := range txs {
			data, err := tx.MarshalBinary()
			require.NoError(t, err)
			batch.Transactions = append(batch.Transactions, data)
		}
		return batch
	}
	span := func(batches ...*SingularBatch) *RawSpanBatch {
		spanBatch := NewSpanBatch(cfg.Genesis.L2Time, cfg.L2ChainID)
		for i, batch := range batches {
			require.NoError(t, spanBatch.AppendSingularBatch(batch, uint64(i)))
		}
		raw, err := spanBatch.ToRawSpanBatch()
		require.NoError(t, err)
		return raw
	}
	encode := func(inner InnerBatchData) []byte {
		data, err := NewBatchData(inner).MarshalBinary()
		require.NoError(t, err)
		return data
	}
	// withTxData replaces the encoded data of the span batch's first transaction, leaving its
	// signature, nonce, gas and recipient in place.
	withTxData := func(raw *RawSpanBatch, rewrite func(*spanBatchTx)) []byte {
		var tx spanBatchTx
		require.NoError(t, tx.UnmarshalBinary(raw.txs.txDatas[0]))
		rewrite(&tx)
		data, err := tx.MarshalBinary()
		require.NoError(t, err)
		raw.txs.txDatas[0] = data
		return encode(raw)
	}

	deposit, err := (&optypes.DepositTx{
		SourceHash: common.Hash{0x01}, From: parityRecipient, To: &parityRecipient,
		Mint: big.NewInt(1e18), Value: big.NewInt(0), Gas: 21_000,
	}).MarshalBinary()
	require.NoError(t, err)
	depositSpan := span(singular(1600, dynamicFee(0)))
	depositSpan.txs.txDatas[0] = deposit

	validSingular := encode(singular(1602, dynamicFee(1)))
	brotliChannel, brotliEnds := parityBrotliChannel(t, encode(singular(3500, dynamicFee(0))), encode(singular(3502, dynamicFee(1))))
	// op-node's brotli reader hands on the output of earlier meta-blocks before it reports a format
	// error, except for roughly the last 4 KiB, and only once the decoder has had to flush them. A
	// large incompressible second batch makes it flush the first batch before the error.
	incompressible := testutils.RandomData(rand.New(rand.NewSource(1)), 70_000)
	largeCalldata := types.MustSignNewTx(parityKey, signer, &types.DynamicFeeTx{
		ChainID: cfg.L2ChainID, Nonce: 20, GasTipCap: big.NewInt(1e9), GasFeeCap: big.NewInt(2e9),
		Gas: 2_000_000, To: &parityRecipient, Data: incompressible,
	})
	largeBrotliChannel, largeBrotliEnds := parityBrotliChannel(t, encode(singular(3500, dynamicFee(0))), encode(singular(3502, largeCalldata)))
	zlibChannel, zlibEnds := parityZlibChannelWithBoundaries(t, encode(singular(1600, dynamicFee(0))), encode(singular(1602, dynamicFee(1))))
	// After a flush the brotli stream sits at a meta-block boundary. This header byte sets the
	// reserved bit of a metadata block, which a brotli decoder must reject.
	brotliFormatError := []byte{0x0e, 0x00, 0x00, 0x00}

	channel := func(name string, originTime uint64, data []byte) parityCase[parityChannelInput, parityChannelOutput] {
		return parityCase[parityChannelInput, parityChannelOutput]{
			Name:  name,
			Input: parityChannelInput{RollupConfig: "staggered", OriginTime: originTime, Channel: data},
		}
	}
	return []parityCase[parityChannelInput, parityChannelOutput]{
		channel("zlib singular batches", parityDeltaOrigin,
			parityZlibChannel(t, encode(singular(1600, dynamicFee(0))), encode(singular(1602, legacy, accessList)))),
		channel("zlib span batch with every transaction type", parityDeltaOrigin,
			parityZlibChannel(t, encode(span(singular(1600, dynamicFee(0), legacy), singular(1602, accessList, setCode))))),
		channel("span batch before Delta", parityPreDelta,
			parityZlibChannel(t, encode(span(singular(400, dynamicFee(0)))))),
		channel("brotli channel before Fjord", parityDeltaOrigin, brotliChannel),
		channel("brotli channel after Fjord", parityFjordOrigin, brotliChannel),
		channel("brotli channel with a format error right after the first batch", parityFjordOrigin,
			append(bytes.Clone(brotliChannel[:brotliEnds[0]]), brotliFormatError...)),
		channel("brotli channel with a format error after a large second batch", parityFjordOrigin,
			append(bytes.Clone(largeBrotliChannel[:largeBrotliEnds[1]]), brotliFormatError...)),
		channel("brotli channel cut off after the first batch", parityFjordOrigin, brotliChannel[:brotliEnds[0]]),
		channel("zlib channel cut off after the first batch", parityDeltaOrigin, zlibChannel[:zlibEnds[0]]),
		channel("span batch with a deposit transaction, then a valid batch", parityFjordOrigin,
			parityZlibChannel(t, encode(depositSpan), validSingular)),
		channel("span batch with a deposit transaction, then a valid batch, before Holocene", parityDeltaOrigin,
			parityZlibChannel(t, encode(depositSpan), validSingular)),
		channel("unknown batch type, then a valid batch", parityDeltaOrigin,
			parityZlibChannel(t, []byte{0x03, 0xc0}, validSingular)),
		channel("malformed singular batch, then a valid batch", parityDeltaOrigin,
			parityZlibChannel(t, []byte{SingularBatchType, 0xc1}, validSingular)),
		channel("empty batch after a valid batch", parityDeltaOrigin,
			parityZlibChannel(t, validSingular, []byte{})),
		channel("span batch with max fee per gas above 128 bits", parityDeltaOrigin,
			parityZlibChannel(t, withTxData(span(singular(1600, dynamicFee(0))), func(tx *spanBatchTx) {
				data := tx.inner.(*spanBatchDynamicFeeTxData)
				data.GasFeeCap = new(big.Int).Add(data.GasFeeCap, twoTo128)
			}))),
		channel("span batch with max priority fee per gas above 128 bits", parityDeltaOrigin,
			parityZlibChannel(t, withTxData(span(singular(1600, dynamicFee(0))), func(tx *spanBatchTx) {
				data := tx.inner.(*spanBatchDynamicFeeTxData)
				data.GasTipCap = new(big.Int).Add(data.GasTipCap, twoTo128)
			}))),
		channel("span batch with legacy gas price above 128 bits", parityDeltaOrigin,
			parityZlibChannel(t, withTxData(span(singular(1600, legacy)), func(tx *spanBatchTx) {
				data := tx.inner.(*spanBatchLegacyTxData)
				data.GasPrice = new(big.Int).Add(data.GasPrice, twoTo128)
			}))),
		channel("span batch with access-list gas price above 128 bits", parityDeltaOrigin,
			parityZlibChannel(t, withTxData(span(singular(1600, accessList)), func(tx *spanBatchTx) {
				data := tx.inner.(*spanBatchAccessListTxData)
				data.GasPrice = new(big.Int).Add(data.GasPrice, twoTo128)
			}))),
		channel("span batch with set-code max fee per gas above 128 bits", parityDeltaOrigin,
			parityZlibChannel(t, withTxData(span(singular(1600, setCode)), func(tx *spanBatchTx) {
				data := tx.inner.(*spanBatchSetCodeTxData)
				data.GasFeeCap = new(uint256.Int).Add(data.GasFeeCap, uint256.MustFromBig(twoTo128))
			}))),
	}
}

// parityZlibChannel compresses the batches, each an RLP string, into a zlib channel.
func parityZlibChannel(t *testing.T, batches ...[]byte) []byte {
	channel, _ := parityZlibChannelWithBoundaries(t, batches...)
	return channel
}

// parityZlibChannelWithBoundaries also returns, for each batch, the length of the prefix of the
// channel from which it decompresses completely.
func parityZlibChannelWithBoundaries(t *testing.T, batches ...[]byte) ([]byte, []int) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	ends := writeParityBatches(t, &buf, w, batches)
	return buf.Bytes(), ends
}

// parityBrotliChannel compresses the batches into a brotli channel, and returns for each batch
// the length of the prefix of the channel from which it decompresses completely.
func parityBrotliChannel(t *testing.T, batches ...[]byte) ([]byte, []int) {
	var buf bytes.Buffer
	buf.WriteByte(ChannelVersionBrotli)
	w := brotli.NewWriterLevel(&buf, 10)
	ends := writeParityBatches(t, &buf, w, batches)
	return buf.Bytes(), ends
}

func writeParityBatches(t *testing.T, buf *bytes.Buffer, w interface {
	io.Writer
	Flush() error
	Close() error
}, batches [][]byte) []int {
	ends := make([]int, len(batches))
	for i, batch := range batches {
		require.NoError(t, rlp.Encode(w, batch))
		require.NoError(t, w.Flush())
		ends[i] = buf.Len()
	}
	require.NoError(t, w.Close())
	return ends
}

func parityL1InfoDepositCases(t *testing.T, mainnet *params.ChainConfig) []parityCase[parityL1InfoDepositInput, parityBytesOutput] {
	preEcotone := parityRollupConfig().Genesis.SystemConfig
	ecotone := parityEcotoneSystemConfig()
	unknownScalarVersion := ecotone
	unknownScalarVersion.Scalar[0] = 2
	dirtyScalarPadding := ecotone
	dirtyScalarPadding.Scalar[5] = 1

	header := func(time uint64, excessBlobGas uint64) hexutil.Bytes {
		return parityL1Header(t, mainnet, time, excessBlobGas)
	}
	cancun, prague, osaka, bpo2 := *mainnet.CancunTime+12, *mainnet.PragueTime+12, *mainnet.OsakaTime+12, *mainnet.BPO2Time+12
	deposit := func(name, rollupCfg, l1Cfg string, sysCfg eth.SystemConfig, l1Header hexutil.Bytes, l2Time uint64) parityCase[parityL1InfoDepositInput, parityBytesOutput] {
		return parityCase[parityL1InfoDepositInput, parityBytesOutput]{
			Name: name,
			Input: parityL1InfoDepositInput{
				RollupConfig: rollupCfg, L1ChainConfig: l1Cfg, SystemConfig: sysCfg,
				SequenceNumber: 3, L1Header: l1Header, L2Timestamp: l2Time,
			},
		}
	}
	return []parityCase[parityL1InfoDepositInput, parityBytesOutput]{
		deposit("bedrock", "staggered", "mainnet", preEcotone, header(1_700_000_000, 0), parityPreDelta),
		deposit("first ecotone block", "staggered", "mainnet", preEcotone, header(cancun, 10_000_000), parityEcotoneTime),
		deposit("ecotone, L1 at cancun", "staggered", "mainnet", ecotone, header(cancun, 10_000_000), parityEcotoneBlock),
		deposit("ecotone, L1 at prague", "staggered", "mainnet", ecotone, header(prague, 10_000_000), parityEcotoneBlock),
		deposit("ecotone, L1 at osaka", "staggered", "mainnet", ecotone, header(osaka, 10_000_000), parityEcotoneBlock),
		deposit("ecotone, L1 at bpo2", "staggered", "mainnet", ecotone, header(bpo2, 10_000_000), parityEcotoneBlock),
		deposit("first isthmus block", "staggered", "mainnet", ecotone, header(bpo2, 10_000_000), parityIsthmusTime),
		deposit("isthmus", "staggered", "mainnet", ecotone, header(bpo2, 10_000_000), parityIsthmusBlock),
		deposit("first jovian block", "staggered", "mainnet", ecotone, header(bpo2, 10_000_000), parityJovianTime),
		deposit("jovian", "staggered", "mainnet", ecotone, header(bpo2, 10_000_000), parityJovianBlock),
		deposit("unknown ecotone scalar version", "staggered", "mainnet", unknownScalarVersion, header(prague, 10_000_000), parityEcotoneBlock),
		deposit("ecotone scalar with non-zero padding", "staggered", "mainnet", dirtyScalarPadding, header(prague, 10_000_000), parityEcotoneBlock),
		deposit("custom L1 blob schedule, L1 at cancun", "staggered", "custom-blob-schedule", ecotone, header(cancun, 50_000_000), parityEcotoneBlock),
		deposit("custom L1 blob schedule, L1 at prague", "staggered", "custom-blob-schedule", ecotone, header(prague, 50_000_000), parityEcotoneBlock),
		deposit("custom L1 blob schedule, L1 at osaka", "staggered", "custom-blob-schedule", ecotone, header(osaka, 50_000_000), parityEcotoneBlock),
		deposit("pectra blob schedule, L1 at prague", "late-pectra-blob-schedule", "mainnet", ecotone, header(prague, 50_000_000), parityEcotoneBlock),
		deposit("pectra blob schedule ending after L1 osaka", "late-pectra-blob-schedule", "mainnet", ecotone, header(osaka, 50_000_000), parityEcotoneBlock),
		deposit("after the pectra blob schedule ends", "late-pectra-blob-schedule", "mainnet", ecotone, header(bpo2, 50_000_000), parityEcotoneBlock),
	}
}

// parityL1Header returns the RLP encoding of an L1 header at the given time, with the fields
// mainnet's forks require at that time.
func parityL1Header(t *testing.T, l1Cfg *params.ChainConfig, time uint64, excessBlobGas uint64) hexutil.Bytes {
	number := big.NewInt(22_000_000)
	header := &types.Header{
		ParentHash:  common.Hash{0xaa},
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{0xcb},
		Root:        common.Hash{0xbb},
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Difficulty:  common.Big0,
		Number:      number,
		GasLimit:    36_000_000,
		GasUsed:     12_000_000,
		Time:        time,
		Extra:       []byte{},
		MixDigest:   common.Hash{0xdd},
		BaseFee:     big.NewInt(7_000_000_000),
	}
	if l1Cfg.IsShanghai(number, time) {
		header.WithdrawalsHash = &types.EmptyWithdrawalsHash
	}
	if l1Cfg.IsCancun(number, time) {
		header.BlobGasUsed = ptr.New(uint64(0))
		header.ExcessBlobGas = ptr.New(excessBlobGas)
		header.ParentBeaconRoot = &common.Hash{0xbe}
	}
	if l1Cfg.IsPrague(number, time) {
		header.RequestsHash = &types.EmptyRequestsHash
	}
	data, err := rlp.EncodeToBytes(header)
	require.NoError(t, err)
	return data
}

func parityL1InfoDecodeCases(t *testing.T) []parityCase[parityL1InfoDecodeInput, parityBytesOutput] {
	info := &L1BlockInfo{
		Number:               22_000_000,
		Time:                 1_746_612_323,
		BaseFee:              big.NewInt(7_000_000_000),
		BlockHash:            parityBlockHash,
		SequenceNumber:       2,
		BatcherAddr:          parityBatcher,
		L1FeeOverhead:        eth.Bytes32{31: 0xbc},
		L1FeeScalar:          eth.Bytes32{29: 0x0a, 30: 0x6f, 31: 0xe0},
		BlobBaseFee:          big.NewInt(12_345),
		BaseFeeScalar:        1_368,
		BlobBaseFeeScalar:    810_949,
		OperatorFeeScalar:    7,
		OperatorFeeConstant:  11,
		DAFootprintGasScalar: 400,
	}
	bedrock, err := info.marshalBinaryBedrock()
	require.NoError(t, err)
	ecotone, err := info.marshalBinaryEcotone()
	require.NoError(t, err)
	isthmus, err := info.marshalBinaryIsthmus()
	require.NoError(t, err)
	jovian, err := info.marshalBinaryJovian()
	require.NoError(t, err)

	// patched returns a copy of data with the byte at offset set to value.
	patched := func(data []byte, offset int, value byte) []byte {
		out := bytes.Clone(data)
		out[offset] = value
		return out
	}
	// Bedrock calldata is the selector and eight 32-byte words: number, time, base fee, block
	// hash, sequence number, batcher, overhead and scalar.
	const bedrockWord = 32
	// Ecotone calldata packs the fields after the selector: the scalars (8 bytes), sequence number
	// (8), timestamp (8), number (8), base fee (32), blob base fee (32), block hash (32) and the
	// batcher as a 32-byte word.
	const ecotoneBaseFee = 4 + 8 + 8 + 8 + 8
	const ecotoneBatcher = ecotoneBaseFee + 32 + 32 + 32

	decode := func(name string, l2Time uint64, calldata []byte) parityCase[parityL1InfoDecodeInput, parityBytesOutput] {
		return parityCase[parityL1InfoDecodeInput, parityBytesOutput]{
			Name:  name,
			Input: parityL1InfoDecodeInput{RollupConfig: "staggered", L2Timestamp: l2Time, Calldata: calldata},
		}
	}
	return []parityCase[parityL1InfoDecodeInput, parityBytesOutput]{
		decode("bedrock", parityPreDelta, bedrock),
		decode("first ecotone block", parityEcotoneTime, bedrock),
		decode("ecotone", parityEcotoneBlock, ecotone),
		decode("first isthmus block", parityIsthmusTime, ecotone),
		decode("isthmus", parityIsthmusBlock, isthmus),
		decode("first jovian block", parityJovianTime, isthmus),
		decode("jovian", parityJovianBlock, jovian),
		decode("bedrock calldata in an ecotone block", parityEcotoneBlock, bedrock),
		decode("ecotone calldata in a bedrock block", parityPreDelta, ecotone),
		decode("ecotone calldata in an isthmus block", parityIsthmusBlock, ecotone),
		decode("isthmus calldata in a jovian block", parityJovianBlock, isthmus),
		decode("jovian calldata in an ecotone block", parityEcotoneBlock, jovian),
		decode("bedrock number with non-zero padding", parityPreDelta, patched(bedrock, 4, 1)),
		decode("bedrock batcher with non-zero padding", parityPreDelta, patched(bedrock, 4+5*bedrockWord, 1)),
		decode("bedrock base fee above 64 bits", parityPreDelta, patched(bedrock, 4+2*bedrockWord+8, 1)),
		decode("ecotone batcher with non-zero padding", parityEcotoneBlock, patched(ecotone, ecotoneBatcher, 1)),
		decode("ecotone base fee above 64 bits", parityEcotoneBlock, patched(ecotone, ecotoneBaseFee+8, 1)),
		decode("ecotone blob base fee above 128 bits", parityEcotoneBlock, patched(ecotone, ecotoneBaseFee+32+8, 1)),
		decode("ecotone cut short", parityEcotoneBlock, ecotone[:len(ecotone)-1]),
		decode("ecotone with a trailing byte", parityEcotoneBlock, append(bytes.Clone(ecotone), 0)),
		decode("jovian with a trailing byte", parityJovianBlock, append(bytes.Clone(jovian), 0)),
	}
}

func parityDepositLogCases(t *testing.T, depositContract common.Address) []parityCase[parityDepositLogInput, parityBytesOutput] {
	marshal := func(deposit *optypes.DepositTx) parityLog {
		ev, err := MarshalDepositLogEvent(depositContract, deposit)
		require.NoError(t, err)
		return parityLog{Address: ev.Address, Topics: ev.Topics, Data: ev.Data}
	}
	transfer := marshal(&optypes.DepositTx{
		From: parityBatcher, To: &parityRecipient, Mint: big.NewInt(1e18), Value: big.NewInt(2), Gas: 100_000,
		Data: []byte{1, 2, 3},
	})
	creation := marshal(&optypes.DepositTx{
		From: parityBatcher, Mint: big.NewInt(0), Value: big.NewInt(0), Gas: 500_000, Data: bytes.Repeat([]byte{0x60}, 70),
	})

	withData := func(log parityLog, data []byte) parityLog {
		log.Data = data
		return log
	}
	withTopic := func(log parityLog, index int, topic common.Hash) parityLog {
		log.Topics = append([]common.Hash(nil), log.Topics...)
		log.Topics[index] = topic
		return log
	}
	dirtyTopic := func(topic common.Hash) common.Hash {
		topic[0] = 1
		return topic
	}
	dirtyPadding := bytes.Clone(transfer.Data)
	dirtyPadding[len(dirtyPadding)-1] = 1

	deposit := func(name string, log parityLog) parityCase[parityDepositLogInput, parityBytesOutput] {
		return parityCase[parityDepositLogInput, parityBytesOutput]{
			Name:  name,
			Input: parityDepositLogInput{Log: log, BlockHash: parityBlockHash, LogIndex: 4},
		}
	}
	return []parityCase[parityDepositLogInput, parityBytesOutput]{
		deposit("value transfer", transfer),
		deposit("contract creation", creation),
		deposit("non-zero padding after the opaque data", withData(transfer, dirtyPadding)),
		deposit("extra word after the opaque data", withData(transfer, append(bytes.Clone(transfer.Data), make([]byte, 32)...))),
		deposit("opaque data cut short", withData(transfer, transfer.Data[:len(transfer.Data)-32])),
		deposit("from topic with non-zero padding", withTopic(transfer, 1, dirtyTopic(transfer.Topics[1]))),
		deposit("to topic with non-zero padding", withTopic(transfer, 2, dirtyTopic(transfer.Topics[2]))),
		deposit("unknown version", withTopic(transfer, 3, common.Hash{31: 1})),
		deposit("three topics", parityLog{Address: transfer.Address, Topics: transfer.Topics[:3], Data: transfer.Data}),
	}
}

func paritySystemConfigLogCases(systemConfig common.Address) []parityCase[paritySystemConfigLogInput, paritySystemConfig] {
	start := parityEcotoneSystemConfig()
	word := func(value uint64) []byte {
		var w [32]byte
		binary.BigEndian.PutUint64(w[24:], value)
		return w[:]
	}
	// update is a ConfigUpdate log carrying payload as ABI-encoded bytes.
	update := func(updateType common.Hash, payload ...[]byte) parityLog {
		body := bytes.Join(payload, nil)
		data := append(append(word(32), word(uint64(len(body)))...), body...)
		return parityLog{
			Address: systemConfig,
			Topics:  []common.Hash{ConfigUpdateEventABIHash, ConfigUpdateEventVersion0, updateType},
			Data:    data,
		}
	}
	eip1559 := func(denominator, elasticity uint32, upper byte) []byte {
		w := make([]byte, 32)
		w[0] = upper
		binary.BigEndian.PutUint32(w[24:], denominator)
		binary.BigEndian.PutUint32(w[28:], elasticity)
		return w
	}
	newBatcher := common.LeftPadBytes(common.HexToAddress("0x00000000000000000000000000000000000b0b0b").Bytes(), 32)
	ecotoneScalar := eth.EncodeScalar(eth.EcotoneScalars{BlobBaseFeeScalar: 7, BaseFeeScalar: 9})
	unknownScalar := ecotoneScalar
	unknownScalar[0] = 2
	gasLimit := update(SystemConfigUpdateGasLimit, word(60_000_000))
	fourTopics := gasLimit
	fourTopics.Topics = append(append([]common.Hash(nil), gasLimit.Topics...), common.Hash{0x04})
	dirtyType := gasLimit
	dirtyType.Topics = []common.Hash{ConfigUpdateEventABIHash, ConfigUpdateEventVersion0, {0: 1, 31: 2}}
	unknownVersion := gasLimit
	unknownVersion.Topics = []common.Hash{ConfigUpdateEventABIHash, {31: 1}, SystemConfigUpdateGasLimit}

	apply := func(name string, l1Time uint64, log parityLog) parityCase[paritySystemConfigLogInput, paritySystemConfig] {
		return parityCase[paritySystemConfigLogInput, paritySystemConfig]{
			Name:  name,
			Input: paritySystemConfigLogInput{RollupConfig: "staggered", L1Time: l1Time, SystemConfig: start, Log: log},
		}
	}
	return []parityCase[paritySystemConfigLogInput, paritySystemConfig]{
		apply("batcher", parityEcotoneBlock, update(SystemConfigUpdateBatcher, newBatcher)),
		apply("fee scalars before ecotone", parityPreDelta, update(SystemConfigUpdateFeeScalars, word(188), word(684_000))),
		apply("fee scalars after ecotone", parityEcotoneBlock, update(SystemConfigUpdateFeeScalars, word(0), ecotoneScalar[:])),
		apply("unknown fee scalar version after ecotone", parityEcotoneBlock, update(SystemConfigUpdateFeeScalars, word(0), unknownScalar[:])),
		apply("gas limit", parityEcotoneBlock, gasLimit),
		apply("eip1559 params", parityEcotoneBlock, update(SystemConfigUpdateEIP1559Params, eip1559(100, 4, 0))),
		apply("eip1559 params with a zero denominator", parityEcotoneBlock, update(SystemConfigUpdateEIP1559Params, eip1559(0, 4, 0))),
		apply("eip1559 params with a zero elasticity", parityEcotoneBlock, update(SystemConfigUpdateEIP1559Params, eip1559(100, 0, 0))),
		apply("eip1559 params with non-zero padding", parityEcotoneBlock, update(SystemConfigUpdateEIP1559Params, eip1559(100, 4, 1))),
		apply("operator fee params", parityEcotoneBlock, update(SystemConfigUpdateOperatorFeeParams, (&eth.Bytes32{23: 5, 31: 3})[:])),
		apply("min base fee", parityEcotoneBlock, update(SystemConfigUpdateMinBaseFee, word(2_000_000))),
		apply("da footprint gas scalar", parityEcotoneBlock, update(SystemConfigUpdateDAFootprintGasScalar, word(800))),
		apply("da footprint gas scalar set to zero", parityEcotoneBlock, update(SystemConfigUpdateDAFootprintGasScalar, word(0))),
		apply("unsafe block signer", parityEcotoneBlock, update(SystemConfigUpdateUnsafeBlockSigner, newBatcher)),
		apply("four topics", parityEcotoneBlock, fourTopics),
		apply("update type with non-zero high bytes", parityEcotoneBlock, dirtyType),
		apply("unknown version", parityEcotoneBlock, unknownVersion),
		apply("unknown update type", parityEcotoneBlock, update(common.Hash{31: 99}, word(1))),
		apply("gas limit with a trailing word", parityEcotoneBlock, update(SystemConfigUpdateGasLimit, word(60_000_000), word(0))),
	}
}

func paritySystemConfigFromBlockCases(t *testing.T, cfg *rollup.Config, mainnet *params.ChainConfig) []parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput] {
	var header types.Header
	require.NoError(t, rlp.DecodeBytes(parityL1Header(t, mainnet, *mainnet.BPO2Time+12, 10_000_000), &header))
	l1Info := func(sysCfg eth.SystemConfig, l2Time uint64) hexutil.Bytes {
		data, err := L1InfoDepositBytes(cfg, mainnet, sysCfg, 3, eth.HeaderBlockInfo(&header), l2Time)
		require.NoError(t, err)
		return data
	}
	preEcotone, ecotone := cfg.Genesis.SystemConfig, parityEcotoneSystemConfig()
	holocene := eip1559.EncodeHoloceneExtraData(250, 6)
	jovian := eip1559.EncodeJovianExtraData(250, 6, 1_000_000)
	// The Karst activation block's gas limit carries the gas of the upgrade transactions on top of
	// the system config's.
	karstUpgradeGas, err := UpgradeGas(forks.Karst)
	require.NoError(t, err)

	blockWithGasLimit := func(name string, timestamp, gasLimit uint64, extraData []byte, l1InfoTx hexutil.Bytes) parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput] {
		return parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput]{
			Name: name,
			Input: paritySystemConfigFromBlockInput{
				RollupConfig: "staggered", Number: 100, Timestamp: timestamp, GasLimit: gasLimit,
				ExtraData: extraData, L1InfoTx: l1InfoTx,
			},
		}
	}
	block := func(name string, timestamp uint64, extraData []byte, l1InfoTx hexutil.Bytes) parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput] {
		return blockWithGasLimit(name, timestamp, 30_000_000, extraData, l1InfoTx)
	}
	return []parityCase[paritySystemConfigFromBlockInput, paritySystemConfigOutput]{
		block("bedrock block", parityPreDelta, nil, l1Info(preEcotone, parityPreDelta)),
		block("ecotone block", parityEcotoneBlock, nil, l1Info(ecotone, parityEcotoneBlock)),
		block("holocene block", parityHoloceneBlock, holocene, l1Info(ecotone, parityHoloceneBlock)),
		block("first isthmus block", parityIsthmusTime, holocene, l1Info(ecotone, parityIsthmusTime)),
		block("isthmus block", parityIsthmusBlock, holocene, l1Info(ecotone, parityIsthmusBlock)),
		block("first jovian block", parityJovianTime, jovian, l1Info(ecotone, parityJovianTime)),
		block("jovian block", parityJovianBlock, jovian, l1Info(ecotone, parityJovianBlock)),
		blockWithGasLimit("karst activation block", parityKarstTime, 30_000_000+karstUpgradeGas, jovian, l1Info(ecotone, parityKarstTime)),
		block("karst block", parityKarstTime+2, jovian, l1Info(ecotone, parityKarstTime+2)),
		block("pre-holocene block with extra data", parityEcotoneBlock, []byte{0, 1}, l1Info(ecotone, parityEcotoneBlock)),
		block("holocene block without extra data", parityHoloceneBlock, nil, l1Info(ecotone, parityHoloceneBlock)),
		block("jovian block with holocene extra data", parityJovianBlock, holocene, l1Info(ecotone, parityJovianBlock)),
		block("ecotone block with a bedrock L1-info deposit", parityEcotoneBlock, nil, l1Info(preEcotone, parityPreDelta)),
	}
}
