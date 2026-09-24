package projection_test

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection/sp1groth16"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

var updateVectors = flag.Bool("update-projection-vectors", false, "regenerate shared Go/Kona projection admission vectors")

type block struct {
	Timestamp    uint64          `json:"timestamp"`
	Epoch        uint64          `json:"epoch"`
	Transactions []hexutil.Bytes `json:"transactions"`
}
type span []block

func (s span) GetBlockCount() int                         { return len(s) }
func (s span) GetBlockTimestamp(i int) uint64             { return s[i].Timestamp }
func (s span) GetBlockEpochNum(i int) uint64              { return s[i].Epoch }
func (s span) GetBlockTransactions(i int) []hexutil.Bytes { return s[i].Transactions }

func (s span) clone() span {
	out := make(span, len(s))
	for i, b := range s {
		out[i] = block{b.Timestamp, b.Epoch, append([]hexutil.Bytes(nil), b.Transactions...)}
	}
	return out
}

// Schema v2 (§F.5): every vector carries its own consensus config and derivation context.
type vectorAnchor struct {
	Number uint64      `json:"number"`
	Hash   common.Hash `json:"hash"`
}
type vectorContinuation struct {
	Anchor       vectorAnchor `json:"anchor"`
	OutputRoot   common.Hash  `json:"output_root"`
	RecoveryHash common.Hash  `json:"recovery_hash"`
}
type vectorContext struct {
	ChainID       uint64             `json:"chain_id"`
	GenesisNumber uint64             `json:"genesis_number"`
	GenesisHash   common.Hash        `json:"genesis_hash"`
	GenesisTime   uint64             `json:"genesis_time"`
	BlockTime     uint64             `json:"block_time"`
	ParentHash    common.Hash        `json:"parent_hash"`
	L1Head        common.Hash        `json:"l1_head"`
	Continuation  vectorContinuation `json:"continuation"`
}

func toVectorContext(c projection.Context) vectorContext {
	return vectorContext{
		ChainID: bigs.Uint64Strict(c.ChainID), GenesisNumber: c.GenesisNumber, GenesisHash: c.GenesisHash, GenesisTime: c.GenesisTime,
		BlockTime: c.BlockTime, ParentHash: c.ParentHash, L1Head: c.L1Head,
		Continuation: vectorContinuation{
			Anchor:     vectorAnchor{c.Continuation.Anchor.Number, c.Continuation.Anchor.Hash},
			OutputRoot: c.Continuation.OutputRoot, RecoveryHash: c.Continuation.RecoveryHash,
		},
	}
}

func (c vectorContext) context() projection.Context {
	return projection.Context{
		ChainID: new(big.Int).SetUint64(c.ChainID), GenesisNumber: c.GenesisNumber, GenesisTime: c.GenesisTime, BlockTime: c.BlockTime,
		GenesisHash: c.GenesisHash, ParentHash: c.ParentHash, L1Head: c.L1Head,
		Continuation: projection.Continuation{
			Anchor:     eth.BlockID{Number: c.Continuation.Anchor.Number, Hash: c.Continuation.Anchor.Hash},
			OutputRoot: c.Continuation.OutputRoot, RecoveryHash: c.Continuation.RecoveryHash,
		},
	}
}

type vector struct {
	Name    string            `json:"name"`
	Config  projection.Config `json:"config"`
	Context vectorContext     `json:"context"`
	Blocks  span              `json:"blocks"`
	Accept  bool              `json:"accept"`
	// Digest is the records root (unchanged meaning). The remaining fields are set for accept
	// cases only.
	Digest         common.Hash   `json:"digest"`
	OutputsRoot    *common.Hash  `json:"outputs_root,omitempty"`
	MessagesRoot   *common.Hash  `json:"messages_root,omitempty"`
	TerminalOutput *common.Hash  `json:"terminal_output,omitempty"`
	ConfigHash     *common.Hash  `json:"config_hash,omitempty"`
	PublicValues   hexutil.Bytes `json:"public_values,omitempty"`
}

// Shared vector constants. Every vector uses chain 901 unless it tests the gate.
var (
	vectorDepSet            = []eth.ChainID{eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)}
	vectorProgramVKey       = common.HexToHash("0x0052a3c0f1d27e8b9a4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809101")
	vectorPrivateRollupJSON = []byte(`{"genesis":{"l2":{"number":0}},"block_time":2,"l2_chain_id":901}` + "\n")
	vectorL1ChainConfigJSON = []byte(`{"chainId":900,"blobSchedule":{"cancun":{"target":3,"max":6}}}` + "\n")
	vectorGenesisHash       = common.Hash{0x0c}
)

func mockConfig() projection.Config {
	return projection.Config{Verifier: projection.ExecutionMock, GenesisOutputRoot: common.Hash{9}, DependencySetHash: projection.DependencySetHash(vectorDepSet)}
}

func sp1Config(mock bool) projection.Config {
	return projection.Config{
		Verifier: projection.SP1PrivateProjectionV1, GenesisOutputRoot: common.Hash{9}, ProgramVKey: vectorProgramVKey,
		PrivateConfigHash: projection.PrivateConfigHash(vectorPrivateRollupJSON, vectorL1ChainConfigJSON),
		DependencySetHash: projection.DependencySetHash(vectorDepSet), MockProofs: mock,
	}
}

// context is the default admission context: normal mode, anchor = first-1 = 9, and l1Head = the
// hash of L1 block 6, the span's last epoch.
func context() projection.Context {
	return projection.Context{
		ChainID: big.NewInt(901), GenesisTime: 1000, BlockTime: 2, GenesisHash: vectorGenesisHash,
		ParentHash: common.Hash{1}, L1Head: common.Hash{6},
		Continuation: projection.Continuation{Anchor: eth.BlockID{Number: 9, Hash: common.Hash{1}}, OutputRoot: common.Hash{9}},
	}
}

