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
)

type SyncClientInterface interface {
	Start(unsafeL2Payloads chan *eth.ExecutionPayload) error
	Close() error
	fetchUnsafeBlockFromRpc(ctx context.Context, blockNumber uint64)
}

type SyncClient struct {
	*L2Client
	FetchUnsafeBlock chan uint64
	done             chan struct{}
	unsafeL2Payloads chan *eth.ExecutionPayload
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

func NewSyncClient(client client.RPC, log log.Logger, metrics caching.Metrics, config *SyncClientConfig) (*SyncClient, error) {
	l2Client, err := NewL2Client(client, log, metrics, &config.L2ClientConfig)
	if err != nil {
		return nil, err
	}

	return &SyncClient{
		L2Client:         l2Client,
		FetchUnsafeBlock: make(chan uint64),
		done:             make(chan struct{}),
	}, nil
}

// Start starts up the state loop.
// The loop will have been started if err is not nil.
func (s *SyncClient) Start(unsafeL2Payloads chan *eth.ExecutionPayload) error {
	if unsafeL2Payloads == nil {
		return errors.New("unsafeL2Payloads channel must not be nil")
	}
	s.unsafeL2Payloads = unsafeL2Payloads

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
	s.log.Info("starting sync client event loop")

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
func (s *SyncClient) fetchUnsafeBlockFromRpc(ctx context.Context, blockNumber uint64) {
	s.log.Info("requesting unsafe payload from backup RPC", "block number", blockNumber)

	// TODO: Post Shanghai hardfork, the engine API's `PayloadBodiesByRange` method will be much more efficient, but for now,
	// the `eth_getBlockByNumber` method is more widely available.

	payload, err := s.PayloadByNumber(ctx, blockNumber)
	if err != nil {
		s.log.Warn("failed to convert block to execution payload", "block number", blockNumber, "err", err)
		return
	}

	// TODO: Validate the integrity of the payload. Is this required?
	// Signature validation is not necessary here since the backup RPC is trusted.
	if _, ok := payload.CheckBlockHash(); !ok {
		s.log.Warn("received invalid payload from backup RPC; invalid block hash", "payload", payload.ID())
		return
	}

	s.log.Info("received unsafe payload from backup RPC", "payload", payload.ID())

	// Send the retrieved payload to the `unsafeL2Payloads` channel.
	s.unsafeL2Payloads <- payload

	s.log.Info("sent received payload into the driver's unsafeL2Payloads channel", "payload", payload.ID())
}
