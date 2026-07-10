package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
)

// testChainID matches the chain id used by the opql signing unit tests
// (crates/lib/opql-direct-sync/src/signing.rs), so the fixtures stay
// comparable across the two languages.
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

// buildDirectSyncMsg assembles the Direct Sync carrier the node's websocket
// emits: `{version, signed, seq, payload: snappy(sig(65) || SSZ(envelope))}`,
// signed over the version-shaped payload hash.
func buildDirectSyncMsg(t *testing.T, payload *eth.ExecutionPayload, beaconRoot *common.Hash, keyHex string) (*directSyncMsg, common.Address) {
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

	raw := make([]byte, 0, 65+buf.Len())
	raw = append(raw, sig[:]...)
	raw = append(raw, buf.Bytes()...)
	seq := hexutil.Uint64(41)
	return &directSyncMsg{
		Version: 3,
		Signed:  true,
		Seq:     &seq,
		Payload: snappy.Encode(nil, raw),
	}, crypto.PubkeyToAddress(priv.PublicKey)
}

// TestDirectSyncDecodeRoundTrip proves the verbatim-publish contract: the
// carrier decodes into the typed envelope AND the exact `sig || ssz` bytes the
// gossip topic will carry, with the ORIGINAL node signature verifying against
// the signer through the stock op-node auth path — the sidecar never re-signs
// and never re-encodes.
func TestDirectSyncDecodeRoundTrip(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	msg, signerAddr := buildDirectSyncMsg(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	decoded, err := decodeDirectSyncPayload(msg)
	require.NoError(t, err)
	require.Equal(t, uint64(7), decoded.blockNumber())
	require.Equal(t, uint64(100), decoded.timestamp())
	require.Equal(t, payload.BlockHash, decoded.envelope.ExecutionPayload.BlockHash)
	require.NotNil(t, decoded.envelope.ParentBeaconBlockRoot)
	require.Equal(t, beaconRoot, *decoded.envelope.ParentBeaconBlockRoot)

	// The raw bytes are exactly the decompressed carrier payload (verbatim).
	uncompressed, err := snappy.Decode(nil, msg.Payload)
	require.NoError(t, err)
	require.Equal(t, uncompressed, decoded.raw)

	// The node's signature over those exact bytes verifies via the stock
	// op-node gossip auth (BlocksV1 domain).
	signed := &opsigner.SignedP2PBlock{Raw: decoded.raw[65:], Signature: decoded.signature}
	require.NoError(t, signed.VerifySignature(&opsigner.OPStackP2PBlockAuthV1{
		Allowed: signerAddr,
		Chain:   testChainID,
	}))
}

// TestDirectSyncNormalizesLegacyRecoveryID: a feed emitting the
// Ethereum-legacy 27/28 v byte still publishes gossip-valid bytes (v is
// recovery metadata, not signed content).
func TestDirectSyncNormalizesLegacyRecoveryID(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	msg, _ := buildDirectSyncMsg(t, payload, &beaconRoot,
		"0202020202020202020202020202020202020202020202020202020202020202")

	// Corrupt the recovery id to the legacy encoding inside the carrier.
	raw, err := snappy.Decode(nil, msg.Payload)
	require.NoError(t, err)
	require.Less(t, raw[64], byte(2))
	raw[64] += 27
	msg.Payload = snappy.Encode(nil, raw)

	decoded, err := decodeDirectSyncPayload(msg)
	require.NoError(t, err)
	require.Less(t, decoded.signature[64], byte(2))
	require.Less(t, decoded.raw[64], byte(2), "published bytes carry the raw recovery id")
}

// TestDecodeUnsafePayloadNotification exercises the websocket framing: only
// well-formed notifications of the right method with SIGNED payloads publish.
func TestDecodeUnsafePayloadNotification(t *testing.T) {
	logger := log.New()
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	msg, _ := buildDirectSyncMsg(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")

	frame := func(method string, m *directSyncMsg) []byte {
		result, err := json.Marshal(m)
		require.NoError(t, err)
		raw, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  map[string]any{"subscription": 1, "result": json.RawMessage(result)},
		})
		require.NoError(t, err)
		return raw
	}

	decoded, ok := decodeUnsafePayloadNotification(logger, frame(unsafePayloadNotifMethod, msg))
	require.True(t, ok)
	require.Equal(t, uint64(7), decoded.blockNumber())

	_, ok = decodeUnsafePayloadNotification(logger, frame("other_method", msg))
	require.False(t, ok)

	unsigned := *msg
	unsigned.Signed = false
	_, ok = decodeUnsafePayloadNotification(logger, frame(unsafePayloadNotifMethod, &unsigned))
	require.False(t, ok, "unsigned (cold-tier) payloads must never gossip")

	_, ok = decodeUnsafePayloadNotification(logger, []byte("not json"))
	require.False(t, ok)
}

// TestPublishGapForcesResubscribe: a block that does not extend the published
// sequence is NOT published; the caller resubscribes from the cursor so the
// producer's ring replays the hole.
func TestPublishGapForcesResubscribe(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	msg, _ := buildDirectSyncMsg(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")
	decoded, err := decodeDirectSyncPayload(msg)
	require.NoError(t, err)

	// Block 7 arrives while the cursor expects 3: gap. `out` is nil — the gap
	// branch must return before touching gossip.
	p := &payloadPublisher{log: log.New(), lastBlock: 2}
	require.True(t, p.publish(context.Background(), decoded))
	require.Equal(t, uint64(2), p.lastBlock)
	require.Equal(t, uint64(3), p.cursor())
}

// TestVerifyRequestIsTheUnifiedShape: one `{version, payload}` request whose
// payload is the canonical compressed envelope, for both node kinds.
func TestVerifyRequestIsTheUnifiedShape(t *testing.T) {
	payload := sampleV3Payload()
	beaconRoot := common.HexToHash("0x7777777777777777777777777777777777777777777777777777777777777777")
	msg, _ := buildDirectSyncMsg(t, payload, &beaconRoot,
		"0101010101010101010101010101010101010101010101010101010101010101")
	decoded, err := decodeDirectSyncPayload(msg)
	require.NoError(t, err)

	req, err := buildVerifyRequest(eth.BlockV3, decoded.signature, decoded.raw[65:])
	require.NoError(t, err)
	require.Equal(t, 3, req["version"])
	require.Len(t, req, 2, "unified shape carries version + payload only")

	compressed, err := hexutil.Decode(req["payload"].(string))
	require.NoError(t, err)
	raw, err := snappy.Decode(nil, compressed)
	require.NoError(t, err)
	require.Equal(t, decoded.raw, raw, "verify payload = the canonical sig||ssz bytes, compressed")
}