// recoveryContext is recovery mode: anchor = first-3 with canonical recovery inputs.
func recoveryContext() projection.Context {
	ctx := context()
	ctx.Continuation = projection.Continuation{Anchor: eth.BlockID{Number: 7, Hash: common.Hash{7}}, OutputRoot: common.Hash{0x77}, RecoveryHash: common.Hash{0x55}}
	return ctx
}

func signed(t *testing.T, tx *types.DynamicFeeTx) hexutil.Bytes {
	t.Helper()
	key, err := crypto.HexToECDSA(strings.Repeat("11", 32))
	require.NoError(t, err)
	out, err := types.SignNewTx(key, types.LatestSignerForChainID(tx.ChainID), tx)
	require.NoError(t, err)
	raw, err := out.MarshalBinary()
	require.NoError(t, err)
	return raw
}

// spanBuilder produces signed projection transactions for one chain, config and context.
type spanBuilder struct {
	t   *testing.T
	cfg projection.Config
	ctx projection.Context
}

// tx signs a carrier with 500k gas, raised to the admission minimum (MinTxGas) for calldata-heavy
// carriers such as a maximal claim.
func (b spanBuilder) tx(to common.Address, data []byte, al types.AccessList) hexutil.Bytes {
	need, err := projection.MinTxGas(data, al)
	require.NoError(b.t, err)
	return signed(b.t, &types.DynamicFeeTx{ChainID: b.ctx.ChainID, Gas: max(500000, need), GasFeeCap: new(big.Int), GasTipCap: new(big.Int), Value: new(big.Int), To: &to, Data: data, AccessList: al})
}

// claimFields is the claim admission accepts for (cfg, ctx) over [first, last].
func (b spanBuilder) claimFields(first, last uint64, proof []byte) *codec.RangeClaim {
	c := b.ctx.Continuation
	parentOutput := c.OutputRoot
	if c.Anchor.Number+1 != first {
		parentOutput = common.Hash{0x66}
	}
	return &codec.RangeClaim{
		FirstBlock: first, LastBlock: last, AnchorBlock: c.Anchor.Number, AnchorOutputRoot: c.OutputRoot,
		RecoveryHash: c.RecoveryHash, ParentOutputRoot: parentOutput, Proof: proof,
		PrivateTerminalBlockHash: common.Hash{2}, PrivateTerminalParentHash: common.Hash{3},
		L1Head: b.ctx.L1Head, RollupConfigHash: projection.ConfigHash(&b.cfg, b.ctx),
		DepSetHash: b.cfg.DependencySetHash, PrivateDataHash: common.Hash{4},
	}
}

func (b spanBuilder) claimTx(c *codec.RangeClaim) hexutil.Bytes {
	data, err := wire.EncodePostClaim(c)
	require.NoError(b.t, err)
	return b.tx(predeploys.ClaimRegistryAddr, data, nil)
}

func (b spanBuilder) claim(first, last uint64, proof []byte) hexutil.Bytes {
	return b.claimTx(b.claimFields(first, last, proof))
}

func (b spanBuilder) export() hexutil.Bytes {
	data, err := wire.EncodeReplaySentMessage(&wire.SentMessage{Destination: big.NewInt(902), Nonce: big.NewInt(1), Sender: common.Address{1}, Target: common.Address{2}, Message: []byte("hello")})
	require.NoError(b.t, err)
	return b.tx(predeploys.L2toL2CrossDomainMessengerAddr, data, nil)
}

func (b spanBuilder) imported(wide bool) hexutil.Bytes {
	chain := big.NewInt(902)
	if wide {
		chain.Lsh(chain, 80)
	}
	trigger := &txintent.ExecTrigger{Msg: messages.Message{Identifier: messages.Identifier{Origin: common.Address{9}, BlockNumber: 4, LogIndex: 2, Timestamp: 1002, ChainID: eth.ChainIDFromBig(chain)}, PayloadHash: common.Hash{7}}}
	data, err := trigger.EncodeInput()
	require.NoError(b.t, err)
	al, err := trigger.AccessList()
	require.NoError(b.t, err)
	return b.tx(predeploys.CrossL2InboxAddr, data, al)
}

// base is the three-block span: claim; nothing; export and import. withOutputs inserts one
// recordOutput per block (after the claim in block 0), with root {20+i}.
func (b spanBuilder) base(proof []byte) span {
	return span{{1020, 5, []hexutil.Bytes{b.claim(10, 12, proof)}}, {1022, 5, nil}, {1024, 6, []hexutil.Bytes{b.export(), b.imported(false)}}}
}

func (b spanBuilder) withOutputs(s span) span {
	for i := range s {
		at := 0
		if i == 0 && len(s[i].Transactions) > 0 {
			at = 1
		}
		txs := append([]hexutil.Bytes(nil), s[i].Transactions[:at]...)
		txs = append(txs, b.tx(predeploys.ClaimRegistryAddr, wire.EncodeOutput(common.Hash{byte(i + 20)}), nil))
		s[i].Transactions = append(txs, s[i].Transactions[at:]...)
	}
	return s
}

func defaultBuilder(t *testing.T) spanBuilder { return spanBuilder{t, mockConfig(), context()} }

