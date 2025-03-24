package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type ExecutionMode bool

type RollupBoostControl interface {
	// GetExecutionMode gets the current execution mode of rollup boost
	GetExecutionMode(ctx context.Context) (bool, error)
	// SetExecutionMode sets the execution mode of rollup boost
	SetExecutionMode(ctx context.Context, enabled bool) error
}

// RollupBoostControlClient implements RollupBoostControl
type RollupBoostControlClient struct {
	client *rpc.Client
	log    log.Logger
}

var _ RollupBoostControl = (*RollupBoostControlClient)(nil)

// NewRollupBoostControlClient creates a new RollupBoostControlClient
func NewRollupBoostControlClient(ctx context.Context, url string, log log.Logger) (*RollupBoostControlClient, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	client, err := rpc.DialOptions(ctx, url, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to dial rollup boost debug API: %w", err)
	}

	return &RollupBoostControlClient{
		client: client,
		log:    log.New("component", "rollupboost-client"),
	}, nil
}

// GetExecutionMode implements RollupBoostControl
func (c *RollupBoostControlClient) GetExecutionMode(ctx context.Context) (bool, error) {
	var result struct {
		ExecutionMode bool `json:"execution_mode"`
	}
	err := c.client.CallContext(ctx, &result, "debug_executionMode")
	if err != nil {
		return false, fmt.Errorf("failed to get execution mode: %w", err)
	}
	return result.ExecutionMode, nil
}

// SetExecutionMode implements RollupBoostControl
func (c *RollupBoostControlClient) SetExecutionMode(ctx context.Context, enabled bool) error {
	var result struct {
		ExecutionMode bool `json:"execution_mode"`
	}
	err := c.client.CallContext(ctx, &result, "debug_setExecutionMode", enabled)
	if err != nil {
		return fmt.Errorf("failed to set execution mode: %w", err)
	}
	c.log.Info("Set rollup boost execution mode", "enabled", enabled)
	return nil
}
