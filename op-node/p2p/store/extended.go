package store

import (
	"context"
	"errors"
	"fmt"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type extendedStore struct {
	peerstore.Peerstore
	peerstore.CertifiedAddrBook
	*scoreBook
}

func NewExtendedPeerstore(ctx context.Context, ps peerstore.Peerstore, store ds.Batching) (ExtendedPeerstore, error) {
	cab, ok := peerstore.GetCertifiedAddrBook(ps)
	if !ok {
		return nil, errors.New("peerstore should also be a certified address book")
	}
	sb, err := newScoreBook(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("create scorebook: %w", err)
	}
	return &extendedStore{
		Peerstore:         ps,
		CertifiedAddrBook: cab,
		scoreBook:         sb,
	}, nil
}

var _ ExtendedPeerstore = (*extendedStore)(nil)
