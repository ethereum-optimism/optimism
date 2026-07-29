package p2p

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestPrivPub2PeerID(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Secp256k1, 32)
	require.NoError(t, err)
	privRaw, err := priv.Raw()
	require.NoError(t, err)
	pubRaw, err := pub.Raw()
	require.NoError(t, err)

	t.Run("with a private key", func(t *testing.T) {
		privPidLib, err := peer.IDFromPrivateKey(priv)
		require.NoError(t, err)
		privPidImpl, err := Priv2PeerID(bytes.NewReader([]byte(hex.EncodeToString(privRaw))))
		require.NoError(t, err)
		require.Equal(t, privPidLib.String(), privPidImpl)
	})
	t.Run("with a public key", func(t *testing.T) {
		pubPidLib, err := peer.IDFromPublicKey(pub)
		require.NoError(t, err)
		pubPidImpl, err := Pub2PeerID(bytes.NewReader([]byte(hex.EncodeToString(pubRaw))))
		require.NoError(t, err)
		require.Equal(t, pubPidLib.String(), pubPidImpl)
	})
	t.Run("with bad hex", func(t *testing.T) {
		_, err := Priv2PeerID(bytes.NewReader([]byte("I am not hex.")))
		require.Error(t, err)
	})
}
