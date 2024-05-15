package opconductor_client

import (
	"context"
	"net/http"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/config"
	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/metrics"
	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	opconductor "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
)

type InstrumentedOpConductorClient struct {
	c      *opconductor.APIClient
	node   string
	rpcUrl string
}

func New(ctx context.Context, config *config.Config, nodeName string, rpcUrl string) (*InstrumentedOpConductorClient, error) {
	pc, err := rpc.DialOptions(ctx, rpcUrl, rpc.WithHTTPClient(&http.Client{
		Timeout: config.RPCTimeout,
	}))
	if err != nil {
		metrics.RecordNetworkErrorDetails(nodeName, "conductor.New", err)
		log.Error("cant create op-conductor rpc client",
			"err", err)
		return nil, errors.Errorf("failed to create op-conductor rpc client with nodeName [%s], rpcUrl [%s]: %v", nodeName, rpcUrl, err)
	}
	p2pClient := opconductor.NewAPIClient(pc)

	return &InstrumentedOpConductorClient{
		c:      p2pClient,
		node:   nodeName,
		rpcUrl: rpcUrl,
	}, nil
}

func (i *InstrumentedOpConductorClient) Paused(ctx context.Context) (bool, error) {
	m := "conductor.Paused"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.Paused(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return false, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) Stopped(ctx context.Context) (bool, error) {
	m := "conductor.Stopped"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.Stopped(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return false, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) Active(ctx context.Context) (bool, error) {
	m := "conductor.Active"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.Active(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return false, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) SequencerHealthy(ctx context.Context) (bool, error) {
	m := "conductor.SequencerHealthy"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.SequencerHealthy(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return false, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) Leader(ctx context.Context) (bool, error) {
	m := "conductor.Leader"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.Leader(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return false, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) LeaderWithID(ctx context.Context) (*consensus.ServerInfo, error) {
	m := "conductor.LeaderWithID"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.LeaderWithID(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return nil, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}

func (i *InstrumentedOpConductorClient) ClusterMembership(ctx context.Context) ([]*consensus.ServerInfo, error) {
	m := "conductor.ClusterMembership"
	start := time.Now()
	log.Debug(m, "rpc_address", i.rpcUrl)
	val, err := i.c.ClusterMembership(ctx)
	if err != nil {
		metrics.RecordNetworkErrorDetails(i.node, m, err)
		return nil, err
	}
	metrics.RecordRPCLatency(i.node, m, time.Since(start))
	return val, err
}
