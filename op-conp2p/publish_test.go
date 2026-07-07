package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

// testChainID matches the chain id used by the op-con-node signing unit tests
// (crates/nodes/op-con-node/src/runtime/sequencer_signing.rs), so the fixtures
// stay comparable across the two languages.
var testChainID = eth.ChainIDFromUInt64(10)

// sampleV3Payload mirrors the Rust `sample_v3_payload` fixture byte for byte: a
// minimal but complete Ecotone-style payload (withdrawals + blob gas set, no
// withdrawals root).
func sampleV3Payload() *eth.ExecutionPayload {
	zero := eth.Uint64Quantity(0)
	return &eth.ExecutionPayload{
		ParentHash:    common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		FeeRecipient:  common.HexToAddress("0x2222222222222222222222222222222222222222"),
		StateRoot:     eth.Bytes32(common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")),
		ReceiptsRoot:  eth.Bytes32(common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")),
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32(common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")),
		BlockNumber:   eth.Uint64Quantity(7),
		GasLimit:      eth.Uint64Quantity(30_000_000),
		GasUsed:       eth.Uint64Quantity(0),
		Timestamp:     eth.Uint64Quantity(100),
		ExtraData:     eth.BytesMax32{},
		BaseFeePerGas: eth.Uint256Quantity(*uint256.NewInt(7)),
		BlockHash:     common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666"),
		Transactions:  []eth.Data{},
		Withdrawals:   &types.Withdrawals{},
		BlobGasUsed:   &zero,
		ExcessBlobGas: &zero,
	}
}

// buildNodeEnvelopeJSON assembles the exact wire JSON op-con-node's
// signed-payload websocket emits for `payload`: the engine executionPayload
// JSON plus parentBeaconBlockRoot, the 65-byte gossip signature, and the SSZ
// payload hash the signature covers.
func buildNodeEnvelopeJSON(t *testing.T, payload *eth.ExecutionPayload, beaconRoot *common.Hash, keyHex string) ([]byte, common.Address) {
	priv, err := crypto.HexToECDSA(keyHex)
	require.NoError(t, err)

	envelope := &eth.ExecutionPayloadEnvelope{ExecutionPayload: payload, ParentBeaconBlockRoot: beaconRoot}
	var buf bytes.Buffer
	if beaconRoot != nil {
		_, err = envelope.MarshalSSZ(&buf)
	} else {
		_, err = payload.MarshalSSZ(&buf)
	}
	require.NoError(t, err)
	payloadHash := opsigner.PayloadHash(buf.Bytes())

	sig, err := opsigner.NewLocalSigner(priv).SignBlockV1(context.Background(), testChainID, payloadHash)
	require.NoError(t, err)

	wire := map[string]any{
		"executionPayload": payload,
		"signature":        fmt.Sprintf("0x%x", sig[:]),
		"payloadHash":      payloadHash.Hex(),
	}
	if beaconRoot != nil {
		wire["parentBeaconBlockRoot"] = beaconRoot.Hex()
	}
	raw, err := json.Marshal(wire)
	require.NoError(t, err)
	return raw, crypto.PubkeyToAddress(priv.PublicKey)
}

// TestPublishRoundTrip proves the publish wire format: a signed envelope in
// op-con-node's feed shape, publish-encoded by the sidecar, must decode through
// the stock op-node receive path (SSZ envelope decode + BlocksV1 signature
// verification against the expected signer) with the ORIGINAL node signature —
// the sidecar never re-signs.
func TestPublishRoundTrip(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	raw, signerAddr := buildNodeEnvelopeJSON(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	// Sidecar-side decode of the ws feed message.
	var msg signedPayloadMsg
	require.NoError(t, json.Unmarshal(raw, &msg))
	signed, err := msg.toSignedEnvelope()
	require.NoError(t, err, "payload-hash fidelity check must pass on a faithful envelope")

	// Publish-encode exactly as PublishSignedL2Payload does: signature prefix,
	// then the SSZ payload envelope (post-Ecotone: beacon root + payload).
	var wireBuf bytes.Buffer
	wireBuf.Write(signed.Signature[:])
	_, err = signed.Envelope.MarshalSSZ(&wireBuf)
	require.NoError(t, err)
	data := wireBuf.Bytes()

	// Receive-side decode, mirroring op-node's BuildBlocksValidator: split the
	// 65-byte signature, verify it against the expected sequencer signer over
	// the payload bytes, then SSZ-decode the envelope.
	require.Greater(t, len(data), 65)
	var gotSig eth.Bytes65
	copy(gotSig[:], data[:65])
	payloadBytes := data[65:]

	block := opsigner.SignedP2PBlock{Raw: payloadBytes, Signature: gotSig}
	auth := &opsigner.OPStackP2PBlockAuthV1{Allowed: signerAddr, Chain: testChainID}
	require.NoError(t, block.VerifySignature(auth),
		"the recovered signer must match the node's sequencer key")

	var decoded eth.ExecutionPayloadEnvelope
	require.NoError(t, decoded.UnmarshalSSZ(eth.BlockV3, uint32(len(payloadBytes)), bytes.NewReader(payloadBytes)))
	require.Equal(t, payload.BlockHash, decoded.ExecutionPayload.BlockHash)
	require.Equal(t, payload.BlockNumber, decoded.ExecutionPayload.BlockNumber)
	require.NotNil(t, decoded.ParentBeaconBlockRoot)
	require.Equal(t, beaconRoot, *decoded.ParentBeaconBlockRoot)

	// A different expected signer must NOT verify (rules out a vacuous check).
	wrongAuth := &opsigner.OPStackP2PBlockAuthV1{Allowed: common.HexToAddress("0xdead"), Chain: testChainID}
	require.Error(t, block.VerifySignature(wrongAuth))
}

// TestPublishRejectsUnfaithfulEnvelope proves the fidelity guard: if the
// node-signed payloadHash does not cover the bytes the sidecar would publish
// (payload tampered or re-encode divergence), the envelope is rejected before
// it ever reaches gossip.
func TestPublishRejectsUnfaithfulEnvelope(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	raw, _ := buildNodeEnvelopeJSON(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	var msg signedPayloadMsg
	require.NoError(t, json.Unmarshal(raw, &msg))

	// Tamper with the payload after signing.
	msg.ExecutionPayload.GasUsed = 1
	_, err := msg.toSignedEnvelope()
	require.ErrorContains(t, err, "payload hash mismatch")

	// Truncated signature is rejected structurally.
	var short signedPayloadMsg
	require.NoError(t, json.Unmarshal(raw, &short))
	short.Signature = short.Signature[:64]
	_, err = short.toSignedEnvelope()
	require.ErrorContains(t, err, "signature must be 65 bytes")
}

// TestPublishNormalizesLegacyRecoveryID proves an envelope whose signature uses
// the Ethereum-legacy v encoding (27/28) is normalized to the raw recovery id
// (0/1) the OP gossip verifiers require, and still verifies against the signer.
func TestPublishNormalizesLegacyRecoveryID(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	raw, signerAddr := buildNodeEnvelopeJSON(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	var msg signedPayloadMsg
	require.NoError(t, json.Unmarshal(raw, &msg))
	require.Less(t, msg.Signature[64], byte(2), "fixture should carry a raw recovery id")
	rawV := msg.Signature[64]
	msg.Signature[64] += 27 // re-encode as legacy 27/28

	signed, err := msg.toSignedEnvelope()
	require.NoError(t, err)
	require.Equal(t, rawV, signed.Signature[64], "legacy v must be normalized back to 0/1")

	var buf bytes.Buffer
	_, err = signed.Envelope.MarshalSSZ(&buf)
	require.NoError(t, err)
	block := opsigner.SignedP2PBlock{Raw: buf.Bytes(), Signature: signed.Signature}
	require.NoError(t, block.VerifySignature(&opsigner.OPStackP2PBlockAuthV1{Allowed: signerAddr, Chain: testChainID}))
}

// TestDecodeSignedPayloadNotification exercises the raw jsonrpsee notification
// framing: only opql_signedUnsafePayload notifications produce envelopes;
// other frames are skipped without error.
func TestDecodeSignedPayloadNotification(t *testing.T) {
	logger := log.NewLogger(log.DiscardHandler())

	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	raw, _ := buildNodeEnvelopeJSON(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	notif := fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":{"subscription":"s1","result":%s}}`,
		signedPayloadNotifMethod, raw)
	signed, ok := decodeSignedPayloadNotification(logger, []byte(notif))
	require.True(t, ok)
	require.Equal(t, payload.BlockHash, signed.Envelope.ExecutionPayload.BlockHash)

	// Non-notification frames (e.g. a stray response) are skipped.
	_, ok = decodeSignedPayloadNotification(logger, []byte(`{"jsonrpc":"2.0","id":9,"result":"ok"}`))
	require.False(t, ok)

	// Garbage is skipped, not fatal.
	_, ok = decodeSignedPayloadNotification(logger, []byte(`{`))
	require.False(t, ok)
}
