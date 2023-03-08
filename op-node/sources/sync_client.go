package sources

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources/caching"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
)

var ErrNoUnsafeL2PayloadChannel = errors.New("unsafeL2Payloads channel must not be nil")

// RpcSyncPeer is a mock PeerID for the RPC sync client.
var RpcSyncPeer peer.ID = "ALT_RPC_SYNC"

type receivePayload = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error

type SyncClientInterface interface {
	Start() error
	Close() error
	fetchUnsafeBlockFromRpc(ctx context.Context, blockNumber uint64)
}

type SyncClient struct {
	*L2Client
	FetchUnsafeBlock chan uint64
	done             chan struct{}
	receivePayload   receivePayload
	wg               sync.WaitGroup
}

var _ SyncClientInterface = (*SyncClient)(nil)

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

	return &SyncClient{
		L2Client:         l2Client,
		FetchUnsafeBlock: make(chan uint64, 128),
		done:             make(chan struct{}),
		receivePayload:   receiver,
	}, nil
}

// Start starts up the state loop.
// The loop will have been started if err is not nil.
func (s *SyncClient) Start() error {
	s.wg.Add(1)
	go s.eventLoop()
	return nil
}

// Close sends a signal to the event loop to stop.
func (s *SyncClient) Close() error {
	s.done <- struct{}{}
	s.wg.Wait()
	return nil
}

// eventLoop is the main event loop for the sync client.
func (s *SyncClient) eventLoop() {
	defer s.wg.Done()
	s.log.Info("Starting sync client event loop")

	for {
		select {
		case <-s.done:
			return
		case blockNumber := <-s.FetchUnsafeBlock:
			s.fetchUnsafeBlockFromRpc(context.Background(), blockNumber)
		}
	}
}

// fetchUnsafeBlockFromRpc attempts to fetch an unsafe execution payload from the backup unsafe sync RPC.
// WARNING: This function fails silently (aside from warning logs).
//
// Post Shanghai hardfork, the engine API's `PayloadBodiesByRange` method will be much more efficient, but for now,
// the `eth_getBlockByNumber` method is more widely available.
func (s *SyncClient) fetchUnsafeBlockFromRpc(ctx context.Context, blockNumber uint64) {
	s.log.Info("Requesting unsafe payload from backup RPC", "block number", blockNumber)

	payload, err := s.PayloadByNumber(ctx, blockNumber)
	if err != nil {
		s.log.Warn("Failed to convert block to execution payload", "block number", blockNumber, "err", err)
		return
	}

	// Signature validation is not necessary here since the backup RPC is trusted.
	if _, ok := payload.CheckBlockHash(); !ok {
		s.log.Warn("Received invalid payload from backup RPC; invalid block hash", "payload", payload.ID())
		return
	}

	s.log.Info("Received unsafe payload from backup RPC", "payload", payload.ID())

	// Send the retrieved payload to the `unsafeL2Payloads` channel.
	if err = s.receivePayload(ctx, RpcSyncPeer, payload); err != nil {
		s.log.Warn("Failed to send payload into the driver's unsafeL2Payloads channel", "payload", payload.ID(), "err", err)
		return
	} else {
		s.log.Info("Sent received payload into the driver's unsafeL2Payloads channel", "payload", payload.ID())
	}
}
