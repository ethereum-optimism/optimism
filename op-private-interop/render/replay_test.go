package render

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
)

// testKey is a fixed key so signed transaction bytes are reproducible across runs. Never used for
// anything but tests.
var testKey, _ = crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")

func testBuilder(t *testing.T) *BatcherTxBuilder {
	t.Helper()
	chainID := big.NewInt(901)
	return NewBatcherTxBuilder(chainID, DefaultGasPolicy(), PrivateKeySigner(testKey, chainID))
}

// TestReplaySentMessageRoundTrips pins the decode/encode pair against the FINAL contract signature.
// The messenger replay takes the SentMessage's decoded fields and re-emits the event itself, so a
// wrong decode here is a rendering that carries a different message than the private chain sent.
func TestReplaySentMessageRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		name    string
		message []byte
	}{
		{name: "empty message"},
		{name: "short message", message: []byte{1, 2, 3}},
		{name: "one word", message: make([]byte, 32)},
		{name: "spanning words", message: make([]byte, 100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &types.Log{
				Address: msgr,
				Topics: []common.Hash{
					SentMessageEventTopic,
					common.BigToHash(big.NewInt(902)),
					common.BytesToHash(otherAddr.Bytes()),
					common.BigToHash(big.NewInt(7)),
				},
				Data: sentMessageData(extraAddr, tc.message),
			}
			got, err := DecodeSentMessage(l.Topics, l.Data)
			require.NoError(t, err)
			require.Equal(t, big.NewInt(902), got.Destination)
			require.Equal(t, big.NewInt(7), got.Nonce)
			require.Equal(t, extraAddr, got.Sender)
			require.Equal(t, otherAddr, got.Target)
			require.True(t, bytes.Equal(tc.message, got.Message), "the message survives the round trip")

			data, err := EncodeReplaySentMessage(got)
			require.NoError(t, err)
			require.Equal(t, ReplaySentMessageSelector[:], data[:4])
			want, err := replaySentMessageArgs.Pack(got.Destination, got.Nonce, got.Sender, got.Target, got.Message)
			require.NoError(t, err)
			require.Equal(t, want, data[4:])
		})
	}
}

func TestDecodeSentMessageRefusesMalformed(t *testing.T) {
	good := exportLog(1)
	t.Run("wrong topic count", func(t *testing.T) {
		_, err := DecodeSentMessage(good.Topics[:3], good.Data)
		require.ErrorContains(t, err, "expected 4")
	})
	t.Run("wrong topic0", func(t *testing.T) {
		topics := append([]common.Hash(nil), good.Topics...)
		topics[0] = common.Hash{0x1}
		_, err := DecodeSentMessage(topics, good.Data)
		require.ErrorContains(t, err, "not SentMessage")
	})
	t.Run("target topic is not an address", func(t *testing.T) {
		// Truncating a dirty topic would render the message to a DIFFERENT target.
		topics := append([]common.Hash(nil), good.Topics...)
		topics[2][0] = 0xff
		_, err := DecodeSentMessage(topics, good.Data)
		require.ErrorContains(t, err, "is not an address")
	})
	t.Run("truncated data", func(t *testing.T) {
		_, err := DecodeSentMessage(good.Topics, good.Data[:31])
		require.Error(t, err)
	})
}

// TestEncodeReplayEventMatchesABI pins the generic replayer's encoding, and its topic cap: the EVM
// has no log5 and EventReplayer reverts, so refusing at build time turns a reverting transaction on
// a live rendering into a build error naming the log.
func TestEncodeReplayEventMatchesABI(t *testing.T) {
	for _, tc := range []struct {
		name   string
		topics []common.Hash
		data   []byte
	}{
		{name: "no topics no data"},
		{name: "one topic", topics: []common.Hash{{0x01}}},
		{name: "four topics", topics: []common.Hash{{0x01}, {0x02}, {0x03}, {0x04}}},
		{name: "data not word aligned", topics: []common.Hash{{0x01}}, data: []byte{1, 2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EncodeReplayEvent(tc.topics, tc.data)
			require.NoError(t, err)
			require.Equal(t, ReplayEventSelector[:], got[:4])
			arr := make([][32]byte, len(tc.topics))
			for i, h := range tc.topics {
				arr[i] = h
			}
			want, err := replayEventArgs.Pack(arr, tc.data)
			require.NoError(t, err)
			require.Equal(t, want, got[4:])
		})
	}
	_, err := EncodeReplayEvent(make([]common.Hash, 5), nil)
	require.ErrorContains(t, err, "maximum is 4")
}