// Helpers over the default builder (chain 901, execution-mock config, normal-mode context).
func transaction(t *testing.T, to common.Address, data []byte, al types.AccessList) hexutil.Bytes {
	return defaultBuilder(t).tx(to, data, al)
}
func claim(t *testing.T, first, last uint64, proof []byte) hexutil.Bytes {
	return defaultBuilder(t).claim(first, last, proof)
}
func export(t *testing.T) hexutil.Bytes { return defaultBuilder(t).export() }
func imported(t *testing.T, wide bool) hexutil.Bytes {
	return defaultBuilder(t).imported(wide)
}
func mutate(t *testing.T, raw hexutil.Bytes, f func(*types.DynamicFeeTx)) hexutil.Bytes {
	var tx types.Transaction
	require.NoError(t, tx.UnmarshalBinary(raw))
	fields := &types.DynamicFeeTx{ChainID: tx.ChainId(), Nonce: tx.Nonce(), Gas: tx.Gas(), GasFeeCap: tx.GasFeeCap(), GasTipCap: tx.GasTipCap(), To: tx.To(), Value: tx.Value(), Data: tx.Data(), AccessList: tx.AccessList()}
	f(fields)
	return signed(t, fields)
}

// mutateClaim re-encodes the opening claim of s with f applied.
func mutateClaim(t *testing.T, s span, f func(*codec.RangeClaim)) {
	s[0].Transactions[0] = mutate(t, s[0].Transactions[0], func(tx *types.DynamicFeeTx) {
		c, err := wire.DecodeClaim(tx.Data)
		require.NoError(t, err)
		f(c)
		tx.Data, err = wire.EncodePostClaim(c)
		require.NoError(t, err)
	})
}

func claimProof(t *testing.T, s span) []byte {
	if len(s) == 0 || len(s[0].Transactions) == 0 {
		return nil
	}
	var tx types.Transaction
	if tx.UnmarshalBinary(s[0].Transactions[0]) != nil {
		return nil
	}
	c, err := wire.DecodeClaim(tx.Data())
	if err != nil {
		return nil
	}
	return c.Proof
}

func newVector(name string, accept bool, b spanBuilder, blocks span) vector {
	return vector{Name: name, Accept: accept, Config: b.cfg, Context: toVectorContext(b.ctx), Blocks: blocks}
}

// dummyProof marks structural vectors whose proof slot is replaced, for accepted spans, by the
// valid execution-mock envelope, so that the vector is also accepted by its configured verifier.
var dummyProof = []byte("dummy")

