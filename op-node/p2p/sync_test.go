package p2p

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type mockPayloadFn func(n uint64) (*eth.ExecutionPayload, error)

func (fn mockPayloadFn) PayloadByNumber(_ context.Context, number uint64) (*eth.ExecutionPayload, error) {
	return fn(number)
}

var _ L2Chain = mockPayloadFn(nil)

func setupSyncTestData(length uint64) (*rollup.Config, map[uint64]*eth.ExecutionPayload, func(i uint64) eth.L2BlockRef) {
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

	// create some simple fake test blocks
	payloads := make(map[uint64]*eth.ExecutionPayload)
	payloads[0] = &eth.ExecutionPayload{
		Timestamp: eth.Uint64Quantity(cfg.Genesis.L2Time),
	}
	payloads[0].BlockHash, _ = payloads[0].CheckBlockHash()
	for i := uint64(1); i <= length; i++ {
		payload := &eth.ExecutionPayload{
			ParentHash:  payloads[i-1].BlockHash,
			BlockNumber: eth.Uint64Quantity(i),
			Timestamp:   eth.Uint64Quantity(cfg.Genesis.L2Time + i*cfg.BlockTime),
		}
		payload.BlockHash, _ = payload.CheckBlockHash()
		payloads[i] = payload
	}

	l2Ref := func(i uint64) eth.L2BlockRef {
		return eth.L2BlockRef{
			Hash:       payloads[i].BlockHash,
			Number:     uint64(payloads[i].BlockNumber),
			ParentHash: payloads[i].ParentHash,
			Time:       uint64(payloads[i].Timestamp),
		}
	}
	return cfg, payloads, l2Ref
}

func TestSinglePeerSync(t *testing.T) {
	t.Parallel() // Takes a while, but can run in parallel

	log := testlog.Logger(t, log.LvlError)

	cfg, payloads, l2Ref := setupSyncTestData(25)

	// Serving payloads: just load them from the map, if they exist
	servePayload := mockPayloadFn(func(n uint64) (*eth.ExecutionPayload, error) {
		p, ok := payloads[n]
		if !ok {
			return nil, ethereum.NotFound
		}
		return p, nil
	})

	// collect received payloads in a buffered channel, so we can verify we get everything
	received := make(chan *eth.ExecutionPayload, 100)
	receivePayload := receivePayloadFn(func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
		received <- payload
		return nil
	})

	// Setup 2 minimal test hosts to attach the sync protocol to
	mnet, err := mocknet.FullMeshConnected(2)
	require.NoError(t, err, "failed to setup mocknet")
	defer mnet.Close()
	hosts := mnet.Hosts()
	hostA, hostB := hosts[0], hosts[1]
	require.Equal(t, hostA.Network().Connectedness(hostB.ID()), network.Connected)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup host A as the server
	srv := NewReqRespServer(cfg, servePayload, metrics.NoopMetrics)
	payloadByNumber := MakeStreamHandler(ctx, log.New("role", "server"), srv.HandleSyncRequest)
	hostA.SetStreamHandler(PayloadByNumberProtocolID(cfg.L2ChainID), payloadByNumber)

	// Setup host B as the client
	cl := NewSyncClient(log.New("role", "client"), cfg, hostB.NewStream, receivePayload, metrics.NoopMetrics)

	// Setup host B (client) to sync from its peer Host A (server)
	cl.AddPeer(hostA.ID())
	cl.Start()
	defer cl.Close()

	// request to start syncing between 10 and 20
	require.NoError(t, cl.RequestL2Range(ctx, l2Ref(10), l2Ref(20)))

	// and wait for the sync results to come in (in reverse order)
	receiveCtx, receiveCancel := context.WithTimeout(ctx, time.Second*5)
	defer receiveCancel()
	for i := uint64(19); i > 10; i-- {
		select {
		case p := <-received:
			require.Equal(t, uint64(p.BlockNumber), i, "expecting payloads in order")
			exp, ok := payloads[uint64(p.BlockNumber)]
			require.True(t, ok, "expecting known payload")
			require.Equal(t, exp.BlockHash, p.BlockHash, "expecting the correct payload")
		case <-receiveCtx.Done():
			t.Fatal("did not receive all expected payloads within expected time")
		}
	}
}

