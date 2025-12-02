package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var ErrNotInSync = errors.New("local node too far behind")

type Provider struct {
	ctx      context.Context
	logger   log.Logger
	cfg      *config.Config
	l1Client *ethclient.Client
	caller   *batching.MultiCaller

	l2EL             *ethclient.Client
	rollupClient     *sources.RollupClient
	syncValidator    *RollupSyncStatusValidator
	supervisorClient *sources.SupervisorClient
	toClose          []func()
}

func NewProvider(ctx context.Context, logger log.Logger, cfg *config.Config, l1Client *ethclient.Client) *Provider {
	return &Provider{
		ctx:      ctx,
		logger:   logger,
		cfg:      cfg,
		l1Client: l1Client,
		caller:   batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize),
	}
}

func (c *Provider) Close() {
	for _, closeFunc := range c.toClose {
		closeFunc()
	}
}

func (c *Provider) L1Client() *ethclient.Client {
	return c.l1Client
}

func (c *Provider) MultiCaller() *batching.MultiCaller {
	return c.caller
}

func (c *Provider) SingleChainClients() (*ethclient.Client, *sources.RollupClient, *RollupSyncStatusValidator, error) {
	headers, err := c.L2HeaderSource()
	if err != nil {
		return nil, nil, nil, err
	}
	rollup, syncValidator, err := c.RollupClients()
	if err != nil {
		return nil, nil, nil, err
	}
	return headers, rollup, syncValidator, nil
}

func (c *Provider) L2HeaderSource() (*ethclient.Client, error) {
	if c.l2EL != nil {
		return c.l2EL, nil
	}
	if len(c.cfg.L2Rpcs) != 1 {
		return nil, fmt.Errorf("incorrect number of L2 RPCs configured, expected 1 but got %d", len(c.cfg.L2Rpcs))
	}

	l2Client, err := ethclient.DialContext(c.ctx, c.cfg.L2Rpcs[0])
	if err != nil {
		return nil, fmt.Errorf("dial l2 client %v: %w", c.cfg.L2Rpcs[0], err)
	}
	c.l2EL = l2Client
	c.toClose = append(c.toClose, l2Client.Close)
	return l2Client, nil
}

func (c *Provider) RollupClients() (*sources.RollupClient, *RollupSyncStatusValidator, error) {
	if c.rollupClient != nil {
		return c.rollupClient, c.syncValidator, nil
	}
	rollupClient, err := dial.DialRollupClientWithTimeout(c.ctx, c.logger, c.cfg.RollupRpc)
	if err != nil {
		return nil, nil, fmt.Errorf("dial rollup client %v: %w", c.cfg.RollupRpc, err)
	}
	c.rollupClient = rollupClient
	c.syncValidator = NewRollupSyncStatusValidator(rollupClient)
	c.toClose = append(c.toClose, rollupClient.Close)
	return rollupClient, c.syncValidator, nil
}

func (c *Provider) SuperchainClients() (*sources.SupervisorClient, *SupervisorSyncValidator, error) {
	supervisorClient, err := dial.DialSupervisorClientWithTimeout(c.ctx, c.logger, c.cfg.SupervisorRPC)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial supervisor: %w", err)
	}
	c.supervisorClient = supervisorClient
	c.toClose = append(c.toClose, supervisorClient.Close)
	return supervisorClient, NewSupervisorSyncValidator(supervisorClient), nil
}