func vectors(t *testing.T) []vector {
	var out []vector
	def := defaultBuilder(t)
	add := func(name string, accept bool, change func(*vector)) {
		v := newVector(name, accept, def, def.base(dummyProof))
		if change != nil {
			change(&v)
		}
		v.Blocks = spanBuilder{t, v.Config, v.Context.context()}.withOutputs(v.Blocks)
		out = append(out, v)
	}
	add("mixed", true, nil)
	add("empty_messages", true, func(v *vector) { v.Blocks[2].Transactions = nil })
	add("empty_proof", true, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 12, nil) })
	add("wide_chain_id", true, func(v *vector) { v.Blocks[2].Transactions[1] = imported(t, true) })
	add("missing_claim", false, func(v *vector) { v.Blocks[0].Transactions = nil })
	add("misplaced_claim", false, func(v *vector) {
		v.Blocks[0].Transactions = append([]hexutil.Bytes{export(t)}, v.Blocks[0].Transactions...)
	})
	add("duplicate_claim", false, func(v *vector) { v.Blocks[2].Transactions = append(v.Blocks[2].Transactions, claim(t, 10, 12, nil)) })
	add("wrong_first", false, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 9, 12, nil) })
	add("wrong_last", false, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 13, nil) })
	add("truncated_span", false, func(v *vector) { v.Blocks = v.Blocks[:2] })
	add("noncontiguous", false, func(v *vector) { v.Blocks[2].Timestamp++ })
	add("unknown_mode", false, func(v *vector) { v.Config.Verifier = "unknown" })
	add("max_proof", true, func(v *vector) { v.Blocks[0].Transactions[0] = claim(t, 10, 12, make([]byte, codec.MaxProofSize)) })
	changes := []struct {
		name string
		f    func(*types.DynamicFeeTx)
	}{
		{"creation", func(tx *types.DynamicFeeTx) { tx.To = nil }},
		{"value", func(tx *types.DynamicFeeTx) { tx.Value = big.NewInt(1) }},
		{"fee", func(tx *types.DynamicFeeTx) { tx.GasFeeCap = big.NewInt(1) }},
		{"tip", func(tx *types.DynamicFeeTx) { tx.GasTipCap = big.NewInt(1) }},
		{"gas_zero", func(tx *types.DynamicFeeTx) { tx.Gas = 0 }},
		{"gas_overflow", func(tx *types.DynamicFeeTx) { tx.Gas = projection.MaxTxGas + 1 }},
		{"destination", func(tx *types.DynamicFeeTx) { a := common.Address{9}; tx.To = &a }},
		{"selector", func(tx *types.DynamicFeeTx) { tx.Data[0] ^= 1 }},
		{"trailing_bytes", func(tx *types.DynamicFeeTx) { tx.Data = append(tx.Data, 0) }},
		{"dirty_address", func(tx *types.DynamicFeeTx) { tx.Data[4+64] = 1 }},
		{"bad_offset", func(tx *types.DynamicFeeTx) { tx.Data[4+159] = 0 }},
		{"extra_access_list", func(tx *types.DynamicFeeTx) { tx.AccessList = types.AccessList{{Address: common.Address{1}}} }},
	}
	for _, change := range changes {
		add("late_"+change.name, false, func(v *vector) { v.Blocks[2].Transactions[0] = mutate(t, v.Blocks[2].Transactions[0], change.f) })
	}
	// §E.1 gas rule: a carrier below max(intrinsic, EIP-7623 floor) is an invalid transaction the
	// EVM would not execute. The import (access list) is bound by its intrinsic gas, the export
	// (calldata only) by the floor; at exactly the minimum both are admitted.
	minGas := func(raw hexutil.Bytes) (need, intrinsic, floor uint64) {
		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(raw))
		need, err := projection.MinTxGas(tx.Data(), tx.AccessList())
		require.NoError(t, err)
		intrinsic, err = core.IntrinsicGas(tx.Data(), tx.AccessList(), nil, false, true, true, true)
		require.NoError(t, err)
		floor, err = core.FloorDataGas(tx.Data())
		require.NoError(t, err)
		return need, intrinsic, floor
	}
	add("late_gas_below_intrinsic", false, func(v *vector) {
		need, intrinsic, floor := minGas(v.Blocks[2].Transactions[1])
		require.Greater(t, intrinsic, floor, "the import is bound by its intrinsic gas")
		v.Blocks[2].Transactions[1] = mutate(t, v.Blocks[2].Transactions[1], func(tx *types.DynamicFeeTx) { tx.Gas = need - 1 })
	})
	add("late_gas_below_floor", false, func(v *vector) {
		need, intrinsic, floor := minGas(v.Blocks[2].Transactions[0])
		require.Greater(t, floor, intrinsic, "the export is bound by the calldata floor")
		v.Blocks[2].Transactions[0] = mutate(t, v.Blocks[2].Transactions[0], func(tx *types.DynamicFeeTx) { tx.Gas = need - 1 })
	})
	add("gas_at_minimum", true, func(v *vector) {
		for k := range v.Blocks[2].Transactions {
			need, _, _ := minGas(v.Blocks[2].Transactions[k])
			v.Blocks[2].Transactions[k] = mutate(t, v.Blocks[2].Transactions[k], func(tx *types.DynamicFeeTx) { tx.Gas = need })
		}
	})
	add("bad_import_checksum", false, func(v *vector) {
		v.Blocks[2].Transactions[1] = mutate(t, v.Blocks[2].Transactions[1], func(tx *types.DynamicFeeTx) { tx.AccessList[0].StorageKeys[1][5] ^= 1 })
	})
	add("duplicate_import_access", false, func(v *vector) {
		v.Blocks[2].Transactions[1] = mutate(t, v.Blocks[2].Transactions[1], func(tx *types.DynamicFeeTx) { tx.AccessList = append(tx.AccessList, tx.AccessList[0]) })
	})
	add("legacy_transaction", false, func(v *vector) {
		key, err := crypto.HexToECDSA(strings.Repeat("11", 32))
		require.NoError(t, err)
		a := predeploys.L2toL2CrossDomainMessengerAddr
		tx, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(901)), &types.LegacyTx{To: &a, Gas: 500000, GasPrice: new(big.Int), Value: new(big.Int)})
		require.NoError(t, err)
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		v.Blocks[2].Transactions[0] = raw
	})
	add("empty_transaction", false, func(v *vector) { v.Blocks[2].Transactions[0] = nil })
	for _, enabled := range []bool{false, true} {
		add(map[bool]string{false: "events_disabled", true: "events_enabled"}[enabled], enabled, func(v *vector) {
			v.Config.AllowEvents = enabled
			v.Blocks[0].Transactions[0] = spanBuilder{t, v.Config, v.Context.context()}.claim(10, 12, dummyProof)
			data, err := wire.EncodeReplayEvent([]common.Hash{{4}}, []byte("event"))
			require.NoError(t, err)
			// The event consumes rendered index 0 without a message leaf; the import is index 1.
			v.Blocks[2].Transactions[0] = transaction(t, predeploys.EventReplayerAddr, data, nil)
		})
	}
	// Recovery mode (§C.6): admission requires canonical recovery inputs; the parent output is
	// bound only by the relation (public-values word 12).
	addRecovery := func(name string, accept bool, ctx projection.Context, f func(*codec.RangeClaim)) {
		b := spanBuilder{t, mockConfig(), ctx}
		blocks := b.base(dummyProof)
		c := spanBuilder{t, mockConfig(), recoveryContext()}.claimFields(10, 12, dummyProof)
		if f != nil {
			f(c)
		}
		blocks[0].Transactions[0] = b.claimTx(c)
		out = append(out, newVector(name, accept, b, b.withOutputs(blocks)))
	}
	addRecovery("recovery_accept", true, recoveryContext(), nil)
	missing := recoveryContext()
	missing.Continuation.RecoveryHash = common.Hash{}
	addRecovery("recovery_missing_inputs", false, missing, func(c *codec.RangeClaim) { c.RecoveryHash = common.Hash{} })
	addRecovery("recovery_wrong_anchor_root", false, recoveryContext(), func(c *codec.RangeClaim) { c.AnchorOutputRoot[0] ^= 1 })
	// The insecure verifiers are test-gated: chain 10 is rejected even with every transaction and
	// the claim built correctly for chain 10.
	{
		cfg := mockConfig()
		cfg.Verifier = projection.InsecureStub
		ctx := context()
		ctx.ChainID = big.NewInt(10)
		b := spanBuilder{t, cfg, ctx}
		out = append(out, newVector("retired_stub_ungated", false, b, b.withOutputs(b.base(dummyProof))))
	}
	// The production profile: structurally admissible under the explicit stub verifier.
	{
		b := spanBuilder{t, sp1Config(false), context()}
		out = append(out, newVector("sp1_accept", true, b, b.withOutputs(b.base(nil))))
		cfg := sp1Config(false)
		cfg.AllowEvents = true
		b = spanBuilder{t, cfg, context()}
		out = append(out, newVector("sp1_allow_events_rejected", false, b, b.withOutputs(b.base(nil))))
	}
	for _, name := range []string{"missing_first_output", "missing_later_output", "duplicate_output", "late_output", "zero_output", "trailing_output", "wrong_anchor", "wrong_anchor_root", "wrong_parent_output", "unexpected_recovery", "wrong_l1_head", "wrong_rollup_config_hash", "wrong_dep_set_hash"} {
		raw, err := json.Marshal(out[0])
		require.NoError(t, err)
		var v vector
		require.NoError(t, json.Unmarshal(raw, &v))
		v.Name, v.Accept = name, false
		switch name {
		case "missing_first_output":
			v.Blocks[0].Transactions = v.Blocks[0].Transactions[:1]
		case "missing_later_output":
			v.Blocks[1].Transactions = nil
		case "duplicate_output":
			v.Blocks[1].Transactions = append(v.Blocks[1].Transactions, v.Blocks[1].Transactions[0])
		case "late_output":
			v.Blocks[2].Transactions[0], v.Blocks[2].Transactions[1] = v.Blocks[2].Transactions[1], v.Blocks[2].Transactions[0]
		case "zero_output":
			v.Blocks[1].Transactions[0] = transaction(t, predeploys.ClaimRegistryAddr, wire.EncodeOutput(common.Hash{}), nil)
		case "trailing_output":
			v.Blocks[1].Transactions[0] = mutate(t, v.Blocks[1].Transactions[0], func(tx *types.DynamicFeeTx) { tx.Data = append(tx.Data, 0) })
		default:
			mutateClaim(t, v.Blocks, func(c *codec.RangeClaim) {
				switch name {
				case "wrong_anchor":
					c.AnchorBlock--
				case "wrong_anchor_root":
					c.AnchorOutputRoot[0]++
				case "wrong_parent_output":
					c.ParentOutputRoot[0]++
				case "unexpected_recovery":
					c.RecoveryHash[0]++
				case "wrong_l1_head":
					c.L1Head = common.Hash{5} // canonical L1 block 5, not the span's last epoch 6
				case "wrong_rollup_config_hash":
					c.RollupConfigHash[31] ^= 1
				case "wrong_dep_set_hash":
					c.DepSetHash = projection.DependencySetHash(append(append([]eth.ChainID(nil), vectorDepSet...), eth.ChainIDFromUInt64(903)))
				}
			})
		}
		out = append(out, v)
	}
	finalize(t, out)
	return out
}

