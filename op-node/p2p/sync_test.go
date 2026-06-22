package p2p

import (
	"context"
	"encoding/binary"
	"math/big"
	"sync"
	"testing"
	"time"

	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockPayloadFn func(n uint64) (*eth.ExecutionPayloadEnvelope, error)

func (fn mockPayloadFn) PayloadByNumber(_ context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	return fn(number)
}

var _ L2Chain = mockPayloadFn(nil)

type syncTestData struct {
	sync.RWMutex
	payloads map[uint64]*eth.ExecutionPayloadEnvelope
}

func (s *syncTestData) getPayload(i uint64) (payload *eth.ExecutionPayloadEnvelope, ok bool) {
	s.RLock()
	defer s.RUnlock()
	payload, ok = s.payloads[i]
	return payload, ok
}

func setupSyncTestData(length uint64) (*rollup.Config, *syncTestData) {
	// minimal rollup config to build mock blocks & verify their time.
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.Hash{0xaa}},
			L2:     eth.BlockID{Hash: common.Hash{0xbb}},
			L2Time: 9000,
		},
		BlockTime: 2,
		L2ChainID: big.NewInt(1234),
	}

	ecotoneBlock := length / 2
	ecotoneTime := cfg.Genesis.L2Time + ecotoneBlock*cfg.BlockTime
	cfg.EcotoneTime = &ecotoneTime

	// create some simple fake test blocks
	payloads := make(map[uint64]*eth.ExecutionPayloadEnvelope)
	payloads[0] = &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			Timestamp: eth.Uint64Quantity(cfg.Genesis.L2Time),
		},
	}

	payloads[0].ExecutionPayload.BlockHash, _ = payloads[0].CheckBlockHash()
	for i := uint64(1); i <= length; i++ {
		timestamp := cfg.Genesis.L2Time + i*cfg.BlockTime
		payload := &eth.ExecutionPayloadEnvelope{
			ExecutionPayload: &eth.ExecutionPayload{
				ParentHash:  payloads[i-1].ExecutionPayload.BlockHash,
				BlockNumber: eth.Uint64Quantity(i),
				Timestamp:   eth.Uint64Quantity(timestamp),
			},
		}

		if cfg.IsEcotone(timestamp) {
			hash := common.BigToHash(big.NewInt(int64(i)))
			payload.ParentBeaconBlockRoot = &hash

			zero := eth.Uint64Quantity(0)
			payload.ExecutionPayload.ExcessBlobGas = &zero
			payload.ExecutionPayload.BlobGasUsed = &zero

			w := types.Withdrawals{}
			payload.ExecutionPayload.Withdrawals = &w
		}

		payload.ExecutionPayload.BlockHash, _ = payload.CheckBlockHash()
		payloads[i] = payload
	}

	return cfg, &syncTestData{payloads: payloads}
}

func TestMutexUnlocks(t *testing.T) {
	cfg, payloads := setupSyncTestData(10)
	mockL2 := mockPayloadFn(func(n uint64) (*eth.ExecutionPayloadEnvelope, error) {
		p, ok := payloads.getPayload(n)
		if !ok {
			return nil, ethereum.NotFound
		}
		return p, nil
	})
	srv := NewReqRespServer(cfg, mockL2, metrics.NoopMetrics)

	mnet, err := mocknet.FullMeshConnected(2)
	require.NoError(t, err)
	defer mnet.Close()
	hosts := mnet.Hosts()
	hostA, hostB := hosts[0], hosts[1]

	ctx := context.Background()
	log := testlog.Logger(t, log.LevelError)
	payloadByNumber := MakeStreamHandler(ctx, log, srv.HandleSyncRequest)
	hostA.SetStreamHandler(PayloadByNumberProtocolID(cfg.L2ChainID), payloadByNumber)

	t.Run("SuccessCase", func(t *testing.T) {
		stream, err := hostB.NewStream(ctx, hostA.ID(), PayloadByNumberProtocolID(cfg.L2ChainID))
		require.NoError(t, err)
		_ = binary.Write(stream, binary.LittleEndian, uint64(1))
		_ = stream.CloseWrite()
		var result [1]byte
		_, _ = stream.Read(result[:])
		require.Equal(t, byte(0), result[0])
		stream.Close()

		srv.peerStatsLock.Lock()
		_ = srv.peerRateLimits.Len() // needed to satisfy linter
		srv.peerStatsLock.Unlock()
	})

	t.Run("ErrorCase", func(t *testing.T) {
		// First request: establish peer in rate limiter
		stream, err := hostB.NewStream(ctx, hostA.ID(), PayloadByNumberProtocolID(cfg.L2ChainID))
		require.NoError(t, err)
		_ = binary.Write(stream, binary.LittleEndian, uint64(1))
		_ = stream.CloseWrite()
		var result [1]byte
		_, _ = stream.Read(result[:])
		stream.Close()

		// Make rate limiter fail on next request
		peerId := hostB.ID()
		srv.peerStatsLock.Lock()
		ps, _ := srv.peerRateLimits.Get(peerId)
		ps.Requests = rate.NewLimiter(0, 0)
		ps.Requests.Reserve()
		srv.peerStatsLock.Unlock()

		// Second request with short timeout - return error but still unlock
		shortCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		stream2SentCh := make(chan bool)
		go func() {
			stream2, _ := hostB.NewStream(shortCtx, hostA.ID(), PayloadByNumberProtocolID(cfg.L2ChainID))
			// sometimes a different error arises where stream2 is null. this causes a panic
			// and the test can't be re-run. The channel fails the test instead, which allows
			// the test to be re-run.
			if stream2 == nil {
				stream2SentCh <- false
			} else {
				_ = binary.Write(stream2, binary.LittleEndian, uint64(2))
				_ = stream2.CloseWrite()
				stream2.Close()
				stream2SentCh <- true
			}
		}()

		stream2Sent := <-stream2SentCh
		require.True(t, stream2Sent)

		// Wait for request to fail
		time.Sleep(100 * time.Millisecond)

		// Test if mutex is stuck - should not hang on lock(), even if error is returned
		done := make(chan struct{})
		go func() {
			srv.peerStatsLock.Lock()
			_ = srv.peerRateLimits.Len() // needed to satisfy linter
			srv.peerStatsLock.Unlock()
			close(done)
		}()

		select {
		case <-done:
			// Mutex unlocked
		case <-time.After(1 * time.Second):
			t.Fatal("Mutex deadlock detected - bug exists")
		}
	})
}
