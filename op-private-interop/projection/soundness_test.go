package projection_test

// Soundness harness for the sp1-private-projection-v1 statement (spec-sound-profile §H WP5).
//
// Two families of negative tests, both driven through the exported admission path
// (ValidateProjectionRange + VerifierFor), never through package internals:
//
//   - Statement binding. For every accepted span (a self-built sp1 span, and every accept vector
//     in testdata/ranges.json and testdata/proofs.json), flipping any one of the 21 public-values
//     words in an otherwise well-formed envelope of either kind is rejected, and a mock envelope
//     re-signed over the flipped values (so its digest is internally consistent) is rejected at
//     the public-values comparison for exactly that word.
//   - Completeness (N15-N19) and outputs (N20). The published span is mutated (a replay dropped,
//     added, swapped, moved or altered; a middle output record altered) while the claim keeps the
//     GENUINE envelope of the unmodified span. The mutated span is structurally admissible, so the
//     rejection is attributable to the statement: word 17 (records root) and word 19
//     (messagesRoot) or word 18 (outputsRoot) differ.
//
// The expected public values are rebuilt here from the published transactions by an independent
// path (interop payload hashes from messages.LogToMessagePayload over the rebuilt SentMessage log,
// raw validateMessage argument bytes), not by calling the admission code under test.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------------------------
// Span construction (independent of validate_test.go helpers so this file compiles on its own).

type sndBlock struct {
	Timestamp    uint64          `json:"timestamp"`
	Epoch        uint64          `json:"epoch"`
	Transactions []hexutil.Bytes `json:"transactions"`
}

type sndSpan []sndBlock

func (s sndSpan) GetBlockCount() int                         { return len(s) }
func (s sndSpan) GetBlockTimestamp(i int) uint64             { return s[i].Timestamp }
func (s sndSpan) GetBlockEpochNum(i int) uint64              { return s[i].Epoch }
func (s sndSpan) GetBlockTransactions(i int) []hexutil.Bytes { return s[i].Transactions }

const (
	sndChainID    = 901
	sndFirstBlock = 10
)

var (
	sndVKey          = common.HexToHash("0x00aa55ed3c7a07d0233a027278a8b7ff8681ffbd5d1ec4795c18966f6e693001")
	sndPrivateConfig = projection.PrivateConfigHash([]byte(`{"private":"rollup"}`), []byte(`{"l1":"config"}`))
	sndDepSet        = projection.DependencySetHash([]eth.ChainID{eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)})
)

func sndConfig() projection.Config {
	return projection.Config{
		Verifier:          projection.SP1PrivateProjectionV1,
		GenesisOutputRoot: common.Hash{9},
		ProgramVKey:       sndVKey,
		PrivateConfigHash: sndPrivateConfig,
		DependencySetHash: sndDepSet,
		MockProofs:        true,
	}
}

func sndContext() projection.Context {
	return projection.Context{
		ChainID:       big.NewInt(sndChainID),
		GenesisNumber: 0,
		GenesisTime:   1000,
		BlockTime:     2,
		GenesisHash:   common.Hash{0x0e},
		ParentHash:    common.Hash{1},
		L1Head:        common.Hash{0x11, 6},
		Continuation: projection.Continuation{
			Anchor:     eth.BlockID{Number: sndFirstBlock - 1, Hash: common.Hash{1}},
			OutputRoot: common.Hash{9},
		},
	}
}

func sndSign(t *testing.T, tx *types.DynamicFeeTx) hexutil.Bytes {
	t.Helper()
	key, err := crypto.HexToECDSA(strings.Repeat("11", 32))
	require.NoError(t, err)
	out, err := types.SignNewTx(key, types.LatestSignerForChainID(big.NewInt(sndChainID)), tx)
	require.NoError(t, err)
	raw, err := out.MarshalBinary()
	require.NoError(t, err)
	return raw
}

func sndTx(t *testing.T, to common.Address, data []byte, al types.AccessList) hexutil.Bytes {
	return sndSign(t, &types.DynamicFeeTx{ChainID: big.NewInt(sndChainID), Gas: 500000, GasFeeCap: new(big.Int), GasTipCap: new(big.Int), Value: new(big.Int), To: &to, Data: data, AccessList: al})
}