// finalize installs the execution-mock envelope in accepted structural vectors carrying the dummy
// proof, and records the admission outputs of every accepted vector.
func finalize(t *testing.T, vs []vector) {
	for i := range vs {
		v := &vs[i]
		if !v.Accept {
			continue
		}
		s, err := projection.ValidateProjectionRange(&v.Config, v.Context.context(), v.Blocks, projection.StubVerifier{})
		require.NoError(t, err, v.Name)
		if v.Config.Verifier == projection.ExecutionMock && string(claimProof(t, v.Blocks)) == string(dummyProof) {
			mutateClaim(t, v.Blocks, func(c *codec.RangeClaim) { c.Proof = projection.ExecutionMockProof(*s) })
		}
		record(v, s)
	}
}

func record(v *vector, s *projection.Statement) {
	cfgHash, pv := s.ProjectionConfigHash, projection.PublicValues(s)
	outputs, msgs, terminal := s.OutputsRoot, s.MessagesRoot, s.TerminalOutput
	v.Digest, v.OutputsRoot, v.MessagesRoot, v.TerminalOutput, v.ConfigHash, v.PublicValues = s.ProjectionHash, &outputs, &msgs, &terminal, &cfgHash, pv[:]
}

func loadVectors(t *testing.T, path string) []vector {
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var cases []vector
	require.NoError(t, json.Unmarshal(raw, &cases))
	return cases
}

func encodeVectors(t *testing.T, v any) []byte {
	raw, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	return append(raw, '\n')
}

// syncVectors writes generated under -update-projection-vectors and otherwise requires the file
// to equal it byte for byte.
func syncVectors(t *testing.T, path string, generated any) {
	want := encodeVectors(t, generated)
	if *updateVectors {
		require.NoError(t, os.MkdirAll("testdata", 0755))
		require.NoError(t, os.WriteFile(path, want, 0644))
	}
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got), "%s is stale; run go test with -update-projection-vectors", path)
}

// checkVector runs v through admission with its verifier and checks the recorded outputs and
// purity (no mutation of inputs, identical results twice).
func checkVector(t *testing.T, v vector, verifier func(*vector) (projection.ProofVerifier, error)) {
	before, err := json.Marshal(v)
	require.NoError(t, err)
	ctx := v.Context.context()
	var first *projection.Statement
	pv, err := verifier(&v)
	if err == nil {
		first, err = projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, pv)
		second, again := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, pv)
		require.Equal(t, first, second)
		require.Equal(t, err, again)
	}
	if v.Accept {
		require.NoError(t, err)
		want := v
		record(&want, first)
		require.Equal(t, want, v)
		require.Len(t, v.PublicValues, projection.PublicValuesLength)
		require.Equal(t, *v.TerminalOutput, common.BytesToHash(v.PublicValues[32*projection.WordTerminalOutput:]))
	} else {
		require.Error(t, err)
	}
	after, e := json.Marshal(v)
	require.NoError(t, e)
	require.Equal(t, before, after)
	require.Equal(t, v.Context.context(), ctx)
}

