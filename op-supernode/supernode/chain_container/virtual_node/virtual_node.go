package virtual_node

import (
	"context"
	"time"

	e2eopnode "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type VirtualNode interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type simpleVirtualNode struct {
	log          gethlog.Logger
	cfg          *opnodecfg.Config
	inner        *e2eopnode.Opnode
	vnID         string
	initOverload *rollupNode.InitializationOverrides // Shared resources
	stopCh       chan struct{}                       // Signal when node should stop running
}

func generateVirtualNodeID() string {
	return uuid.New().String()[:4]
}

func NewVirtualNode(cfg *opnodecfg.Config, log gethlog.Logger, initOverload *rollupNode.InitializationOverrides) *simpleVirtualNode {
	vnID := generateVirtualNodeID()
	l := log.New("chain_id", cfg.Rollup.L2ChainID.String(), "vn_id", vnID)
	return &simpleVirtualNode{
		vnID:         vnID,
		cfg:          cfg,
		log:          l,
		initOverload: initOverload,
		stopCh:       make(chan struct{}),
	}
}

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	if v.cfg == nil {
		return nil
	}

	errorFn := func(err error) {
		if err != nil {
			select {
			case v.stopCh <- struct{}{}:
			default:
			}
		}
	}

	opNode, err := e2eopnode.NewOpnodeWithOverload(v.log, v.cfg, errorFn, v.initOverload)
	if err != nil {
		return err
	}
	v.inner = opNode

	go func() {
		<-ctx.Done()
		if v.inner != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			v.inner.Stop(stopCtx)
		}
		select {
		case v.stopCh <- struct{}{}:
		default:
		}
	}()

	<-v.stopCh
	return nil
}

func (v *simpleVirtualNode) Stop(ctx context.Context) error {
	select {
	case v.stopCh <- struct{}{}:
	default:
	}

	if v.inner != nil {
		return v.inner.Stop(ctx)
	}
	return nil
}
