package virtual_node

import (
	"context"
	"errors"
	"sync/atomic"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	opmetrics "github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type VirtualNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type simpleVirtualNode struct {
	log        gethlog.Logger
	vnID       string
	appVersion string

	inner        *rollupNode.OpNode       // Inner node
	cfg          *opnodecfg.Config        //op-node config for the virtual node
	initOverload *rollupNode.InitOverload // Shared resources which are overridden by the supernode

	stopCh      chan struct{} // Signal when the virtual node stop requested
	innerStopCh chan struct{} // Signal when inner node has stopped
	running     atomic.Bool   // Flag to track if the virtual node is running
}

func generateVirtualNodeID() string {
	return uuid.New().String()[:4]
}

func NewVirtualNode(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitOverload, appVersion string) *simpleVirtualNode {
	vnID := generateVirtualNodeID()
	l := log.New("chain_id", cfg.Rollup.L2ChainID.String(), "vn_id", vnID)
	return &simpleVirtualNode{
		vnID:         vnID,
		cfg:          cfg,
		log:          l,
		initOverload: initOverload,
		stopCh:       make(chan struct{}),
		appVersion:   appVersion,
	}
}

var ErrVirtualNodeConfigNil = errors.New("virtual node config is nil")

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	if v.running.Load() {
		v.log.Debug("virtual node already running")
	}
	if v.cfg == nil {
		return ErrVirtualNodeConfigNil
	}

	// when the node exits, it will send a signal to the stopCh
	// which allows the virtual node to exit
	var innerErr error
	v.cfg.Cancel = func(err error) {
		innerErr = err
		v.innerStopCh <- struct{}{}
	}

	m := opmetrics.NewMetrics("supernode")
	n, err := rollupNode.NewWithOverload(ctx, v.cfg, v.log, v.appVersion, m, v.initOverload)
	if err != nil {
		return err
	}
	v.inner = n

	go func() {
		v.running.Store(true)
		v.inner.Start(ctx)
	}()

	// wait for a stop request, inner node exit or context done
	select {
	case <-ctx.Done():
		v.log.Warn("virtual node context done", "err", ctx.Err())
		return v.inner.Stop(ctx)
	case <-v.stopCh:
		v.log.Warn("virtual node stopped")
		return v.inner.Stop(ctx)
	case <-v.innerStopCh:
		v.log.Warn("inner node stopped")
		return innerErr
	}
}

func (v *simpleVirtualNode) Stop(ctx context.Context) error {
	// if the virtual node is not running, return nil
	if !v.running.Load() {
		return nil
	}
	// send a signal to the stopCh to initiate the stop
	v.stopCh <- struct{}{}
	// set the running flag to false
	v.running.Store(false)
	return nil
}