func stubVerifier(*vector) (projection.ProofVerifier, error) { return projection.StubVerifier{}, nil }
func configuredVerifier(v *vector) (projection.ProofVerifier, error) {
	return projection.VerifierFor(&v.Config, new(big.Int).SetUint64(v.Context.ChainID))
}

func TestProjectionVectors(t *testing.T) {
	cases := vectors(t)
	syncVectors(t, "testdata/ranges.json", cases)
	for _, v := range loadVectors(t, "testdata/ranges.json") {
		t.Run(v.Name, func(t *testing.T) {
			checkVector(t, v, stubVerifier)
			requireRejectReason(t, v, stubVerifier)
			if v.Accept {
				require.NotNil(t, v.OutputsRoot)
			} else {
				require.Nil(t, v.OutputsRoot)
				require.Empty(t, v.PublicValues)
			}
		})
	}
}

// Structural vectors carrying the execution-mock envelope are also accepted by their configured
// verifier; every other vector is rejected by it.
func TestProjectionVectorsUnderConfiguredVerifier(t *testing.T) {
	accepted := 0
	for _, v := range loadVectors(t, "testdata/ranges.json") {
		t.Run(v.Name, func(t *testing.T) {
			withMock := v.Config.Verifier == projection.ExecutionMock && strings.HasPrefix(string(claimProof(t, v.Blocks)), "optimism.private-execution.mock.v1")
			v.Accept = v.Accept && withMock
			if v.Accept {
				accepted++
			}
			checkVector(t, v, configuredVerifier)
		})
	}
	require.GreaterOrEqual(t, accepted, 5)
}

type rejectVerifier struct{}

func (rejectVerifier) Verify(projection.Statement, []byte) error {
	return errors.New("deliberate proof rejection")
}

type bindingVerifier struct {
	want  projection.Statement
	proof []byte
}

func (v bindingVerifier) Verify(got projection.Statement, proof []byte) error {
	a, _ := json.Marshal(v.want)
	b, _ := json.Marshal(got)
	if string(a) != string(b) || string(proof) != string(v.proof) {
		return errors.New("wrong statement or proof")
	}
	return nil
}
func TestProofGate(t *testing.T) {
	v := vectors(t)[0]
	statement, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	require.Empty(t, statement.Claim.Proof)
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, rejectVerifier{})
	require.ErrorContains(t, err, "deliberate proof rejection")
	checker := bindingVerifier{*statement, claimProof(t, v.Blocks)}
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, checker)
	require.NoError(t, err)
	v.Blocks[2].Transactions = v.Blocks[2].Transactions[:1]
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, checker)
	require.ErrorContains(t, err, "wrong statement")
}

func TestProofStatementBindsExecutionEnvelopeAndContext(t *testing.T) {
	v := vectors(t)[0]
	want, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	verifier := bindingVerifier{*want, claimProof(t, v.Blocks)}
	for _, field := range []string{"nonce", "gas", "parent"} {
		t.Run(field, func(t *testing.T) {
			candidate := vectors(t)[0]
			ctx := context()
			if field == "parent" {
				ctx.ParentHash[1]++
				ctx.Continuation.Anchor.Hash = ctx.ParentHash
			} else {
				candidate.Blocks[2].Transactions[0] = mutate(t, candidate.Blocks[2].Transactions[0], func(tx *types.DynamicFeeTx) {
					if field == "nonce" {
						tx.Nonce++
					} else {
						tx.Gas++
					}
				})
			}
			_, err := projection.ValidateProjectionRange(&candidate.Config, ctx, candidate.Blocks, verifier)
			require.ErrorContains(t, err, "wrong statement")
		})
	}
}

func TestProjectionContextBounds(t *testing.T) {
	for _, field := range []string{"zero_chain", "zero_interval", "height_overflow", "before_genesis", "empty_range", "zero_l1_head"} {
		t.Run(field, func(t *testing.T) {
			v, ctx := vectors(t)[0], context()
			switch field {
			case "zero_chain":
				ctx.ChainID = new(big.Int)
			case "zero_interval":
				ctx.BlockTime = 0
			case "height_overflow":
				ctx.GenesisNumber = ^uint64(0)
			case "before_genesis":
				ctx.GenesisTime = 1021
			case "empty_range":
				v.Blocks = nil
			case "zero_l1_head":
				ctx.L1Head = common.Hash{}
			}
			_, err := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
			require.Error(t, err)
		})
	}
}

// The claim's l1Head, rollupConfigHash and depSetHash are bound in every verifier mode (§C.4.3).
func TestClaimFieldBindings(t *testing.T) {
	for _, cfg := range []projection.Config{mockConfig(), sp1Config(false), sp1Config(true)} {
		b := spanBuilder{t, cfg, context()}
		for name, tc := range map[string]struct {
			f    func(*codec.RangeClaim)
			want string
		}{
			"l1_head":            {func(c *codec.RangeClaim) { c.L1Head = common.Hash{5} }, "claim l1Head does not match"},
			"rollup_config_hash": {func(c *codec.RangeClaim) { c.RollupConfigHash[0] ^= 1 }, "claim rollupConfigHash does not match"},
			"dep_set_hash":       {func(c *codec.RangeClaim) { c.DepSetHash[0] ^= 1 }, "claim depSetHash does not match"},
		} {
			blocks := b.withOutputs(b.base(nil))
			_, err := projection.ValidateProjectionRange(&cfg, b.ctx, blocks, projection.StubVerifier{})
			require.NoError(t, err)
			mutateClaim(t, blocks, tc.f)
			_, err = projection.ValidateProjectionRange(&cfg, b.ctx, blocks, projection.StubVerifier{})
			require.ErrorContains(t, err, tc.want, "%s under %s mock=%v", name, cfg.Verifier, cfg.MockProofs)
		}
	}
	// The config hash covers the genesis hash, so a node with another genesis rejects the claim.
	v := vectors(t)[0]
	ctx := context()
	ctx.GenesisHash[0] ^= 1
	_, err := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
	require.ErrorContains(t, err, "claim rollupConfigHash does not match")
}