func TestPostClaimCalldata(t *testing.T) {
	claim := &codec.RangeClaim{
		FirstBlock:               300,
		LastBlock:                599,
		PrivateTerminalBlockHash: common.Hash{0xaa},
		L1Head:                   common.Hash{0xbb},
		RollupConfigHash:         common.Hash{0xcc},
		DepSetHash:               common.Hash{0xdd},
		PrivateDataHash:          common.Hash{0xee},
	}
	got, err := EncodePostClaim(claim)
	require.NoError(t, err)
	require.Equal(t, PostClaimSelector[:], got[:4])

	// The registry receives exactly what the codec encodes, and the registry is log-less, so its
	// CALLDATA is the record and a reader applies the strict decoder to it without a second framing.
	body, err := codec.Encode(claim)
	require.NoError(t, err)
	require.Equal(t, body, got[4:])
	require.Len(t, got, 4+codec.EncodedSizeEmptyProof)

	round, err := codec.Decode(got[4:])
	require.NoError(t, err)
	require.Equal(t, claim, round)
}

func TestBatcherTxBuilderClaimTx(t *testing.T) {
	claim := &codec.RangeClaim{FirstBlock: 0, LastBlock: 299, PrivateTerminalBlockHash: common.Hash{0x7}}

	// Unconfigured, the builder refuses rather than posting the claim to the zero address.
	unset := testBuilder(t)
	_, err := unset.ClaimTx(claim)
	require.ErrorContains(t, err, "not configured")

	b := testBuilder(t)
	b.SetRegistry(extraAddr)
	b.Reset(9)
	tx, err := b.ClaimTx(claim)
	require.NoError(t, err)
	require.Equal(t, extraAddr, *tx.To())
	require.Equal(t, uint64(9), tx.Nonce())
	want, err := EncodePostClaim(claim)
	require.NoError(t, err)
	require.Equal(t, want, tx.Data())
	require.Equal(t, uint64(10), b.Nonce())
}

func TestSelectorsAreStable(t *testing.T) {
	// Pinned literally against the contracts on karl/private-interop: a signature change there must
	// show up here rather than as a reverting transaction on a live rendering.
	require.Equal(t, "replaySentMessage(uint256,uint256,address,address,bytes)", ReplaySentMessageSig)
	require.Equal(t, "replayEvent(bytes32[],bytes)", ReplayEventSig)
	require.Equal(t, crypto.Keccak256([]byte(ReplaySentMessageSig))[:4], ReplaySentMessageSelector[:])
	require.Equal(t, crypto.Keccak256([]byte(ReplayEventSig))[:4], ReplayEventSelector[:])
	require.Equal(t,
		"postClaim((uint8,uint64,uint64,bytes32,bytes32,uint64,bytes32,bytes32,bytes32,bytes32,bytes32,bytes32,bytes32,bytes))",
		PostClaimSig)
	require.Equal(t, crypto.Keccak256([]byte(PostClaimSig))[:4], PostClaimSelector[:])

	// THE ONE CHECK THAT IS NOT SELF-REFERENTIAL. Everything above compares this package against
	// itself or against a string in this file, and that is exactly how the selector went stale: when
	// the claim gained privateTerminalParentHash, PostClaimSig kept its nine-field list and the
	// literal above was updated to match it, so every assertion here passed while the batcher sent
	// 0x41a02b4d to a registry that answers 0x4db071ca. Every postClaim reverted on a live devstack
	// pair, and the follow module -- comparing incoming calldata against the same stale constant --
	// decoded those very transactions without complaint.
	//
	// 0x46e3eef2 is solc's own methodIdentifier for
	// ClaimRegistry.postClaim(RangeClaim), read from the compiled artifact. It is a fact about the
	// CONTRACT, so it cannot move when this package does.
	require.Equal(t, "46e3eef2", fmt.Sprintf("%x", PostClaimSelector),
		"the postClaim selector must match the deployed ClaimRegistry's; a claim sent with any other "+
			"selector reaches a contract with no fallback and reverts")
}