func sndClaimTx(t *testing.T, cfg projection.Config, ctx projection.Context, first, last uint64, proof []byte) hexutil.Bytes {
	c := &codec.RangeClaim{
		FirstBlock:                first,
		LastBlock:                 last,
		PrivateTerminalBlockHash:  common.Hash{2},
		PrivateTerminalParentHash: common.Hash{3},
		L1Head:                    ctx.L1Head,
		RollupConfigHash:          projection.ConfigHash(&cfg, ctx),
		DepSetHash:                cfg.DependencySetHash,
		PrivateDataHash:           common.Hash{4},
		AnchorBlock:               ctx.Continuation.Anchor.Number,
		AnchorOutputRoot:          ctx.Continuation.OutputRoot,
		RecoveryHash:              ctx.Continuation.RecoveryHash,
		ParentOutputRoot:          ctx.Continuation.OutputRoot,
		Proof:                     proof,
	}
	data, err := wire.EncodePostClaim(c)
	require.NoError(t, err)
	return sndTx(t, predeploys.ClaimRegistryAddr, data, nil)
}

func sndOutputTx(t *testing.T, root common.Hash) hexutil.Bytes {
	return sndTx(t, predeploys.ClaimRegistryAddr, wire.EncodeOutput(root), nil)
}

func sndSentMessage(nonce int64, message string) *wire.SentMessage {
	return &wire.SentMessage{Destination: big.NewInt(902), Nonce: big.NewInt(nonce), Sender: common.Address{1}, Target: common.Address{2}, Message: []byte(message)}
}

func sndExportTx(t *testing.T, m *wire.SentMessage) hexutil.Bytes {
	data, err := wire.EncodeReplaySentMessage(m)
	require.NoError(t, err)
	return sndTx(t, predeploys.L2toL2CrossDomainMessengerAddr, data, nil)
}

func sndImportTx(t *testing.T, logIndex uint32, payload byte) hexutil.Bytes {
	trigger := &txintent.ExecTrigger{Msg: messages.Message{
		Identifier:  messages.Identifier{Origin: common.Address{9}, BlockNumber: 4, LogIndex: logIndex, Timestamp: 1002, ChainID: eth.ChainIDFromUInt64(902)},
		PayloadHash: common.Hash{payload},
	}}
	data, err := trigger.EncodeInput()
	require.NoError(t, err)
	al, err := trigger.AccessList()
	require.NoError(t, err)
	return sndTx(t, predeploys.CrossL2InboxAddr, data, al)
}

// sndOutputRoot is the recordOutput root published for span block i.
func sndOutputRoot(i int) common.Hash { return common.Hash{byte(0x20 + i), 0xaa} }

// sndBaseSpan is a three-block span (10..12) with replays in two blocks, so every completeness
// mutation (drop, add, swap within a block, move across blocks, alter) is expressible:
//
//	block 10: postClaim, recordOutput, export(nonce 1)
//	block 11: recordOutput, export(nonce 2), import(log 2), export(nonce 3)
//	block 12: recordOutput, import(log 5)
func sndBaseSpan(t *testing.T, cfg projection.Config, ctx projection.Context, proof []byte) sndSpan {
	return sndSpan{
		{1020, 5, []hexutil.Bytes{sndClaimTx(t, cfg, ctx, 10, 12, proof), sndOutputTx(t, sndOutputRoot(0)), sndExportTx(t, sndSentMessage(1, "one"))}},
		{1022, 5, []hexutil.Bytes{sndOutputTx(t, sndOutputRoot(1)), sndExportTx(t, sndSentMessage(2, "two")), sndImportTx(t, 2, 7), sndExportTx(t, sndSentMessage(3, "three"))}},
		{1024, 6, []hexutil.Bytes{sndOutputTx(t, sndOutputRoot(2)), sndImportTx(t, 5, 8)}},
	}
}

// ---------------------------------------------------------------------------------------------
// Verifier helpers.

// sndCapture accepts any proof and records the statement admission computed.
type sndCapture struct{ got *projection.Statement }

func (c *sndCapture) Verify(s projection.Statement, _ []byte) error {
	*c.got = s
	return nil
}

