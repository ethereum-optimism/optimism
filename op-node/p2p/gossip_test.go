package p2p

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/golang/snappy"

	// "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestGuardGossipValidator(t *testing.T) {
	logger := testlog.Logger(t, log.LvlCrit)
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
	logger := testlog.Logger(t, log.LvlCrit)
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	peerId := peer.ID("foo")
	secrets, err := e2eutils.DefaultMnemonicConfig.Secrets()
	require.NoError(t, err)
	msg := []byte("any msg")

	t.Run("Valid", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.SequencerP2P.PublicKey)}
		signer := &PreparedSigner{Signer: NewLocalSigner(secrets.SequencerP2P)}
		sig, err := signer.Sign(context.Background(), SigningDomainBlocksV1, cfg.L2ChainID, msg)
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig[:65], msg)
		require.Equal(t, pubsub.ValidationAccept, result)
	})

	t.Run("WrongSigner", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: common.HexToAddress("0x1234")}
		signer := &PreparedSigner{Signer: NewLocalSigner(secrets.SequencerP2P)}
		sig, err := signer.Sign(context.Background(), SigningDomainBlocksV1, cfg.L2ChainID, msg)
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig[:65], msg)
		require.Equal(t, pubsub.ValidationReject, result)
	})

	t.Run("InvalidSignature", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.SequencerP2P.PublicKey)}
		sig := make([]byte, 65)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig, msg)
		require.Equal(t, pubsub.ValidationReject, result)
	})

	t.Run("NoSequencer", func(t *testing.T) {
		runCfg := &testutils.MockRuntimeConfig{}
		signer := &PreparedSigner{Signer: NewLocalSigner(secrets.SequencerP2P)}
		sig, err := signer.Sign(context.Background(), SigningDomainBlocksV1, cfg.L2ChainID, msg)
		require.NoError(t, err)
		result := verifyBlockSignature(logger, cfg, runCfg, peerId, sig[:65], msg)
		require.Equal(t, pubsub.ValidationIgnore, result)
	})
}

func createSignedP2Payload(payload *eth.ExecutionPayload, signer Signer, l2ChainID *big.Int) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write(make([]byte, 65))
	if _, err := payload.MarshalSSZ(&buf); err != nil {
		return nil, fmt.Errorf("failed to encoded execution payload to publish: %w", err)
	}
	data := buf.Bytes()
	payloadData := data[65:]
	sig, err := signer.Sign(context.TODO(), SigningDomainBlocksV1, l2ChainID, payloadData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign execution payload with signer: %w", err)
	}
	copy(data[:65], sig[:])

	// compress the full message
	// This also copies the data, freeing up the original buffer to go back into the pool
	return snappy.Encode(nil, data), nil
}

// TestBlockValidator does some very basic tests of the p2p block validation logic
func TestBlockValidator(t *testing.T) {
	// Params Set 1: Create the validation function
	cfg := &rollup.Config{
		L2ChainID: big.NewInt(100),
	}
	secrets, err := e2eutils.DefaultMnemonicConfig.Secrets()
	require.NoError(t, err)
	runCfg := &testutils.MockRuntimeConfig{P2PSeqAddress: crypto.PubkeyToAddress(secrets.SequencerP2P.PublicKey)}
	signer := &PreparedSigner{Signer: NewLocalSigner(secrets.SequencerP2P)}

	// valFnV1 := BuildBlocksValidator(testlog.Logger(t, log.LvlCrit), rollupCfg, runCfg, eth.BlockV1)
	valFnV2 := BuildBlocksValidator(testlog.Logger(t, log.LvlCrit), cfg, runCfg, eth.BlockV2)

	// Params Set 2: Call the validation function
	peerID := peer.ID("foo")

	// Valid Case
	payload := eth.ExecutionPayload{
		Timestamp:   hexutil.Uint64(time.Now().Unix()),
		Withdrawals: &types.Withdrawals{},
	}
	payload.BlockHash, _ = payload.CheckBlockHash() // hack to generate the block hash easily.
	data, err := createSignedP2Payload(&payload, signer, cfg.L2ChainID)
	require.NoError(t, err)
	message := &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}
	res := valFnV2(context.TODO(), peerID, message)
	require.Equal(t, res, pubsub.ValidationAccept)

	// Invalid because non-empty withdrawals when Canyon is active
	payload = eth.ExecutionPayload{
		Timestamp:   hexutil.Uint64(time.Now().Unix()),
		Withdrawals: &types.Withdrawals{&types.Withdrawal{Index: 1, Validator: 1}},
	}
	payload.BlockHash, _ = payload.CheckBlockHash()
	data, err = createSignedP2Payload(&payload, signer, cfg.L2ChainID)
	require.NoError(t, err)
	message = &pubsub.Message{Message: &pubsub_pb.Message{Data: data}}
	res = valFnV2(context.TODO(), peerID, message)
	require.Equal(t, res, pubsub.ValidationReject)

}
