package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
	"github.com/ethereum-optimism/optimism/op-service/backoff"

	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var ErrNoUnsafeL2PayloadChannel = errors.New("unsafeL2Payloads channel must not be nil")

// RpcSyncPeer is a mock PeerID for the RPC sync client.
var RpcSyncPeer peer.ID = "ALT_RPC_SYNC"

// receivePayload queues the received payload for processing.
// This may return an error if there's no capacity for the payload.
type receivePayload = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error

type RPCSync interface {
	io.Closer
	// Start starts an additional worker syncing job
	Start() error
	// RequestL2Range signals that the given range should be fetched, implementing the alt-sync interface.
	RequestL2Range(ctx context.Context, start uint64, end eth.L2BlockRef) error
}

// SyncClient implements the driver AltSync interface, including support for fetching an open-ended chain of L2 blocks.
type SyncClient struct {
	*L2Client

	requests chan uint64

	resCtx    context.Context
	resCancel context.CancelFunc

	receivePayload receivePayload
	wg             sync.WaitGroup
}

type SyncClientConfig struct {
	L2ClientConfig
}

func SyncClientDefaultConfig(config *rollup.Config, trustRPC bool) *SyncClientConfig {
	return &SyncClientConfig{
		*L2ClientDefaultConfig(config, trustRPC),
	}
}

func NewSyncClient(receiver receivePayload, client client.RPC, log log.Logger, metrics caching.Metrics, config *SyncClientConfig) (*SyncClient, error) {
	l2Client, err := NewL2Client(client, log, metrics, &config.L2ClientConfig)
	if err != nil {
		return nil, err
	}
	// This resource context is shared between all workers that may be started
	resCtx, resCancel := context.WithCancel(context.Background())
	return &SyncClient{
		L2Client:       l2Client,
		resCtx:         resCtx,
		resCancel:      resCancel,
		requests:       make(chan uint64, 128),
		receivePayload: receiver,
	}, nil
}

// Start starts the syncing background work. This may not be called after Close().
func (s *SyncClient) Start() error {
	// TODO(CLI-3635): we can start multiple event loop runners as workers, to parallelize the work
	s.wg.Add(1)
	go s.eventLoop()
	return nil
}

// Close sends a signal to close all concurrent syncing work.
func (s *SyncClient) Close() error {
	s.resCancel()
	s.wg.Wait()
	return nil
}

func (s *SyncClient) RequestL2Range(ctx context.Context, start, end eth.L2BlockRef) error {
	// Drain previous requests now that we have new information
	for len(s.requests) > 0 {
		select { // in case requests is being read at the same time, don't block on draining it.
		case <-s.requests:
		default:
			break
		}
	}

	endNum := end.Number
	if end == (eth.L2BlockRef{}) {
		n, err := s.rollupCfg.TargetBlockNumber(uint64(time.Now().Unix()))
		if err != nil {
			return err
		}
		if n <= start.Number {
			return nil
		}
		endNum = n
	}

	// TODO(CLI-3635): optimize the by-range fetching with the Engine API payloads-by-range method.

	s.log.Info("Scheduling to fetch trailing missing payloads from backup RPC", "start", start, "end", endNum, "size", endNum-start.Number-1)

	for i := start.Number + 1; i < endNum; i++ {
		select {
		case s.requests <- i:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// eventLoop is the main event loop for the sync client.
func (s *SyncClient) eventLoop() {
	defer s.wg.Done()
	s.log.Info("Starting sync client event loop")

	backoffStrategy := &backoff.ExponentialStrategy{
		Min:       1000,
		Max:       20_000,
		MaxJitter: 250,
	}

	for {
		select {
		case <-s.resCtx.Done():
			s.log.Debug("Shutting down RPC sync worker")
			return
		case reqNum := <-s.requests:
			err := backoff.DoCtx(s.resCtx, 5, backoffStrategy, func() error {
				// Limit the maximum time for fetching payloads
				ctx, cancel := context.WithTimeout(s.resCtx, time.Second*10)
				defer cancel()
				// We are only fetching one block at a time here.
				return s.fetchUnsafeBlockFromRpc(ctx, reqNum)
			})
			if err != nil {
				if err == s.resCtx.Err() {
					return
				}
				s.log.Error("failed syncing L2 block via RPC", "err", err, "num", reqNum)
				// Reschedule at end of queue
				select {
				case s.requests <- reqNum:
				default:
					// drop syncing job if we are too busy with sync jobs already.
				}
			}
		}
	}
}

// fetchUnsafeBlockFromRpc attempts to fetch an unsafe execution payload from the backup unsafe sync RPC.
// WARNING: This function fails silently (aside from warning logs).
//
// Post Shanghai hardfork, the engine API's `PayloadBodiesByRange` method will be much more efficient, but for now,
// the `eth_getBlockByNumber` method is more widely available.
func (s *SyncClient) fetchUnsafeBlockFromRpc(ctx context.Context, blockNumber uint64) error {
	s.log.Info("Requesting unsafe payload from backup RPC", "block number", blockNumber)

	payload, err := s.PayloadByNumber(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch payload by number (%d): %w", blockNumber, err)
	}
	// Note: the underlying RPC client used for syncing verifies the execution payload blockhash, if set to untrusted.

	s.log.Info("Received unsafe payload from backup RPC", "payload", payload.ID())

	// Send the retrieved payload to the `unsafeL2Payloads` channel.
	if err = s.receivePayload(ctx, RpcSyncPeer, payload); err != nil {
		return fmt.Errorf("failed to send payload %s into the driver's unsafeL2Payloads channel: %w", payload.ID(), err)
	} else {
		s.log.Debug("Sent received payload into the driver's unsafeL2Payloads channel", "payload", payload.ID())
		return nil
	}
}
