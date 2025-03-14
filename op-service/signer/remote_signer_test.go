package signer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockRemoteSigner struct {
	priv *ecdsa.PrivateKey
	err  error
}

func (t *mockRemoteSigner) SignBlockPayload(args BlockPayloadArgs) (hexutil.Bytes, error) {
	if t.err != nil {
		return nil, t.err
	}
	msg, err := args.Message()
	if err != nil {
		return nil, err
	}
	signingHash := msg.ToSigningHash()
	signature, err := crypto.Sign(signingHash[:], t.priv)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func TestRemoteSigner(t *testing.T) {
	secret, err := crypto.GenerateKey()
	require.NoError(t, err)

	remoteSigner := &mockRemoteSigner{priv: secret, err: nil}
	server := oprpc.NewServer(
		"127.0.0.1",
		0,
		"test",
	)
	server.AddAPI(rpc.API{
		Namespace: "opsigner",
		Service:   remoteSigner,
	})

	require.NoError(t, server.Start())
	defer func() {
		_ = server.Stop()
	}()

	logger := testlog.Logger(t, log.LevelCrit)

	msg := []byte("any msg")
	payloadHash := PayloadHash(msg)

	signerCfg := NewCLIConfig()
	signerCfg.Endpoint = fmt.Sprintf("http://%s", server.Endpoint())
	signerCfg.TLSConfig.TLSKey = ""
	signerCfg.TLSConfig.TLSCert = ""
	signerCfg.TLSConfig.TLSCaCert = ""
	signerCfg.TLSConfig.Enabled = false

	must := func(fn func() error) func() {
		return func() {
			require.NoError(t, fn())
		}
	}
	addr := crypto.PubkeyToAddress(secret.PublicKey)
	chainID := eth.ChainIDFromUInt64(100)

	t.Run("Valid", func(t *testing.T) {
		remote, err := NewRemoteSigner(logger, signerCfg)
		require.NoError(t, err)
		t.Cleanup(must(remote.Close))
		sig, err := remote.SignBlockV1(context.Background(), chainID, payloadHash)
		require.NoError(t, err)
		authCtx := &OPStackP2PBlockAuthV1{
			Allowed: addr,
			Chain:   chainID,
		}
		require.NoError(t, authCtx.VerifyP2PBlockSignature(payloadHash, sig))
	})
	t.Run("RPC err", func(t *testing.T) {
		remote, err := NewRemoteSigner(logger, signerCfg)
		require.NoError(t, err)
		t.Cleanup(must(remote.Close))
		testErr := &rpc.JsonError{Code: -39000, Message: "test error"}
		remoteSigner.err = testErr
		_, err = remote.SignBlockV1(context.Background(), chainID, payloadHash)
		var rpcErr rpc.Error
		require.True(t, errors.As(err, &rpcErr))
		require.Equal(t, -39000, rpcErr.ErrorCode())
		remoteSigner.err = nil
	})

	t.Run("RemoteSignerNoTLS", func(t *testing.T) {
		signerCfg := NewCLIConfig()
		signerCfg.Endpoint = fmt.Sprintf("http://%s", server.Endpoint())
		signerCfg.TLSConfig.TLSKey = "invalid"
		signerCfg.TLSConfig.TLSCert = "invalid"
		signerCfg.TLSConfig.TLSCaCert = "invalid"
		signerCfg.TLSConfig.Enabled = true

		_, err := NewRemoteSigner(logger, signerCfg)
		require.Error(t, err)
	})

	t.Run("RemoteSignerInvalidEndpoint", func(t *testing.T) {
		signerCfg := NewCLIConfig()
		signerCfg.Endpoint = "Invalid"
		signerCfg.TLSConfig.TLSKey = ""
		signerCfg.TLSConfig.TLSCert = ""
		signerCfg.TLSConfig.TLSCaCert = ""
		_, err := NewRemoteSigner(logger, signerCfg)
		require.Error(t, err)
	})
}
