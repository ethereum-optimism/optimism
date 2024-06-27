package source

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const waitDuration = 10 * time.Second
const checkInterval = 10 * time.Millisecond

func TestUnsafeHeadUpdates(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1337))
	header1 := testutils.RandomHeader(rng)
	header2 := testutils.RandomHeader(rng)

	t.Run("NotifyOfNewHeads", func(t *testing.T) {
		rpc, callback := startHeadMonitor(t)

		rpc.NewUnsafeHead(t, header1)
		callback.RequireUnsafeHeaders(t, header1)

		rpc.NewUnsafeHead(t, header2)
		callback.RequireUnsafeHeaders(t, header1, header2)
	})

	t.Run("ResubscribeOnError", func(t *testing.T) {
		rpc, callback := startHeadMonitor(t)

		rpc.SubscriptionError(t)

		rpc.NewUnsafeHead(t, header1)
		callback.RequireUnsafeHeaders(t, header1)
	})
}

func TestSafeHeadUpdates(t *testing.T) {
	rpc, callback := startHeadMonitor(t)

	head1 := eth.L1BlockRef{
		Hash:   common.Hash{0xaa},
		Number: 1,
	}
	head2 := eth.L1BlockRef{
		Hash:   common.Hash{0xbb},
		Number: 2,
	}

	rpc.SetSafeHead(head1)
	callback.RequireSafeHeaders(t, head1)
	rpc.SetSafeHead(head2)
	callback.RequireSafeHeaders(t, head1, head2)
}

func TestFinalizedHeadUpdates(t *testing.T) {
	rpc, callback := startHeadMonitor(t)

	head1 := eth.L1BlockRef{
		Hash:   common.Hash{0xaa},
		Number: 1,
	}
	head2 := eth.L1BlockRef{
		Hash:   common.Hash{0xbb},
		Number: 2,
	}

	rpc.SetFinalizedHead(head1)
	callback.RequireFinalizedHeaders(t, head1)
	rpc.SetFinalizedHead(head2)
	callback.RequireFinalizedHeaders(t, head1, head2)
}

func startHeadMonitor(t *testing.T) (*stubRPC, *stubCallback) {
	logger := testlog.Logger(t, log.LvlInfo)
	rpc := &stubRPC{}
	callback := &stubCallback{}
	monitor := NewHeadMonitor(logger, 50*time.Millisecond, rpc, callback)
	require.NoError(t, monitor.Start())
	t.Cleanup(func() {
		require.NoError(t, monitor.Stop())
	})
	return rpc, callback
}

type stubCallback struct {
	sync.Mutex
	unsafe    []eth.L1BlockRef
	safe      []eth.L1BlockRef
	finalized []eth.L1BlockRef
}

func (s *stubCallback) RequireUnsafeHeaders(t *testing.T, heads ...*types.Header) {
	expected := make([]eth.L1BlockRef, len(heads))
	for i, head := range heads {
		expected[i] = eth.InfoToL1BlockRef(eth.HeaderBlockInfo(head))
	}
	s.requireHeaders(t, func(s *stubCallback) []eth.L1BlockRef { return s.unsafe }, expected)
}

func (s *stubCallback) RequireSafeHeaders(t *testing.T, expected ...eth.L1BlockRef) {
	s.requireHeaders(t, func(s *stubCallback) []eth.L1BlockRef { return s.safe }, expected)
}

func (s *stubCallback) RequireFinalizedHeaders(t *testing.T, expected ...eth.L1BlockRef) {
	s.requireHeaders(t, func(s *stubCallback) []eth.L1BlockRef { return s.finalized }, expected)
}

func (s *stubCallback) requireHeaders(t *testing.T, getter func(*stubCallback) []eth.L1BlockRef, expected []eth.L1BlockRef) {
	require.Eventually(t, func() bool {
		s.Lock()
		defer s.Unlock()
		return len(getter(s)) >= len(expected)
	}, waitDuration, checkInterval)
	s.Lock()
	defer s.Unlock()
	require.Equal(t, expected, getter(s))
}

func (s *stubCallback) OnNewUnsafeHead(ctx context.Context, block eth.L1BlockRef) {
	s.Lock()
	defer s.Unlock()
	s.unsafe = append(s.unsafe, block)
}

func (s *stubCallback) OnNewSafeHead(ctx context.Context, block eth.L1BlockRef) {
	s.Lock()
	defer s.Unlock()
	s.safe = append(s.safe, block)
}

func (s *stubCallback) OnNewFinalizedHead(ctx context.Context, block eth.L1BlockRef) {
	s.Lock()
	defer s.Unlock()
	s.finalized = append(s.finalized, block)
}

var _ HeadChangeCallback = (*stubCallback)(nil)

type stubRPC struct {
	sync.Mutex
	sub *mockSubscription

	safeHead      eth.L1BlockRef
	finalizedHead eth.L1BlockRef
}

func (s *stubRPC) SubscribeNewHead(_ context.Context, unsafeCh chan<- *types.Header) (ethereum.Subscription, error) {
	s.Lock()
	defer s.Unlock()
	if s.sub != nil {
		return nil, errors.New("already subscribed to unsafe heads")
	}
	errChan := make(chan error)
	s.sub = &mockSubscription{errChan, unsafeCh, s}
	return s.sub, nil
}

func (s *stubRPC) SetSafeHead(head eth.L1BlockRef) {
	s.Lock()
	defer s.Unlock()
	s.safeHead = head
}

func (s *stubRPC) SetFinalizedHead(head eth.L1BlockRef) {
	s.Lock()
	defer s.Unlock()
	s.finalizedHead = head
}

func (s *stubRPC) L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	s.Lock()
	defer s.Unlock()
	switch label {
	case eth.Safe:
		if s.safeHead == (eth.L1BlockRef{}) {
			return eth.L1BlockRef{}, errors.New("no unsafe head")
		}
		return s.safeHead, nil
	case eth.Finalized:
		if s.finalizedHead == (eth.L1BlockRef{}) {
			return eth.L1BlockRef{}, errors.New("no finalized head")
		}
		return s.finalizedHead, nil
	default:
		return eth.L1BlockRef{}, fmt.Errorf("unknown label: %v", label)
	}
}

func (s *stubRPC) NewUnsafeHead(t *testing.T, header *types.Header) {
	s.WaitForSub(t)
	s.Lock()
	defer s.Unlock()
	require.NotNil(t, s.sub, "Attempting to publish a header with no subscription")
	s.sub.headers <- header
}

func (s *stubRPC) SubscriptionError(t *testing.T) {
	s.WaitForSub(t)
	s.Lock()
	defer s.Unlock()
	s.sub.errChan <- errors.New("subscription error")
	s.sub = nil
}

func (s *stubRPC) WaitForSub(t *testing.T) {
	require.Eventually(t, func() bool {
		s.Lock()
		defer s.Unlock()
		return s.sub != nil
	}, waitDuration, checkInterval, "Head monitor did not subscribe to unsafe head")
}

var _ HeadMonitorClient = (*stubRPC)(nil)

type mockSubscription struct {
	errChan chan error
	headers chan<- *types.Header
	rpc     *stubRPC
}

func (m *mockSubscription) Unsubscribe() {
	fmt.Println("Unsubscribed")
	m.rpc.Lock()
	defer m.rpc.Unlock()
	m.rpc.sub = nil
}

func (m *mockSubscription) Err() <-chan error {
	return m.errChan
}
