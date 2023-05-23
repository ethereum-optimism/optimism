package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/log"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type extendedStore struct {
	peerstore.Peerstore
	peerstore.CertifiedAddrBook
	*scoreBook
}

func NewExtendedPeerstore(ctx context.Context, logger log.Logger, clock clock.Clock, ps peerstore.Peerstore, store ds.Batching) (ExtendedPeerstore, error) {
	cab, ok := peerstore.GetCertifiedAddrBook(ps)
	if !ok {
		return nil, errors.New("peerstore should also be a certified address book")
	}
	sb, err := newScoreBook(ctx, logger, clock, store)
	if err != nil {
		return nil, fmt.Errorf("create scorebook: %w", err)
	}
	sb.startGC()
	return &extendedStore{
		Peerstore:         ps,
		CertifiedAddrBook: cab,
		scoreBook:         sb,
	}, nil
}

func (s *extendedStore) Close() error {
	s.scoreBook.Close()
	return s.Peerstore.Close()
}

var _ ExtendedPeerstore = (*extendedStore)(nil)
