package p2p

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/golang/snappy"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestGuardGossipValidator(t *testing.T) {
	logger := testlog.Logger(t, log.LevelCrit)
	val := guardGossipValidator(logger, func(ctx context.Context, id peer.ID, message *pubsub.Message) pubsub.ValidationResult {
		if id == "mallory" {
			panic("mallory was here")
		}
		if id == "bob" {
			return pubsub.ValidationIgnore
		}
		return pubsub.ValidationAccept
	})
	// Test that panics from mallory are recovered and rejected,
	// and test that we can continue to ignore bob and accept alice.
	require.Equal(t, pubsub.ValidationAccept, val(context.Background(), "alice", nil))
	require.Equal(t, pubsub.ValidationReject, val(context.Background(), "mallory", nil))
	require.Equal(t, pubsub.ValidationIgnore, val(context.Background(), "bob", nil))
	require.Equal(t, pubsub.ValidationReject, val(context.Background(), "mallory", nil))
	require.Equal(t, pubsub.ValidationAccept, val(context.Background(), "alice", nil))
	require.Equal(t, pubsub.ValidationIgnore, val(context.Background(), "bob", nil))
}

func TestCombinePeers(t *testing.T) {
	res := combinePeers([]peer.ID{"foo", "bar"}, []peer.ID{"bar", "baz"})
	require.Equal(t, []peer.ID{"foo", "bar", "baz"}, res)
}

func TestVerifyBlockSignature(t *testing.T) {
	logger := testlog.Logger(t, log.LevelCrit)
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	peerId := peer.ID("foo")
	secrets, err := crypto.GenerateKey()
	require.NoError(t, err)
	msg := []byte("any msg")

	t.Run("Valid", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.PublicKey)}
		signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
		sig, err := signer.SignBlockV1(context.Background(), eth.ChainIDFromBig(cfg.L2ChainID), opsigner.PayloadHash(msg))
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
		require.Equal(t, pubsub.ValidationAccept, result)
	})

	t.Run("WrongSigner", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: common.HexToAddress("0x1234")}
		signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
		sig, err := signer.SignBlockV1(context.Background(), eth.ChainIDFromBig(cfg.L2ChainID), opsigner.PayloadHash(msg))
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
		require.Equal(t, pubsub.ValidationReject, result)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.PublicKey)}
		sig := eth.Bytes65{}
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
		require.Equal(t, pubsub.ValidationReject, result)
	})

	t.Run("NoSequencer", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{}
		signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
		sig, err := signer.SignBlockV1(context.Background(), eth.ChainIDFromBig(cfg.L2ChainID), opsigner.PayloadHash(msg))
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
		require.Equal(t, pubsub.ValidationIgnore, result)
	})

	// Grace period tests: when the signer rotates, blocks from the previous signer
	// should still be accepted during the grace period.
	newSecrets, err := crypto.GenerateKey()
	require.NoError(t, err)
	thirdSecrets, err := crypto.GenerateKey()
	require.NoError(t, err)

	secretsAddr := crypto.PubkeyToAddress(secrets.PublicKey)
	newAddr := crypto.PubkeyToAddress(newSecrets.PublicKey)

	gracePeriodTests := []struct {
		name        string
		current     common.Address
		prev        common.Address
		signWith    *ecdsa.PrivateKey
		wantResult  pubsub.ValidationResult
		wantConfirm bool
	}{
		{"PreviousSignerAccepted", newAddr, secretsAddr, secrets, pubsub.ValidationAccept, false},
		{"NewSignerConfirms", newAddr, secretsAddr, newSecrets, pubsub.ValidationAccept, true},
		{"GracePeriodExpired", newAddr, common.Address{}, secrets, pubsub.ValidationReject, false},
		{"ThirdPartySignerRejected", newAddr, secretsAddr, thirdSecrets, pubsub.ValidationReject, false},
		{"ValidNoGracePeriod", secretsAddr, common.Address{}, secrets, pubsub.ValidationAccept, true},
	}

	for _, tc := range gracePeriodTests {
		t.Run(tc.name, func(t *testing.T) {
			runCfg := &testutils.MockRuntimeConfig{
				P2PSeqAddress:     tc.current,
				PrevP2PSeqAddress: tc.prev,
			}
			signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(tc.signWith)}
			sig, err := signer.SignBlockV1(context.Background(), eth.ChainIDFromBig(cfg.L2ChainID), opsigner.PayloadHash(msg))
			require.NoError(t, err)
			result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
			require.Equal(t, tc.wantResult, result)
			require.Equal(t, tc.wantConfirm, runCfg.Confirmed)
		})
	}
}

