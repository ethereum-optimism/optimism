package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// ExecutionMode defines the execution mode for rollup boost
type ExecutionMode string

const (
	// ExecutionModeEnabled means rollup boost will forward tx to builder
	ExecutionModeEnabled ExecutionMode = "enabled"
	// ExecutionModeDisabled means rollup boost will not forward tx to builder
	ExecutionModeDisabled ExecutionMode = "disabled"
)

// RollupBoostDebug defines the interface for controlling the rollup boost debug API.
//
//go:generate mockery --name RollupBoostDebug --output mocks/ --with-expecter=true
type RollupBoostDebug interface {
	// GetExecutionMode gets the current execution mode of rollup boost
	GetExecutionMode(ctx context.Context) (ExecutionMode, error)
	// SetExecutionMode sets the execution mode of rollup boost
	SetExecutionMode(ctx context.Context, mode ExecutionMode) error
}

// RollupBoostDebugClient implements RollupBoostDebug
type RollupBoostDebugClient struct {
	client *rpc.Client
	log    log.Logger
}

var _ RollupBoostDebug = (*RollupBoostDebugClient)(nil)

// NewRollupBoostDebugClient creates a new RollupBoostDebugClient
func NewRollupBoostDebugClient(ctx context.Context, url string, log log.Logger) (*RollupBoostDebugClient, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	client, err := rpc.DialOptions(ctx, url, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to dial rollup boost debug API: %w", err)
	}

	return &RollupBoostDebugClient{
		client: client,
		log:    log.New("component", "rollupboost-client"),
	}, nil
}

// GetExecutionMode implements RollupBoostDebug
func (c *RollupBoostDebugClient) GetExecutionMode(ctx context.Context) (ExecutionMode, error) {
	var result struct {
		ExecutionMode string `json:"execution_mode"`
	}
	err := c.client.CallContext(ctx, &result, "debug_executionMode")
	if err != nil {
		return "", fmt.Errorf("failed to get execution mode: %w", err)
	}
	return ExecutionMode(result.ExecutionMode), nil
}

// SetExecutionMode implements RollupBoostDebug
func (c *RollupBoostDebugClient) SetExecutionMode(ctx context.Context, mode ExecutionMode) error {
	var result struct {
		ExecutionMode string `json:"execution_mode"`
	}
	err := c.client.CallContext(ctx, &result, "debug_setExecutionMode", string(mode))
	if err != nil {
		return fmt.Errorf("failed to set execution mode: %w", err)
	}
	c.log.Info("Set rollup boost execution mode", "mode", mode)
	return nil
}
