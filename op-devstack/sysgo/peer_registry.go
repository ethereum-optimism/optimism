package sysgo

import "sync"

// PeerRegistrar lets connect helpers record a re-connect closure on a node so
// the same peering is re-established whenever the node (re)starts. Nodes that
// implement this MUST replay registered connectors once they finish coming up.
type PeerRegistrar interface {
	RegisterPeerConnector(connect func())
}

// peerRegistry is the embeddable state behind RegisterPeerConnector.
type peerRegistry struct {
	mu         sync.Mutex
	connectors []func()
}

func (r *peerRegistry) RegisterPeerConnector(connect func()) {
	r.mu.Lock()
	r.connectors = append(r.connectors, connect)
	r.mu.Unlock()
}

func (r *peerRegistry) snapshotConnectors() []func() {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]func(), len(r.connectors))
	copy(out, r.connectors)
	return out
}