type MarshalSSZ interface {
	MarshalSSZ(w io.Writer) (n int, err error)
}

func createSignedP2Payload(payload MarshalSSZ, signer Signer, l2ChainID *big.Int) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(make([]byte, 65))
	if _, err := payload.MarshalSSZ(&buf); err != nil {
		return nil, fmt.Errorf("failed to encoded execution payload to publish: %w", err)
	}
	data := buf.Bytes()
	payloadData := data[65:]
	sig, err := signer.SignBlockV1(context.TODO(), eth.ChainIDFromBig(l2ChainID), opsigner.PayloadHash(payloadData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])

	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	return snappy.Encode(nil, data), nil
}

func createExecutionPayload(w types.Withdrawals, withdrawalsRoot *common.Hash, excessBlobGas, blobGasUsed *uint64) *eth.ExecutionPayload {
	return &eth.ExecutionPayload{
		Timestamp:       hexutil.Uint64(time.Now().Unix()),
		Withdrawals:     &w,
		WithdrawalsRoot: withdrawalsRoot,
		ExcessBlobGas:   (*eth.Uint64Quantity)(excessBlobGas),
		BlobGasUsed:     (*eth.Uint64Quantity)(blobGasUsed),
	}
}

func createEnvelope(h *common.Hash, w types.Withdrawals, withdrawalsRoot *common.Hash, excessBlobGas, blobGasUsed *uint64) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload:      createExecutionPayload(w, withdrawalsRoot, excessBlobGas, blobGasUsed),
		ParentBeaconBlockRoot: h,
	}
}

// TestBlockValidator does some very basic tests of the p2p block validation logic
func TestBlockValidator(t *testing.T) {
	// Params Set 1: Create the validation function
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	secrets, err := crypto.GenerateKey()
	require.NoError(t, err)
	runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.PublicKey)}
	signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
	// Params Set 2: Call the validation function
	peerID := peer.ID("foo")

	// Create a mock gossip configuration for testing
	mockGossipConf := &mockGossipSetupConfigurablesWithThreshold{threshold: 60 * time.Second}
	v2Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV2, mockGossipConf, clock.SystemClock)
	v3Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV3, mockGossipConf, clock.SystemClock)
	v4Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelDebug), cfg, runCfg, eth.BlockV4, mockGossipConf, clock.SystemClock)
	jovianCfg := &rollup.Config{
		L2ChainID:  big.NewInt(100),
		JovianTime: ptr.New(uint64(0)),
	}
	v4JovianValidator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), jovianCfg, runCfg, eth.BlockV4, mockGossipConf, clock.SystemClock)

	zero, one := uint64(0), uint64(1)
	beaconHash, withdrawalsRoot := common.HexToHash("0x1234"), common.HexToHash("0x9876")

	payloadTests := []struct {
		name      string
		validator pubsub.ValidatorEx
		result    pubsub.ValidationResult
		payload   *eth.ExecutionPayload
	}{
		{"V2Valid", v2Validator, pubsub.ValidationAccept, createExecutionPayload(types.Withdrawals{}, nil, nil, nil)},
		{"V2NonZeroWithdrawals", v2Validator, pubsub.ValidationReject, createExecutionPayload(types.Withdrawals{&types.Withdrawal{Index: 1, Validator: 1}}, nil, nil, nil)},
		{"V2NonZeroBlobProperties", v2Validator, pubsub.ValidationReject, createExecutionPayload(types.Withdrawals{}, nil, &zero, &zero)},
		{"V3RejectExecutionPayload", v3Validator, pubsub.ValidationReject, createExecutionPayload(types.Withdrawals{}, nil, &zero, &zero)},
	}

	for _, test := range payloadTests {
		t.Run(fmt.Sprintf("ExecutionPayload_%s", test.name), func(t *testing.T) {
			e := &eth.ExecutionPayloadEnvelope{ExecutionPayload: test.payload}
			test.payload.BlockHash, _ = e.CheckBlockHash() // hack to generate the block hash easily.
			data, err := createSignedP2Payload(test.payload, signer, cfg.L2ChainID)
			require.NoError(t, err)
			message := &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}
			res := test.validator(context.TODO(), peerID, message)
			require.Equal(t, res, test.result)
		})
	}

	envelopeTests := []struct {
		name      string
		validator pubsub.ValidatorEx
		result    pubsub.ValidationResult
		payload   *eth.ExecutionPayloadEnvelope
	}{
		{"V3RejectNonZeroExcessGas", v3Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &one, &zero)},
		{"V3RejectNonZeroBlobGasUsed", v3Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &one)},
		{"V3Valid", v3Validator, pubsub.ValidationAccept, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &zero)},
		{"V4Valid", v4Validator, pubsub.ValidationAccept, createEnvelope(&beaconHash, types.Withdrawals{}, &withdrawalsRoot, &zero, &zero)},
		{"V4RejectNoWithdrawalRoot", v4Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &zero)},
		{"V4AcceptNonZeroBlobGasUsedJovian", v4JovianValidator, pubsub.ValidationAccept, createEnvelope(&beaconHash, types.Withdrawals{}, &withdrawalsRoot, &zero, &one)},
		// Note: v3+ test cases with nil blobGasUsed cannot easily be included because they already fail at the SSZ marshaling stage.
	}

	for _, test := range envelopeTests {
		t.Run(fmt.Sprintf("ExecutionPayloadEnvelope_%s", test.name), func(t *testing.T) {
			test.payload.ExecutionPayload.BlockHash, _ = test.payload.CheckBlockHash() // hack to generate the block hash easily.
			data, err := createSignedP2Payload(test.payload, signer, cfg.L2ChainID)
			require.NoError(t, err)
			message := &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}
			res := test.validator(context.TODO(), peerID, message)
			require.Equal(t, res, test.result)
		})
	}
}

