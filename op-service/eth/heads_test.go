package eth

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"
)

type fixedHeadSource struct{ header *types.Header }

func (s fixedHeadSource) SubscribeNewHead(_ context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		select {
		case ch <- s.header:
		case <-quit:
			return nil
		}
		<-quit
		return nil
	}), nil
}

func TestWatchHeadChanges(t *testing.T) {
	watch := func(t *testing.T, header *types.Header) (L1BlockRef, error) {
		signals := make(chan L1BlockRef, 1)
		sub, err := WatchHeadChanges(context.Background(), fixedHeadSource{header}, func(_ context.Context, sig L1BlockRef) {
			signals <- sig
		})
		require.NoError(t, err)
		defer sub.Unsubscribe()
		select {
		case sig := <-signals:
			return sig, nil
		case err := <-sub.Err():
			return L1BlockRef{}, err
		case <-time.After(5 * time.Second):
			t.Fatal("no head signal or subscription error")
			return L1BlockRef{}, nil
		}
	}

	t.Run("valid", func(t *testing.T) {
		header := &types.Header{Number: big.NewInt(7), Time: 42}
		sig, err := watch(t, header)
		require.NoError(t, err)
		require.Equal(t, *BlockRefFromHeader(header), sig)
	})

	invalid := map[string]*types.Header{
		"nil header":       nil,
		"nil number":       {},
		"number above u64": {Number: new(big.Int).Lsh(big.NewInt(1), 64)},
	}
	for name, header := range invalid {
		t.Run(name, func(t *testing.T) {
			_, err := watch(t, header)
			require.Error(t, err)
		})
	}
}