func sndStatement(t *testing.T, cfg projection.Config, ctx projection.Context, span projection.Range) *projection.Statement {
	t.Helper()
	var s projection.Statement
	_, err := projection.ValidateProjectionRange(&cfg, ctx, span, &sndCapture{got: &s})
	require.NoError(t, err, "the span must be structurally admissible")
	return &s
}

func sndVerifier(t *testing.T, cfg projection.Config) projection.ProofVerifier {
	t.Helper()
	v, err := projection.VerifierFor(&cfg, big.NewInt(sndChainID))
	require.NoError(t, err)
	return v
}

// sndMaskedDigest is SP1's committed-values digest: sha256 with the top three bits cleared.
func sndMaskedDigest(pv []byte) [32]byte {
	d := sha256.Sum256(pv)
	d[0] &= 0x1f
	return d
}

// sndMockEnvelope builds a kind-0x02 envelope byte by byte from the spec's §F.2 layout, rather than
// through projection.MockEnvelope, so the envelope framing is checked too.
func sndMockEnvelope(vkey common.Hash, pv []byte) []byte {
	out := []byte{0x01, 0x02, 0x00, 0xa0}
	out = append(out, vkey[:]...)
	d := sndMaskedDigest(pv)
	out = append(out, d[:]...)
	out = append(out, make([]byte, 96)...)
	return append(out, pv...)
}

// sndGroth16Envelope frames a well-formed-length but non-verifying Groth16 proof around pv.
func sndGroth16Envelope(pv []byte) []byte {
	out := []byte{0x01, 0x01, 0x01, 0x64}
	proof := make([]byte, 356)
	for i := range proof {
		proof[i] = byte(i)
	}
	out = append(out, proof...)
	return append(out, pv...)
}

// sndRequireWord asserts err is the verifier's public-values mismatch at exactly word k.
func sndRequireWord(t *testing.T, err error, k int) {
	t.Helper()
	require.Error(t, err)
	require.Regexp(t, fmt.Sprintf(`public values differ from the admission statement at word %d\b`, k), err.Error())
}

// ---------------------------------------------------------------------------------------------
// Independent expected public values.

func sndWord(pv []byte, k int) common.Hash { return common.BytesToHash(pv[32*k : 32*k+32]) }

func sndU256(n uint64) common.Hash {
	var w common.Hash
	binary.BigEndian.PutUint64(w[24:], n)
	return w
}

var sndSentMessageTopic = crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)"))

// sndExportHash rebuilds the SentMessage log from replay calldata with go-ethereum's ABI package and
// hashes it with the standard interop payload rule.
func sndExportHash(t *testing.T, calldata []byte) common.Hash {
	m, err := wire.DecodeReplaySentMessage(calldata)
	require.NoError(t, err)
	addrT, _ := abi.NewType("address", "", nil)
	bytesT, _ := abi.NewType("bytes", "", nil)
	data, err := abi.Arguments{{Type: addrT}, {Type: bytesT}}.Pack(m.Sender, m.Message)
	require.NoError(t, err)
	log := &types.Log{Topics: []common.Hash{sndSentMessageTopic, common.BigToHash(m.Destination), common.BytesToHash(m.Target[:]), common.BigToHash(m.Nonce)}, Data: data}
	return crypto.Keccak256Hash(messages.LogToMessagePayload(log))
}

// sndExpectedRoots recomputes outputsRoot, messagesRoot and terminalOutput from the published span.
func sndExpectedRoots(t *testing.T, first uint64, span sndSpan) (outputs, msgs, terminal common.Hash) {
	var outLeaves, msgLeaves []common.Hash
	for i, b := range span {
		n := first + uint64(i)
		var rendered uint32
		for _, raw := range b.Transactions {
			var tx types.Transaction
			require.NoError(t, tx.UnmarshalBinary(raw))
			data := tx.Data()
			switch *tx.To() {
			case predeploys.ClaimRegistryAddr:
				if root, err := wire.DecodeOutput(data); err == nil {
					var nb [8]byte
					binary.BigEndian.PutUint64(nb[:], n)
					outLeaves = append(outLeaves, crypto.Keccak256Hash([]byte{0}, nb[:], root[:]))
					terminal = root
				}
			case predeploys.L2toL2CrossDomainMessengerAddr:
				msgLeaves = append(msgLeaves, projection.MessageLeaf(n, rendered, 0x01, sndExportHash(t, data)))
				rendered++
			case predeploys.CrossL2InboxAddr:
				require.Len(t, data, 196)
				msgLeaves = append(msgLeaves, projection.MessageLeaf(n, rendered, 0x02, crypto.Keccak256Hash(data[4:196])))
				rendered++
			}
		}
	}
	return projection.CommitmentRoot("optimism.private-outputs.v1\x00", outLeaves),
		projection.CommitmentRoot("optimism.private-messages.v1\x00", msgLeaves), terminal
}

