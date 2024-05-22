package test

import (
	"crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"

	mh "github.com/multiformats/go-multihash"
)

func RandPeerID() (peer.ID, error) {
	buf := make([]byte, 16)
	rand.Read(buf)
	h, _ := mh.Sum(buf, mh.SHA2_256, -1)
	return peer.ID(h), nil
}

func RandPeerIDFatal(t testing.TB) peer.ID {
	p, err := RandPeerID()
	if err != nil {
		t.Fatal(err)
	}
	return p
}