func TestMultiPeerSync(t *testing.T) {
	t.Parallel() // Takes a while, but can run in parallel

	log := testlog.Logger(t, log.LvlError)

	cfg, payloads, l2Ref := setupSyncTestData(100)

	setupPeer := func(ctx context.Context, h host.Host) (*SyncClient, chan *eth.ExecutionPayload) {
		// Serving payloads: just load them from the map, if they exist
		servePayload := mockPayloadFn(func(n uint64) (*eth.ExecutionPayload, error) {
			p, ok := payloads[n]
			if !ok {
				return nil, ethereum.NotFound
			}
			return p, nil
		})

		// collect received payloads in a buffered channel, so we can verify we get everything
		received := make(chan *eth.ExecutionPayload, 100)
		receivePayload := receivePayloadFn(func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
			received <- payload
			return nil
		})

		// Setup as server
		srv := NewReqRespServer(cfg, servePayload, metrics.NoopMetrics)
		payloadByNumber := MakeStreamHandler(ctx, log.New("serve", "payloads_by_number"), srv.HandleSyncRequest)
		h.SetStreamHandler(PayloadByNumberProtocolID(cfg.L2ChainID), payloadByNumber)

		cl := NewSyncClient(log.New("role", "client"), cfg, h.NewStream, receivePayload, metrics.NoopMetrics)
		return cl, received
	}

	// Setup 3 minimal test hosts to attach the sync protocol to
	mnet, err := mocknet.FullMeshConnected(3)
	require.NoError(t, err, "failed to setup mocknet")
	defer mnet.Close()
	hosts := mnet.Hosts()
	hostA, hostB, hostC := hosts[0], hosts[1], hosts[2]
	require.Equal(t, hostA.Network().Connectedness(hostB.ID()), network.Connected)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clA, recvA := setupPeer(ctx, hostA)
	clB, recvB := setupPeer(ctx, hostB)
	clC, _ := setupPeer(ctx, hostC)

	// Make them all sync from each other
	clA.AddPeer(hostB.ID())
	clA.AddPeer(hostC.ID())
	clA.Start()
	defer clA.Close()
	clB.AddPeer(hostA.ID())
	clB.AddPeer(hostC.ID())
	clB.Start()
	defer clB.Close()
	clC.AddPeer(hostA.ID())
	clC.AddPeer(hostB.ID())
	clC.Start()
	defer clC.Close()

	// request to start syncing between 10 and 100
	require.NoError(t, clA.RequestL2Range(ctx, l2Ref(10), l2Ref(90)))

	// With such large range to request we are going to hit the rate-limits of B and C,
	// but that means we'll balance the work between the peers.

	// wait for the results to come in, based on the expected rate limit, divided by 2 (because we have 2 servers), with a buffer of 2 seconds
	receiveCtx, receiveCancel := context.WithTimeout(ctx, time.Second*time.Duration(math.Ceil(float64((89-10)/peerServerBlocksRateLimit)))/2+time.Second*2)
	defer receiveCancel()
	for i := uint64(89); i > 10; i-- {
		select {
		case p := <-recvA:
			exp, ok := payloads[uint64(p.BlockNumber)]
			require.True(t, ok, "expecting known payload")
			require.Equal(t, exp.BlockHash, p.BlockHash, "expecting the correct payload")
		case <-receiveCtx.Done():
			t.Fatal("did not receive all expected payloads within expected time")
		}
	}

	// now see if B can sync a range, and fill the gap with a re-request
	bl25 := payloads[25] // temporarily remove it from the available payloads. This will create a gap
	delete(payloads, uint64(25))
	require.NoError(t, clB.RequestL2Range(ctx, l2Ref(20), l2Ref(30)))
	for i := uint64(29); i > 25; i-- {
		select {
		case p := <-recvB:
			exp, ok := payloads[uint64(p.BlockNumber)]
			require.True(t, ok, "expecting known payload")
			require.Equal(t, exp.BlockHash, p.BlockHash, "expecting the correct payload")
		case <-receiveCtx.Done():
			t.Fatal("did not receive all expected payloads within expected time")
		}
	}
	// the request for 25 should fail. See:
	// server: WARN  peer requested unknown block by number   num=25
	// client: WARN  failed p2p sync request    num=25 err="peer failed to serve request with code 1"
	require.Zero(t, len(recvB), "there is a gap, should not see other payloads yet")
	// Add back the block
	payloads[25] = bl25
	// And request a range again, 25 is there now, and 21-24 should follow quickly (some may already have been fetched and wait in quarantine)
	require.NoError(t, clB.RequestL2Range(ctx, l2Ref(20), l2Ref(26)))
	receiveCtx, receiveCancel = context.WithTimeout(ctx, time.Second*10)
	defer receiveCancel()
	for i := uint64(25); i > 20; i-- {
		select {
		case p := <-recvB:
			exp, ok := payloads[uint64(p.BlockNumber)]
			require.True(t, ok, "expecting known payload")
			require.Equal(t, exp.BlockHash, p.BlockHash, "expecting the correct payload")
		case <-receiveCtx.Done():
			t.Fatal("did not receive all expected payloads within expected time")
		}
	}
}