// ---------------------------------------------------------------------------------------------
// Tests.

// TestSoundnessPublicValuesMatchIndependentPath pins the words a verifier must compute itself
// against values rebuilt here from the context, config and published transactions.
func TestSoundnessPublicValuesMatchIndependentPath(t *testing.T) {
	cfg, ctx := sndConfig(), sndContext()
	span := sndBaseSpan(t, cfg, ctx, nil)
	s := sndStatement(t, cfg, ctx, span)
	pv := projection.PublicValues(s)
	require.Len(t, pv[:], 672)
	outputs, msgs, terminal := sndExpectedRoots(t, sndFirstBlock, span)
	want := map[int]common.Hash{
		0:  crypto.Keccak256Hash([]byte("optimism.private-projection.public-values.v1")),
		1:  sndU256(sndChainID),
		2:  projection.ConfigHash(&cfg, ctx),
		3:  sndPrivateConfig,
		4:  sndDepSet,
		5:  ctx.ParentHash,
		6:  sndU256(ctx.Continuation.Anchor.Number),
		7:  ctx.Continuation.Anchor.Hash,
		8:  ctx.Continuation.OutputRoot,
		9:  ctx.Continuation.RecoveryHash,
		10: sndU256(10),
		11: sndU256(12),
		12: ctx.Continuation.OutputRoot,
		13: {2},
		14: {3},
		15: ctx.L1Head,
		16: {4},
		17: s.ProjectionHash,
		18: outputs,
		19: msgs,
		20: terminal,
	}
	for k := 0; k < 21; k++ {
		require.Equal(t, want[k], sndWord(pv[:], k), "public-values word %d", k)
	}
	require.Equal(t, sndOutputRoot(2), terminal, "terminalOutput is the last record")
	require.NotEqual(t, common.Hash{}, s.ProjectionHash)
}

// TestSoundnessEnvelopeAcceptsGenuineSpan is the positive control for every negative below.
func TestSoundnessEnvelopeAcceptsGenuineSpan(t *testing.T) {
	cfg, ctx := sndConfig(), sndContext()
	s := sndStatement(t, cfg, ctx, sndBaseSpan(t, cfg, ctx, nil))
	pv := projection.PublicValues(s)
	env := sndMockEnvelope(cfg.ProgramVKey, pv[:])
	require.Len(t, env, 836)
	require.Equal(t, projection.MockEnvelope(cfg.ProgramVKey, pv), env, "hand-framed and library mock envelopes agree")
	span := sndBaseSpan(t, cfg, ctx, env)
	_, err := projection.ValidateProjectionRange(&cfg, ctx, span, sndVerifier(t, cfg))
	require.NoError(t, err)
	// With mock_proofs disabled, a mock envelope over that config's own (correct) public values
	// is rejected by the gate, not by the statement binding.
	off := cfg
	off.MockProofs = false
	offStatement := sndStatement(t, off, ctx, sndBaseSpan(t, off, ctx, nil))
	offPV := projection.PublicValues(offStatement)
	offSpan := sndBaseSpan(t, off, ctx, sndMockEnvelope(off.ProgramVKey, offPV[:]))
	_, err = projection.ValidateProjectionRange(&off, ctx, offSpan, sndVerifier(t, off))
	require.ErrorContains(t, err, "mock envelopes are not enabled")
	// A mock envelope under the wrong vkey is rejected.
	wrongKey := sndBaseSpan(t, cfg, ctx, sndMockEnvelope(common.Hash{0x01}, pv[:]))
	_, err = projection.ValidateProjectionRange(&cfg, ctx, wrongKey, sndVerifier(t, cfg))
	require.ErrorContains(t, err, "vkey")
	_, err = projection.VerifierFor(&cfg, big.NewInt(10))
	require.Error(t, err, "a mock-proof config must not pass CheckChain on chain 10")
}