// TestGossipSnappyLengthPreamble pins which snappy frames the message-ID function accepts as valid
// snappy, in particular frames whose length preamble is padded with continuation bytes up to the
// ten bytes encoding/binary.Uvarint reads. An accepted frame gets the message ID of the minimal
// frame for its payload. kona-gossip's snappy_length_preamble_matches_op_node
// (rust/kona/crates/node/gossip/src/config.rs) pins the same vectors, so both nodes give every
// frame the same verdict and message ID.
func TestGossipSnappyLengthPreamble(t *testing.T) {
	const topic = "test"
	msgIDFn := BuildMsgIdFn(&rollup.Config{})
	msgID := func(frame []byte) string {
		return msgIDFn(&pubsub_pb.Message{Data: frame, Topic: ptr.New(topic)})
	}
	expectedMsgID := func(domain [4]byte, data []byte) string {
		h := sha256.New()
		h.Write(domain[:])
		var topicLen [8]byte
		binary.LittleEndian.PutUint64(topicLen[:], uint64(len(topic)))
		h.Write(topicLen[:])
		h.Write([]byte(topic))
		h.Write(data)
		return string(h.Sum(nil)[:20])
	}

	vectors := []struct {
		name  string
		frame string
		// decoded is the payload of an accepted frame, empty for a rejected one.
		decoded string
	}{
		{"Minimal", "051068656c6c6f", "68656c6c6f"},
		{"Preamble5Bytes", "85808080001068656c6c6f", "68656c6c6f"},
		{"Preamble6Bytes", "8580808080001068656c6c6f", "68656c6c6f"},
		{"Preamble10Bytes", "858080808080808080001068656c6c6f", "68656c6c6f"},
		{"Preamble11Bytes", "85808080808080808080001068656c6c6f", ""},
		{"Preamble10BytesOverflow", "858080808080808080021068656c6c6f", ""},
		{"Preamble10BytesAboveUint32", "858080808080808080011068656c6c6f", ""},
		{"Preamble5BytesAboveUint32", "80808080101068656c6c6f", ""},
		{"Preamble6BytesAboveUint32", "8080808090001068656c6c6f", ""},
		{"Empty", "", ""},
		{"LiteralLength1Byte", "01f00041", "41"},
		{"LiteralLength2Bytes", "01f4000041", "41"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			frame := common.FromHex(v.frame)
			original := bytes.Clone(frame)
			id := msgID(frame)
			require.Equal(t, original, frame, "the message-ID function must not modify the frame")
			if v.decoded == "" {
				require.Equal(t, expectedMsgID(MessageDomainInvalidSnappy, frame), id)
				return
			}
			decoded := common.FromHex(v.decoded)
			require.Equal(t, expectedMsgID(MessageDomainValidSnappy, decoded), id)
			require.Equal(t, msgID(snappy.Encode(nil, decoded)), id)
		})
	}
}

// padSnappyLengthPreamble returns frame with its snappy length preamble re-encoded in n bytes,
// padded with continuation bytes.
func padSnappyLengthPreamble(frame []byte, n int) []byte {
	v, m := binary.Uvarint(frame)
	out := make([]byte, 0, n+len(frame)-m)
	for range n - 1 {
		out = append(out, byte(v)|0x80)
		v >>= 7
	}
	return append(append(out, byte(v)), frame[m:]...)
}

