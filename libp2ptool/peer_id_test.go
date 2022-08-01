package libp2ptool

import (
	"bytes"
	"encoding/hex"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"testing"
)

func TestReadPeerID(t *testing.T) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Secp256k1, 32)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	privRaw, err := priv.Raw()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	pubRaw, err := pub.Raw()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	t.Run("with a private key", func(t *testing.T) {
		privPidLib, err := peer.IDFromPrivateKey(priv)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		privPidImpl, err := ReadPeerID(true, bytes.NewReader([]byte(hex.EncodeToString(privRaw))))
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if privPidImpl != privPidLib.String() {
			t.Fatalf("expected %s to equal %s", privPidImpl, privPidLib.String())
		}
	})
	t.Run("with a public key", func(t *testing.T) {
		pubPidLib, err := peer.IDFromPublicKey(pub)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		pubPidImpl, err := ReadPeerID(false, bytes.NewReader([]byte(hex.EncodeToString(pubRaw))))
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		if pubPidImpl != pubPidLib.String() {
			t.Fatalf("expected %s to equal %s", pubPidImpl, pubPidLib.String())
		}
	})
}
