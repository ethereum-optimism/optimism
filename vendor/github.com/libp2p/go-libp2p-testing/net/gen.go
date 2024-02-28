package tnet

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/libp2p/go-libp2p-testing/etc"
	ci "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/test"

	ma "github.com/multiformats/go-multiaddr"
)

// ZeroLocalTCPAddress is the "zero" tcp local multiaddr. This means:
//   /ip4/127.0.0.1/tcp/0
var ZeroLocalTCPAddress, _ = ma.NewMultiaddr("/ip4/127.0.0.1/tcp/0")

// RandLocalTCPAddress returns a random multiaddr. it suppresses errors
// for nice composability-- do check the address isn't nil.
//
// NOTE: for real network tests, use ZeroLocalTCPAddress so the kernel
// assigns an unused TCP port. otherwise you may get clashes. This
// function remains here so that p2p/net/mock (which does not touch the
// real network) can assign different addresses to peers.
func RandLocalTCPAddress() ma.Multiaddr {
	// chances are it will work out, but it **might** fail if the port is in use
	// most ports above 10000 aren't in use by long running processes, so yay.
	// (maybe there should be a range of "loopback" ports that are guaranteed
	// to be open for the process, but naturally can only talk to self.)

	lastPort.Lock()
	if lastPort.port == 0 {
		lastPort.port = 10000 + tetc.SeededRand.Intn(50000)
	}
	port := lastPort.port
	lastPort.port++
	lastPort.Unlock()

	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port)
	maddr, _ := ma.NewMultiaddr(addr)
	return maddr
}

var lastPort = struct {
	port int
	sync.Mutex
}{}

// PeerNetParams is a struct to bundle together the four things
// you need to run a connection with a peer: id, 2keys, and addr.
type PeerNetParams struct {
	ID      peer.ID
	PrivKey ci.PrivKey
	PubKey  ci.PubKey
	Addr    ma.Multiaddr
}

func (p *PeerNetParams) checkKeys() error {
	if !p.ID.MatchesPrivateKey(p.PrivKey) {
		return errors.New("p.ID does not match p.PrivKey")
	}

	if !p.ID.MatchesPublicKey(p.PubKey) {
		return errors.New("p.ID does not match p.PubKey")
	}

	buf := new(bytes.Buffer)
	buf.Write([]byte("hello world. this is me, I swear."))
	b := buf.Bytes()

	sig, err := p.PrivKey.Sign(b)
	if err != nil {
		return fmt.Errorf("sig signing failed: %s", err)
	}

	sigok, err := p.PubKey.Verify(b, sig)
	if err != nil {
		return fmt.Errorf("sig verify failed: %s", err)
	}
	if !sigok {
		return fmt.Errorf("sig verify failed: sig invalid")
	}

	return nil // ok. move along.
}

func RandPeerNetParamsOrFatal(t *testing.T) PeerNetParams {
	p, err := RandPeerNetParams()
	if err != nil {
		t.Fatal(err)
		return PeerNetParams{} // TODO return nil
	}
	return *p
}

func RandPeerNetParams() (*PeerNetParams, error) {
	var p PeerNetParams
	var err error
	p.Addr = ZeroLocalTCPAddress
	p.PrivKey, p.PubKey, err = test.RandTestKeyPair(ci.Ed25519, 0)
	if err != nil {
		return nil, err
	}
	p.ID, err = peer.IDFromPublicKey(p.PubKey)
	if err != nil {
		return nil, err
	}
	if err := p.checkKeys(); err != nil {
		return nil, err
	}
	return &p, nil
}