// proofVectors are the verifier vectors: each claim carries an envelope and is admitted with
// VerifierFor(config, chain) (§F.3, §F.5).
func proofVectors(t *testing.T) []vector {
	var out []vector
	{
		b := defaultBuilder(t)
		base := b.withOutputs(b.base(nil))
		statement, err := projection.ValidateProjectionRange(&b.cfg, b.ctx, base, projection.StubVerifier{})
		require.NoError(t, err)
		good := projection.ExecutionMockProof(*statement)
		altered := append([]byte{}, good...)
		altered[len(altered)-1] ^= 1
		for _, tc := range []struct {
			name   string
			proof  []byte
			accept bool
		}{
			{"execution_mock_valid", good, true},
			{"execution_mock_missing", nil, false},
			{"execution_mock_legacy", []byte(projection.InsecureStub), false},
			{"execution_mock_truncated", good[:len(good)-1], false},
			{"execution_mock_trailing", append(append([]byte{}, good...), 0), false},
			{"execution_mock_altered", altered, false},
		} {
			blocks := base.clone()
			mutateClaim(t, blocks, func(c *codec.RangeClaim) { c.Proof = tc.proof })
			out = append(out, newVector(tc.name, tc.accept, b, blocks))
		}
	}
	type envelopeFn func(pv [projection.PublicValuesLength]byte) []byte
	build := func(name string, accept bool, cfg projection.Config, ctx projection.Context, f envelopeFn) {
		b := spanBuilder{t, cfg, ctx}
		blocks := b.withOutputs(b.base(nil))
		statement, err := projection.ValidateProjectionRange(&cfg, ctx, blocks, projection.StubVerifier{})
		if err != nil {
			// Only the ungated chain, which CheckChain rejects before any statement exists: carry
			// the allowlisted twin's public values. A conforming verifier never reaches them.
			require.Equal(t, uint64(10), bigs.Uint64Strict(ctx.ChainID), name)
			twin := spanBuilder{t, cfg, context()}
			statement, err = projection.ValidateProjectionRange(&cfg, twin.ctx, twin.withOutputs(twin.base(nil)), projection.StubVerifier{})
			require.NoError(t, err, name)
		}
		pv := projection.PublicValues(statement)
		mutateClaim(t, blocks, func(c *codec.RangeClaim) { c.Proof = f(pv) })
		out = append(out, newVector(name, accept, b, blocks))
	}
	mock := func(words func(*[5]common.Hash)) envelopeFn {
		return func(pv [projection.PublicValuesLength]byte) []byte {
			env, err := projection.DecodeEnvelope(projection.MockEnvelope(vectorProgramVKey, pv))
			require.NoError(t, err)
			if words != nil {
				var w [5]common.Hash
				for i := range w {
					w[i] = common.BytesToHash(env.Proof[32*i : 32*i+32])
				}
				words(&w)
				env.Proof = env.Proof[:0]
				for _, x := range w {
					env.Proof = append(env.Proof, x[:]...)
				}
			}
			return projection.EncodeEnvelope(env)
		}
	}
	ungated := context()
	ungated.ChainID = big.NewInt(10)
	build("sp1_mock_valid", true, sp1Config(true), context(), mock(nil))
	build("sp1_mock_disabled", false, sp1Config(false), context(), mock(nil))
	build("sp1_mock_ungated_chain", false, sp1Config(true), ungated, mock(nil))
	build("sp1_mock_wrong_vkey", false, sp1Config(true), context(), mock(func(w *[5]common.Hash) { w[0][31] ^= 1 }))
	build("sp1_mock_wrong_digest", false, sp1Config(true), context(), mock(func(w *[5]common.Hash) { w[1][31] ^= 1 }))
	build("sp1_mock_nonzero_exit", false, sp1Config(true), context(), mock(func(w *[5]common.Hash) { w[2][31] = 1 }))
	build("sp1_mock_nonzero_vk_root", false, sp1Config(true), context(), mock(func(w *[5]common.Hash) { w[3][31] = 1 }))
	build("sp1_mock_nonzero_nonce", false, sp1Config(true), context(), mock(func(w *[5]common.Hash) { w[4][31] = 1 }))
	for k := 0; k < projection.PublicValuesWords; k++ {
		build(fmt.Sprintf("sp1_pv_word_%d_flipped", k), false, sp1Config(true), context(), func(pv [projection.PublicValuesLength]byte) []byte {
			pv[32*k+31] ^= 1
			// The mock digest is recomputed over the flipped values: only the statement binding
			// rejects the envelope.
			return projection.MockEnvelope(vectorProgramVKey, pv)
		})
	}
	edit := func(f func([]byte) []byte) envelopeFn {
		return func(pv [projection.PublicValuesLength]byte) []byte { return f(mock(nil)(pv)) }
	}
	build("sp1_bad_version", false, sp1Config(true), context(), edit(func(e []byte) []byte { e[0] = 0x02; return e }))
	build("sp1_bad_kind", false, sp1Config(true), context(), edit(func(e []byte) []byte { e[1] = 0x03; return e }))
	build("sp1_trailing_byte", false, sp1Config(true), context(), edit(func(e []byte) []byte { return append(e, 0) }))
	build("sp1_truncated", false, sp1Config(true), context(), edit(func(e []byte) []byte { return e[:len(e)-1] }))
	build("sp1_groth16_len_355", false, sp1Config(false), context(), func(pv [projection.PublicValuesLength]byte) []byte {
		return projection.EncodeEnvelope(&projection.Envelope{Kind: projection.EnvelopeKindGroth16, Proof: groth16Shaped()[:355], PublicValues: pv})
	})
	build("sp1_groth16_invalid", false, sp1Config(false), context(), func(pv [projection.PublicValuesLength]byte) []byte {
		return projection.EncodeEnvelope(&projection.Envelope{Kind: projection.EnvelopeKindGroth16, Proof: groth16Shaped(), PublicValues: pv})
	})
	for i := range out {
		if out[i].Accept {
			s, err := projection.ValidateProjectionRange(&out[i].Config, out[i].Context.context(), out[i].Blocks, projection.StubVerifier{})
			require.NoError(t, err)
			record(&out[i], s)
		}
	}
	return out
}

