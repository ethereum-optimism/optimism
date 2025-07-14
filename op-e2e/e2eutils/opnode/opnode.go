package opnode

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/node/runcfg"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type Opnode struct {
	srv *service.Service
}

func (o *Opnode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return o.srv.Backend().InteropRPC()
}

func (o *Opnode) InteropRPCPort() (int, error) {
	return o.srv.Backend().InteropRPCPort()
}

func (o *Opnode) UserRPC() endpoint.RPC {
	return endpoint.HttpURL(o.srv.HttpRPC())
}

func (o *Opnode) UserRPCPort() (int, error) {
	return o.srv.Port()
}

func (o *Opnode) Stop(ctx context.Context) error {
	return o.srv.Stop(ctx)
}

func (o *Opnode) Stopped() bool {
	return o.srv.Stopped()
}

func (o *Opnode) RuntimeConfig() runcfg.ReadonlyRuntimeConfig {
	return o.srv.Backend().RuntimeConfig()
}

func (o *Opnode) P2P() p2p.Node {
	return o.srv.Backend().P2P()
}

var _ services.RollupNode = (*Opnode)(nil)

func NewOpnode(l log.Logger, c *config.Config, errFn func(error)) (*Opnode, error) {
	var cycle cliapp.Lifecycle
	c.Cancel = func(errCause error) {
		l.Warn("node requested early shutdown!", "err", errCause)
		go func() {
			postCtx, postCancel := context.WithCancel(context.Background())
			postCancel() // don't allow the stopping to continue for longer than needed
			if err := cycle.Stop(postCtx); err != nil {
				errFn(err)
			}
			l.Warn("closed op-node!")
		}()
	}
	ctx := context.Background()
	srv, err := service.FromConfig(ctx, c, l)
	if err != nil {
		return nil, fmt.Errorf("failed to setup node service: %w", err)
	}
	cycle = srv
	err = srv.Start(context.Background())
	if err != nil {
		return nil, err
	}
	return &Opnode{srv: srv}, nil
}