func TestBatcherTxBuilderExport(t *testing.T) {
	b := testBuilder(t)
	b.Reset(7)
	rendered, err := RenderBlock(block(50, 5000, []*types.Log{exportLog(3)}), EmitterSet{})
	require.NoError(t, err)
	require.Len(t, rendered.Actions, 1)
	act := rendered.Actions[0]
	require.Equal(t, ReplayExport, act.Kind)
	require.NotNil(t, act.Export)

	tx, err := b.ReplayTx(act)
	require.NoError(t, err)
	require.Equal(t, uint64(7), tx.Nonce())
	require.Equal(t, uint8(types.DynamicFeeTxType), tx.Type())
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, *tx.To(),
		"the replay implementation lives AT the messenger predeploy address")
	want, err := EncodeReplaySentMessage(act.Export)
	require.NoError(t, err)
	require.Equal(t, want, tx.Data())
	require.Equal(t, DefaultGasPolicy().GasLimitExport+uint64(len(act.Export.Message))*ExportGasPerMessageByte, tx.Gas())
	require.Empty(t, tx.AccessList(), "an export re-emission needs no access list")
	require.Equal(t, uint64(8), b.Nonce())
}

func TestBatcherTxBuilderRefusesOversizeExport(t *testing.T) {
	b := testBuilder(t)
	_, err := b.ReplayTx(ReplayAction{
		Kind: ReplayExport,
		Export: &SentMessage{
			Message: make([]byte, MaxRenderableMessageSize+1),
		},
	})
	require.ErrorContains(t, err, "exceeding the 1048576-byte rendering limit")
	require.Zero(t, b.Nonce(), "a refused action must not consume the deterministic sender nonce")
}

func TestBatcherTxBuilderImportReusesTxintent(t *testing.T) {
	id := sampleIdentifier(4)
	payloadHash := common.Hash{0x99}
	b := testBuilder(t)
	b.Reset(0)

	rendered, err := RenderBlock(block(50, 5000, []*types.Log{importLog(t, id, payloadHash)}), EmitterSet{})
	require.NoError(t, err)
	require.Len(t, rendered.Actions, 1)

	tx, err := b.ReplayTx(rendered.Actions[0])
	require.NoError(t, err)
	require.Equal(t, predeploys.CrossL2InboxAddr, *tx.To())

	// Byte-for-byte the same as the in-tree encoder every other executing-message sender uses.
	trigger := &txintent.ExecTrigger{
		Executor: predeploys.CrossL2InboxAddr,
		Msg:      messages.Message{Identifier: id, PayloadHash: payloadHash},
	}
	wantData, err := trigger.EncodeInput()
	require.NoError(t, err)
	wantAL, err := trigger.AccessList()
	require.NoError(t, err)
	require.Equal(t, wantData, tx.Data())
	require.Equal(t, wantAL, tx.AccessList())
	require.NotEmpty(t, tx.AccessList(), "an import carries the checksum access list")
}

func TestBatcherTxBuilderNoncesSequentially(t *testing.T) {
	b := testBuilder(t)
	b.Reset(100)
	rendered, err := RenderBlock(block(50, 5000, []*types.Log{exportLog(0)}), EmitterSet{})
	require.NoError(t, err)
	for i := range 5 {
		tx, err := b.ReplayTx(rendered.Actions[0])
		require.NoError(t, err)
		require.Equal(t, uint64(100+i), tx.Nonce())
	}
	require.Equal(t, uint64(105), b.Nonce())
}

