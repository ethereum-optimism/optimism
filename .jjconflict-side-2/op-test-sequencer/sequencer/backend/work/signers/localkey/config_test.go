package localkey

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestConfig(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	// Create a key
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey)

	// select a chain
	chainID := eth.ChainIDFromUInt64(123)

	id := seqtypes.SignerID("foobar")
	opts := &work.ServiceOpts{
		StartOpts: &work.StartOpts{
			Log:     logger,
			Metrics: &metrics.NoopMetrics{},
		},
		Services: &work.Ensemble{},
	}

	t.Run("from-file", func(t *testing.T) {

		keyDir := t.TempDir()
		keyPath := filepath.Join(keyDir, "key.txt")
		require.NoError(t, crypto.SaveECDSA(keyPath, key))
		loaded, err := crypto.LoadECDSA(keyPath)
		require.NoError(t, err)
		require.Equal(t, addr, crypto.PubkeyToAddress(loaded.PublicKey))

		cfg := &Config{
			ChainID: chainID,
			KeyPath: keyPath,
			RawKey:  nil,
		}
		signer, err := cfg.Start(context.Background(), id, opts)
		require.NoError(t, err)
		testSigner(t, signer, chainID, addr)
	})

	t.Run("from-raw", func(t *testing.T) {
		rawKey := crypto.FromECDSA(key)
		cfg := &Config{
			ChainID: chainID,
			KeyPath: "",
			RawKey:  (*hexutil.Bytes)(&rawKey),
		}
		signer, err := cfg.Start(context.Background(), id, opts)
		require.NoError(t, err)
		testSigner(t, signer, chainID, addr)
	})
}

func testSigner(t *testing.T, signer work.Signer, chainID eth.ChainID, addr common.Address) {
	require.Equal(t, chainID, signer.ChainID(), "sanity-check chain ID")

	parentBeaconBlockRoot := common.Hash{0x42}
	block := &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &parentBeaconBlockRoot,
		ExecutionPayload:      &eth.ExecutionPayload{BlockNumber: 1234},
	}
	// Sign a test payload
	signedBlock, err := signer.Sign(context.Background(), block)
	require.NoError(t, err)

	authCtx := &opsigner.OPStackP2PBlockAuthV1{Allowed: addr, Chain: chainID}
	require.NoError(t, signedBlock.VerifySignature(authCtx))

	otherKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	otherAddr := crypto.PubkeyToAddress(otherKey.PublicKey)
	authCtx2 := &opsigner.OPStackP2PBlockAuthV1{Allowed: otherAddr, Chain: chainID}
	require.Error(t, signedBlock.VerifySignature(authCtx2), "sanity check that other address is not also valid")
}