// groth16Shaped is a 356-byte proof with the production circuit prefix and vk root, zero exit code
// and nonce, and every gnark proof coordinate equal to 1: it passes framing and must fail the
// curve check of point A.
func groth16Shaped() []byte {
	p := make([]byte, sp1groth16.ProofLength)
	prefix := sp1groth16.CircuitV6_1_0.CircuitPrefix()
	copy(p[0:4], prefix[:])
	copy(p[36:68], sp1groth16.CircuitV6_1_0.VKRoot[:])
	for i := 100 + 31; i < len(p); i += 32 {
		p[i] = 1
	}
	return p
}

// rejectReason pins why selected vectors are rejected, so a vector cannot pass by failing early
// for an unrelated reason.
var rejectReason = map[string]string{
	"wrong_l1_head":              "claim l1Head does not match",
	"wrong_rollup_config_hash":   "claim rollupConfigHash does not match",
	"wrong_dep_set_hash":         "claim depSetHash does not match",
	"wrong_parent_output":        "private parent differs from canonical checkpoint",
	"recovery_missing_inputs":    "missing canonical recovery inputs",
	"recovery_wrong_anchor_root": "claim does not match canonical continuation",
	"retired_stub_ungated":       "is test-gated and not enabled for chain 10",
	"sp1_allow_events_rejected":  "does not support event replays",
	"sp1_mock_disabled":          "sp1 mock envelopes are not enabled",
	"sp1_mock_ungated_chain":     "is test-gated and not enabled for chain 10",
	"sp1_mock_wrong_vkey":        "program vkey mismatch",
	"sp1_mock_wrong_digest":      "public values digest mismatch",
	"sp1_mock_nonzero_exit":      "exit code, vk root or nonce is non-zero",
	"sp1_mock_nonzero_vk_root":   "exit code, vk root or nonce is non-zero",
	"sp1_mock_nonzero_nonce":     "exit code, vk root or nonce is non-zero",
	"sp1_bad_version":            "unsupported sp1 envelope version",
	"sp1_bad_kind":               "unsupported sp1 envelope kind",
	"sp1_trailing_byte":          "sp1 envelope length 837",
	"sp1_truncated":              "sp1 envelope length 835",
	"sp1_groth16_len_355":        "proof length 355",
	"sp1_groth16_invalid":        "invalid sp1 groth16 proof point",
	"late_gas_below_intrinsic":   "transaction gas below intrinsic or calldata floor",
	"late_gas_below_floor":       "transaction gas below intrinsic or calldata floor",
}

func requireRejectReason(t *testing.T, v vector, verifier func(*vector) (projection.ProofVerifier, error)) {
	want, ok := rejectReason[v.Name]
	var word int
	if _, err := fmt.Sscanf(v.Name, "sp1_pv_word_%d_flipped", &word); err == nil {
		want, ok = fmt.Sprintf("differ from the admission statement at word %d", word), true
	}
	if !ok {
		return
	}
	pv, err := verifier(&v)
	if err == nil {
		_, err = projection.ValidateProjectionRange(&v.Config, v.Context.context(), v.Blocks, pv)
	}
	require.ErrorContains(t, err, want)
}

func TestProofVectors(t *testing.T) {
	syncVectors(t, "testdata/proofs.json", proofVectors(t))
	accepted := 0
	for _, v := range loadVectors(t, "testdata/proofs.json") {
		t.Run(v.Name, func(t *testing.T) {
			checkVector(t, v, configuredVerifier)
			requireRejectReason(t, v, configuredVerifier)
			if v.Accept {
				accepted++
			}
		})
	}
	require.Equal(t, 2, accepted)
}

// The execution-mock digest binds every statement field it covers.
func TestExecutionMockEnvelopes(t *testing.T) {
	v := vectors(t)[0]
	statement, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	good := projection.ExecutionMockProof(*statement)
	require.Equal(t, good, claimProof(t, v.Blocks))
	require.NoError(t, (projection.ExecutionMockVerifier{}).Verify(*statement, good))
	for _, change := range []func(*projection.Statement){
		func(s *projection.Statement) { s.ParentHash[0] ^= 1 },
		func(s *projection.Statement) { s.ChainID[0] ^= 1 },
		func(s *projection.Statement) { s.ProjectionHash[0] ^= 1 },
		func(s *projection.Statement) { s.Continuation.Anchor.Hash[0] ^= 1 },
		func(s *projection.Statement) { s.Continuation.OutputRoot[0] ^= 1 },
		func(s *projection.Statement) { s.Continuation.RecoveryHash[0] ^= 1 },
		func(s *projection.Statement) { s.Continuation.Anchor.Number++ },
	} {
		changed := *statement
		change(&changed)
		require.Error(t, (projection.ExecutionMockVerifier{}).Verify(changed, good))
	}
}
