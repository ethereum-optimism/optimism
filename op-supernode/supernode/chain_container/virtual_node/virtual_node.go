package virtual_node

import (
	"context"

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
	}
}

func (v *simpleVirtualNode) Start(ctx context.Context) error {
	if v.cfg == nil {
		v.log.Error("virtual node missing config")
		return nil
	}

	v.log.Info("virtual node starting with shared L1 and Beacon clients")
	opNode, err := e2eopnode.NewOpnode(v.log, v.cfg, func(err error) {
		if err != nil {
			v.log.Error("virtual op-node error", "error", err)
		}
	})
	if err != nil {
		v.log.Error("failed to start virtual op-node", "error", err)
		return err
	}
	v.inner = opNode
	return nil
}

func (v *simpleVirtualNode) Stop(ctx context.Context) error {
	v.log.Info("virtual node stopping")
	if v.inner != nil {
		return v.inner.Stop(ctx)
	}
	return nil
}