// flipAllWords runs the statement-binding matrix for one admission statement.
func flipAllWords(t *testing.T, s *projection.Statement, verifier projection.ProofVerifier, vkey common.Hash) {
	genuine := projection.PublicValues(s)
	require.NoError(t, verifier.Verify(*s, sndMockEnvelope(vkey, genuine[:])), "genuine mock envelope")
	for k := 0; k < 21; k++ {
		for _, kind := range []string{"mock", "mock-resigned", "groth16"} {
			t.Run(fmt.Sprintf("word%02d/%s", k, kind), func(t *testing.T) {
				pv := genuine
				pv[32*k+31] ^= 0x01
				var env []byte
				switch kind {
				case "mock":
					// Digest still over the genuine values: inconsistent envelope.
					env = sndMockEnvelope(vkey, genuine[:])
					copy(env[4+160:], pv[:])
				case "mock-resigned":
					// Internally consistent mock proof over the flipped values: only the
					// statement binding can reject it.
					env = sndMockEnvelope(vkey, pv[:])
				case "groth16":
					env = sndGroth16Envelope(pv[:])
				}
				sndRequireWord(t, verifier.Verify(*s, env), k)
			})
		}
	}
}

func TestSoundnessEveryPublicValuesWordIsBound(t *testing.T) {
	cfg, ctx := sndConfig(), sndContext()
	s := sndStatement(t, cfg, ctx, sndBaseSpan(t, cfg, ctx, nil))
	flipAllWords(t, s, sndVerifier(t, cfg), cfg.ProgramVKey)
}

// TestSoundnessRecoveryModeWordsBound covers the recovery-mode continuation (anchor < first-1),
// where parentOutputRoot (word 12) is bound only by the proof.
func TestSoundnessRecoveryModeWordsBound(t *testing.T) {
	cfg, ctx := sndConfig(), sndContext()
	ctx.Continuation = projection.Continuation{Anchor: eth.BlockID{Number: sndFirstBlock - 3, Hash: common.Hash{0x77}}, OutputRoot: common.Hash{0x78}, RecoveryHash: common.Hash{0x79}}
	span := sndBaseSpan(t, cfg, ctx, nil)
	s := sndStatement(t, cfg, ctx, span)
	pv := projection.PublicValues(s)
	require.Equal(t, sndU256(sndFirstBlock-3), sndWord(pv[:], 6))
	require.Equal(t, common.Hash{0x79}, sndWord(pv[:], 9))
	flipAllWords(t, s, sndVerifier(t, cfg), cfg.ProgramVKey)
}