// TestBatcherTxBuilderReplayEvent covers the third kind. Only CONFIGURED EXTRA EMITTERS reach it:
// EventReplayer emits at its own address, so nothing carrying a protocol claim may be routed here
// (which is why the messenger's RelayedMessage is excluded from the emitter set outright rather
// than rendered through this path).
func TestBatcherTxBuilderReplayEvent(t *testing.T) {
	extra := &types.Log{
		Address: extraAddr,
		Topics:  []common.Hash{{0x1}, {0x2}, {0x3}},
		Data:    common.Hash{0x4}.Bytes(),
	}
	rendered, err := RenderBlock(block(50, 5000, []*types.Log{extra}), NewEmitterSet(extraAddr))
	require.NoError(t, err)
	require.Len(t, rendered.Actions, 1)
	require.Equal(t, ReplayEvent, rendered.Actions[0].Kind)

	// Unconfigured, the builder refuses rather than sending to the zero address: a reverting replay
	// is a rendering that silently lost a log.
	unset := testBuilder(t)
	_, err = unset.ReplayTx(rendered.Actions[0])
	require.ErrorContains(t, err, "genesis address is not configured")

	b := testBuilder(t)
	replayer := common.HexToAddress("0x00000000000000000000000000000000000e0e0e")
	b.SetEventReplayer(replayer)
	tx, err := b.ReplayTx(rendered.Actions[0])
	require.NoError(t, err)
	require.Equal(t, replayer, *tx.To(), "the log is re-emitted at the REPLAYER's address")
	want, err := EncodeReplayEvent(extra.Topics, extra.Data)
	require.NoError(t, err)
	require.Equal(t, want, tx.Data())
}

// TestBatcherTxBuilderIsDeterministic is the replay half of the byte-determinism gate: two
// independent builders at the same starting nonce must sign identical bytes.
func TestBatcherTxBuilderIsDeterministic(t *testing.T) {
	run := func() [][]byte {
		b := testBuilder(t)
		b.SetRegistry(extraAddr)
		b.Reset(12)
		rendered, err := RenderBlock(block(60, 6000,
			[]*types.Log{exportLog(0), importLog(t, sampleIdentifier(1), common.Hash{0x5}), exportLog(1)},
		), EmitterSet{})
		require.NoError(t, err)
		var out [][]byte
		for _, act := range rendered.Actions {
			tx, err := b.ReplayTx(act)
			require.NoError(t, err)
			raw, err := tx.MarshalBinary()
			require.NoError(t, err)
			out = append(out, raw)
		}
		claim, err := b.ClaimTx(&codec.RangeClaim{FirstBlock: 0, LastBlock: 299, PrivateTerminalBlockHash: common.Hash{0x7}})
		require.NoError(t, err)
		raw, err := claim.MarshalBinary()
		require.NoError(t, err)
		return append(out, raw)
	}
	require.Equal(t, run(), run())
}

func TestBatcherTxBuilderRefusesBrokenImport(t *testing.T) {
	b := testBuilder(t)
	_, err := b.ReplayTx(ReplayAction{Kind: ReplayImport})
	require.ErrorContains(t, err, "no decoded message")
}

func TestExportGasBudgetBoundary(t *testing.T) {
	base := DefaultGasPolicy().GasLimitExport
	// All-zero payloads have the cheapest calldata floor, so the linear rule sets the ceiling.
	gas, err := exportGasLimit(base, 581329, make([]byte, 581329+196))
	require.NoError(t, err)
	require.LessOrEqual(t, gas, uint64(16777216))
	_, err = exportGasLimit(base, 581330, make([]byte, 581330+196))
	require.ErrorContains(t, err, "projection transaction ceiling")
}

// exportTx builds the export replay of a payload of n copies of fill.
func exportTx(t *testing.T, n int, fill byte) (*types.Transaction, error) {
	t.Helper()
	b := testBuilder(t)
	return b.ReplayTx(ReplayAction{Kind: ReplayExport, Export: &SentMessage{
		Destination: big.NewInt(902), Nonce: big.NewInt(1), Sender: extraAddr, Target: otherAddr,
		Message: bytes.Repeat([]byte{fill}, n),
	}})
}

func requireCoversFloor(t *testing.T, tx *types.Transaction, margin uint64) {
	t.Helper()
	floor, err := core.FloorDataGas(tx.Data())
	require.NoError(t, err)
	intrinsic, err := core.IntrinsicGas(tx.Data(), tx.AccessList(), nil, false, true, true, true)
	require.NoError(t, err)
	require.GreaterOrEqual(t, tx.Gas(), floor+margin, "gas must cover the EIP-7623 floor plus margin")
	require.GreaterOrEqual(t, tx.Gas(), intrinsic)
}

