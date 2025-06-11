package p2p

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/golang/snappy"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

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

func createExecutionPayload(w types.Withdrawals, withdrawalsRoot *common.Hash, excessGas, gasUsed *uint64) *eth.ExecutionPayload {
	return &eth.ExecutionPayload{
		Timestamp:       hexutil.Uint64(time.Now().Unix()),
		Withdrawals:     &w,
		WithdrawalsRoot: withdrawalsRoot,
		ExcessBlobGas:   (*eth.Uint64Quantity)(excessGas),
		BlobGasUsed:     (*eth.Uint64Quantity)(gasUsed),
	}
}

func createEnvelope(h *common.Hash, w types.Withdrawals, withdrawalsRoot *common.Hash, excessGas, gasUsed *uint64, requestsHash *common.Hash) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload:      createExecutionPayload(w, withdrawalsRoot, excessGas, gasUsed),
		ParentBeaconBlockRoot: h,
		RequestsHash:          requestsHash,
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

	v2Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV2)
	v3Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV3)
	v4Validator := BuildBlocksValidator(testlog.Logger(t, log.LevelCrit), cfg, runCfg, eth.BlockV4)

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

	for _, tt := range payloadTests {
		test := tt
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
		{"V3RejectNonZeroExcessGas", v3Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &one, &zero, nil)},
		{"V3RejectNonZeroBlobGasUsed", v3Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &one, nil)},
		{"V3Valid", v3Validator, pubsub.ValidationAccept, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &zero, nil)},
		{"V4Valid", v4Validator, pubsub.ValidationAccept, createEnvelope(&beaconHash, types.Withdrawals{}, &withdrawalsRoot, &zero, &zero, &types.EmptyRequestsHash)},
		{"V4RejectNoWithdrawalRoot", v4Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, nil, &zero, &zero, &types.EmptyRequestsHash)},
		{"V4RejectNoRequestsHash", v4Validator, pubsub.ValidationReject, createEnvelope(&beaconHash, types.Withdrawals{}, &common.Hash{}, &zero, &zero, nil)},
	}

	for _, tt := range envelopeTests {
		test := tt
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