// TestBlockValidatorPaddedSnappyLengthPreamble checks that the block validator accepts a signed
// block whose snappy length preamble is padded to ten bytes and rejects one padded to eleven,
// without modifying the message.
func TestBlockValidatorPaddedSnappyLengthPreamble(t *testing.T) {
	cfg := &rollup.Config{L2ChainID: big.NewInt(100)}
	secrets, err := crypto.GenerateKey()
	require.NoError(t, err)
	runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.PublicKey)}
	signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
	gossipConf := &mockGossipSetupConfigurablesWithThreshold{threshold: 60 * time.Second}
	validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV2, gossipConf, clock.SystemClock)

	payload := createExecutionPayload(types.Withdrawals{}, nil, nil, nil)
	payload.BlockHash, _ = (&eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}).CheckBlockHash()
	data, err := createSignedP2Payload(payload, signer, cfg.L2ChainID)
	require.NoError(t, err)

	validate := func(frame []byte) pubsub.ValidationResult {
		original := bytes.Clone(frame)
		res := validator(context.Background(), peer.ID("foo"), &pubsub.Message{Message: &pubsub_pb.Message{Data: frame}})
		require.Equal(t, original, frame, "the validator must not modify the frame")
		return res
	}
	require.Equal(t, pubsub.ValidationReject, validate(padSnappyLengthPreamble(data, 11)))
	require.Equal(t, pubsub.ValidationAccept, validate(padSnappyLengthPreamble(data, 10)))
}

// TestGossipTimestampThreshold tests that the configurable timestamp threshold works correctly
func TestGossipTimestampThreshold(t *testing.T) {
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	secrets, err := crypto.GenerateKey()
	require.NoError(t, err)
	runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.PublicKey)}
	signer := &PreparedSigner{Signer: opsigner.NewLocalSigner(secrets)}
	peerID := peer.ID("foo")

	// Test with different threshold values
	testCases := []struct {
		name           string
		threshold      time.Duration
		payloadTime    time.Time
		expectedResult pubsub.ValidationResult
	}{
		{
			name:           "AcceptWithinThreshold",
			threshold:      30 * time.Second,
			payloadTime:    time.Now().Add(-15 * time.Second), // 15 seconds ago
			expectedResult: pubsub.ValidationAccept,
		},
		{
			name:           "RejectOutsideThreshold",
			threshold:      30 * time.Second,
			payloadTime:    time.Now().Add(-45 * time.Second), // 45 seconds ago
			expectedResult: pubsub.ValidationReject,
		},
		{
			name:           "AcceptWithLongerThreshold",
			threshold:      120 * time.Second,
			payloadTime:    time.Now().Add(-90 * time.Second), // 90 seconds ago
			expectedResult: pubsub.ValidationAccept,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock config with the specific threshold
			mockConfig := &mockGossipSetupConfigurablesWithThreshold{threshold: tc.threshold}

			// Create validator with the mock config
			validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV2, mockConfig, clock.SystemClock)

			// Create payload with the specific timestamp
			payload := createExecutionPayload(types.Withdrawals{}, nil, nil, nil)
			payload.Timestamp = hexutil.Uint64(tc.payloadTime.Unix())
			// Set block number to avoid duplicate block validation issues
			payload.BlockNumber = hexutil.Uint64(time.Now().Unix())
			// Set a valid block hash for the payload
			payload.BlockHash, _ = (&eth.ExecutionPayloadEnvelope{ExecutionPayload: payload}).CheckBlockHash()

			// Create signed message
			data, err := createSignedP2Payload(payload, signer, cfg.L2ChainID)
			require.NoError(t, err)
			message := &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}

			// Test validation
			result := validator(context.Background(), peerID, message)
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

// mockGossipSetupConfigurablesWithThreshold implements GossipSetupConfigurables with configurable threshold
type mockGossipSetupConfigurablesWithThreshold struct {
	threshold time.Duration
}

func (m *mockGossipSetupConfigurablesWithThreshold) PeerScoringParams() *ScoringParams {
	return nil
}

func (m *mockGossipSetupConfigurablesWithThreshold) ConfigureGossip(rollupCfg *rollup.Config) []pubsub.Option {
	return nil
}

func (m *mockGossipSetupConfigurablesWithThreshold) GetGossipTimestampThreshold() time.Duration {
	return m.threshold
}