// TestExportGasCoversCalldataFloor is the H4 boundary: the export limit is
// max(base + 28·len, FloorDataGas + 21000) at every size up to the documented maximum.
func TestExportGasCoversCalldataFloor(t *testing.T) {
	base := DefaultGasPolicy().GasLimitExport
	for _, n := range []int{0, 1, 40 * 1024, 64 * 1024, MaxExportMessageBytesAllNonZero} {
		for _, fill := range []byte{0x00, 0xff} {
			t.Run(fmt.Sprintf("%d bytes of %#x", n, fill), func(t *testing.T) {
				tx, err := exportTx(t, n, fill)
				require.NoError(t, err)
				requireCoversFloor(t, tx, ExportFloorMargin)
				require.GreaterOrEqual(t, tx.Gas(), base+uint64(n)*ExportGasPerMessageByte)
				floor, err := core.FloorDataGas(tx.Data())
				require.NoError(t, err)
				require.Equal(t, max(base+uint64(n)*ExportGasPerMessageByte, floor+ExportFloorMargin), tx.Gas())
				require.LessOrEqual(t, tx.Gas(), uint64(16777216))
			})
		}
	}
	// Above 38 KiB of incompressible payload the floor, not the linear rule, sets the limit: the
	// previous policy under-gassed exactly these replays.
	tx, err := exportTx(t, 40*1024, 0xff)
	require.NoError(t, err)
	require.Greater(t, tx.Gas(), base+40*1024*ExportGasPerMessageByte)

	// The documented maximum is exact for incompressible payloads.
	_, err = exportTx(t, MaxExportMessageBytesAllNonZero+1, 0xff)
	require.ErrorContains(t, err, "projection transaction ceiling")
	// Zero-byte payloads beyond it still render, up to the linear ceiling.
	_, err = exportTx(t, MaxExportMessageBytesAllNonZero+1, 0x00)
	require.NoError(t, err)
}

// TestClaimGasCoversCalldataFloor: the claim limit is max(GasLimitClaim, FloorDataGas + 100000),
// so a claim carrying the largest proof is not under-gassed.
func TestClaimGasCoversCalldataFloor(t *testing.T) {
	b := testBuilder(t)
	b.SetRegistry(extraAddr)
	for _, proofLen := range []int{0, 836, 1032, 65536} {
		t.Run(fmt.Sprintf("proof %d bytes", proofLen), func(t *testing.T) {
			claim := &codec.RangeClaim{FirstBlock: 1, LastBlock: 2,
				Proof: bytes.Repeat([]byte{0xff}, proofLen)}
			tx, err := b.ClaimTx(claim)
			require.NoError(t, err)
			requireCoversFloor(t, tx, ClaimFloorMargin)
			floor, err := core.FloorDataGas(tx.Data())
			require.NoError(t, err)
			require.Equal(t, max(DefaultGasPolicy().GasLimitClaim, floor+ClaimFloorMargin), tx.Gas())
		})
	}
	// The largest proof needs far more than the fixed default.
	tx, err := b.ClaimTx(&codec.RangeClaim{Proof: bytes.Repeat([]byte{0xff}, 65536)})
	require.NoError(t, err)
	require.Greater(t, tx.Gas(), DefaultGasPolicy().GasLimitClaim)
}

// TestOutputGasCoversCalldataFloor: the fixed recordOutput limit covers its worst-case floor.
func TestOutputGasCoversCalldataFloor(t *testing.T) {
	b := testBuilder(t)
	b.SetRegistry(extraAddr)
	tx, err := b.OutputTx(common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))
	require.NoError(t, err)
	require.Equal(t, OutputGasLimit, tx.Gas())
	requireCoversFloor(t, tx, 21_000)
}

func TestSetGasPolicy(t *testing.T) {
	b := testBuilder(t)
	override := DefaultGasPolicy()
	override.GasLimitImport = 21_100
	b.SetGasPolicy(override)
	require.Equal(t, override, b.GasPolicy())
}