// TestSoundnessCompletenessMutationsRejected is N15-N20 at admission: a correct proof for the
// unmodified span does not admit the modified span.
func TestSoundnessCompletenessMutationsRejected(t *testing.T) {
	cfg, ctx := sndConfig(), sndContext()
	genuineStatement := sndStatement(t, cfg, ctx, sndBaseSpan(t, cfg, ctx, nil))
	genuinePV := projection.PublicValues(genuineStatement)
	env := sndMockEnvelope(cfg.ProgramVKey, genuinePV[:])
	cases := []struct {
		name   string
		id     string
		mutate func(sndSpan)
		word   int // the first public-values word expected to differ after word 17
		also   int // the commitment root that must change
	}{
		{"drop_export_replay", "N15", func(s sndSpan) {
			s[1].Transactions = append(s[1].Transactions[:1:1], s[1].Transactions[2:]...)
		}, projection.WordProjectionHash, projection.WordMessagesRoot},
		{"extra_validate_message", "N16", func(s sndSpan) {
			s[2].Transactions = append(s[2].Transactions, sndImportTx(t, 9, 9))
		}, projection.WordProjectionHash, projection.WordMessagesRoot},
		{"swap_two_replays", "N17", func(s sndSpan) {
			s[1].Transactions[1], s[1].Transactions[2] = s[1].Transactions[2], s[1].Transactions[1]
		}, projection.WordProjectionHash, projection.WordMessagesRoot},
		{"move_replay_to_next_block", "N18", func(s sndSpan) {
			moved := s[1].Transactions[3]
			s[1].Transactions = s[1].Transactions[:3]
			s[2].Transactions = append([]hexutil.Bytes{s[2].Transactions[0], moved}, s[2].Transactions[1:]...)
		}, projection.WordProjectionHash, projection.WordMessagesRoot},
		{"replay_different_message_bytes", "N19", func(s sndSpan) {
			s[1].Transactions[1] = sndExportTx(t, sndSentMessage(2, "twO"))
		}, projection.WordProjectionHash, projection.WordMessagesRoot},
		{"middle_output_altered", "N20", func(s sndSpan) {
			s[1].Transactions[0] = sndOutputTx(t, common.Hash{0xde, 0xad})
		}, projection.WordProjectionHash, projection.WordOutputsRoot},
	}
	for _, tc := range cases {
		t.Run(tc.id+"_"+tc.name, func(t *testing.T) {
			// Structurally admissible, but a different statement.
			bare := sndBaseSpan(t, cfg, ctx, nil)
			tc.mutate(bare)
			mutated := sndStatement(t, cfg, ctx, bare)
			mpv := projection.PublicValues(mutated)
			require.NotEqual(t, sndWord(genuinePV[:], tc.also), sndWord(mpv[:], tc.also), "commitment root %d must change", tc.also)
			require.NotEqual(t, sndWord(genuinePV[:], projection.WordProjectionHash), sndWord(mpv[:], projection.WordProjectionHash))

			// The independent recomputation agrees with admission on the mutated span.
			outputs, msgs, terminal := sndExpectedRoots(t, sndFirstBlock, bare)
			require.Equal(t, outputs, mutated.OutputsRoot)
			require.Equal(t, msgs, mutated.MessagesRoot)
			require.Equal(t, terminal, mutated.TerminalOutput)

			// With the genuine envelope in the claim, admission rejects at the first differing word.
			withProof := sndBaseSpan(t, cfg, ctx, env)
			tc.mutate(withProof)
			_, err := projection.ValidateProjectionRange(&cfg, ctx, withProof, sndVerifier(t, cfg))
			sndRequireWord(t, err, tc.word)
		})
	}
}

// TestSoundnessClaimBindingsInEveryMode checks the three new claim-field bindings (§C.4 step 3)
// reject in the sp1 mode and in both legacy test-gated modes.
func TestSoundnessClaimBindingsInEveryMode(t *testing.T) {
	for _, verifier := range []string{projection.SP1PrivateProjectionV1, projection.ExecutionMock, projection.InsecureStub} {
		cfg := sndConfig()
		if verifier != projection.SP1PrivateProjectionV1 {
			cfg = projection.Config{Verifier: verifier, GenesisOutputRoot: common.Hash{9}, DependencySetHash: sndDepSet}
		}
		for field, wantErr := range map[string]string{"l1_head": "l1Head", "rollup_config_hash": "rollupConfigHash", "dep_set_hash": "depSetHash"} {
			t.Run(verifier+"/"+field, func(t *testing.T) {
				ctx := sndContext()
				var tx types.Transaction
				span := sndBaseSpan(t, cfg, ctx, nil)
				require.NoError(t, tx.UnmarshalBinary(span[0].Transactions[0]))
				c, err := wire.DecodeClaim(tx.Data())
				require.NoError(t, err)
				switch field {
				case "l1_head":
					c.L1Head[31] ^= 1
				case "rollup_config_hash":
					c.RollupConfigHash[31] ^= 1
				case "dep_set_hash":
					c.DepSetHash[31] ^= 1
				}
				data, err := wire.EncodePostClaim(c)
				require.NoError(t, err)
				span[0].Transactions[0] = sndTx(t, predeploys.ClaimRegistryAddr, data, nil)
				var s projection.Statement
				_, err = projection.ValidateProjectionRange(&cfg, ctx, span, &sndCapture{got: &s})
				require.ErrorContains(t, err, wantErr)
			})
		}
	}
}

// ---------------------------------------------------------------------------------------------
// Shared vectors: every accept vector in ranges.json and proofs.json (schema v2, §F.5).

type sndFlexUint uint64

func (u *sndFlexUint) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	var (
		n   uint64
		err error
	)
	if strings.HasPrefix(s, "0x") {
		n, err = strconv.ParseUint(s[2:], 16, 64)
	} else {
		n, err = strconv.ParseUint(s, 10, 64)
	}
	*u = sndFlexUint(n)
	return err
}

type sndVectorContext struct {
	ChainID       sndFlexUint `json:"chain_id"`
	GenesisNumber sndFlexUint `json:"genesis_number"`
	GenesisHash   common.Hash `json:"genesis_hash"`
	GenesisTime   sndFlexUint `json:"genesis_time"`
	BlockTime     sndFlexUint `json:"block_time"`
	ParentHash    common.Hash `json:"parent_hash"`
	L1Head        common.Hash `json:"l1_head"`
	Continuation  struct {
		Anchor struct {
			Number sndFlexUint `json:"number"`
			Hash   common.Hash `json:"hash"`
		} `json:"anchor"`
		OutputRoot   common.Hash `json:"output_root"`
		RecoveryHash common.Hash `json:"recovery_hash"`
	} `json:"continuation"`
}

type sndVector struct {
	Name         string            `json:"name"`
	Config       projection.Config `json:"config"`
	Context      *sndVectorContext `json:"context"`
	Blocks       sndSpan           `json:"blocks"`
	Accept       bool              `json:"accept"`
	PublicValues hexutil.Bytes     `json:"public_values"`
}

func (v *sndVector) context() projection.Context {
	c := v.Context
	return projection.Context{
		ChainID:       new(big.Int).SetUint64(uint64(c.ChainID)),
		GenesisNumber: uint64(c.GenesisNumber),
		GenesisTime:   uint64(c.GenesisTime),
		BlockTime:     uint64(c.BlockTime),
		GenesisHash:   c.GenesisHash,
		ParentHash:    c.ParentHash,
		L1Head:        c.L1Head,
		Continuation: projection.Continuation{
			Anchor:       eth.BlockID{Number: uint64(c.Continuation.Anchor.Number), Hash: c.Continuation.Anchor.Hash},
			OutputRoot:   c.Continuation.OutputRoot,
			RecoveryHash: c.Continuation.RecoveryHash,
		},
	}
}

// errVectorsNotV2 marks a vector file that predates schema v2 (no per-vector context).
var errVectorsNotV2 = errors.New("vector file has no schema-v2 context")

func sndLoadVectors(path string) ([]sndVector, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []sndVector
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	for _, v := range out {
		if v.Context == nil {
			return nil, errVectorsNotV2
		}
	}
	return out, nil
}

func TestSoundnessSharedVectorsEveryWordBound(t *testing.T) {
	for _, file := range []string{"testdata/ranges.json", "testdata/proofs.json"} {
		t.Run(file, func(t *testing.T) {
			vectors, err := sndLoadVectors(file)
			if errors.Is(err, errVectorsNotV2) {
				t.Fatalf("%s is not schema v2 (spec §F.5); regenerate with -update-projection-vectors", file)
			}
			require.NoError(t, err)
			accepted := 0
			for _, v := range vectors {
				if !v.Accept {
					continue
				}
				accepted++
				t.Run(v.Name, func(t *testing.T) {
					ctx := v.context()
					var s projection.Statement
					_, err := projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, &sndCapture{got: &s})
					require.NoError(t, err)
					pv := projection.PublicValues(&s)
					if len(v.PublicValues) > 0 {
						require.Equal(t, hexutil.Bytes(pv[:]), v.PublicValues, "vector public values")
					}
					// The sp1 verifier over this statement, whatever the vector's own mode.
					sp1 := sndConfig()
					flipAllWords(t, &s, sndVerifier(t, sp1), sp1.ProgramVKey)
				})
			}
			require.NotZero(t, accepted, "no accept vectors in %s", file)
		})
	}
}
