package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
)

// FailsafeClient defines the interface for controlling failsafe functionality
type FailsafeClient interface {
	SetFailsafeEnabled(ctx context.Context, enabled bool) error
	GetFailsafeEnabled(ctx context.Context) (bool, error)
}

// SupervisorClient provides functionality to call admin_setFailsafeEnabled on the supervisor
type SupervisorClient struct {
	client client.RPC
	log    log.Logger
}

var _ FailsafeClient = (*SupervisorClient)(nil)

// NewSupervisorClient creates a new supervisor client
func NewSupervisorClient(endpoint string, log log.Logger) (*SupervisorClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("supervisor endpoint not configured")
	}

	client, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create supervisor client: %w", err)
	}

	return &SupervisorClient{
		client: client,
		log:    log,
	}, nil
}

// SetFailsafeEnabled calls admin_setFailsafeEnabled on the supervisor
func (sc *SupervisorClient) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	err := sc.client.CallContext(ctx, nil, "admin_setFailsafeEnabled", enabled)
	if err != nil {
		return fmt.Errorf("failed to set failsafe mode for Supervisor: %w", err)
	}

	sc.log.Info("Successfully called admin_setFailsafeEnabled",
		"enabled", enabled)

	return nil
}

// Close closes the underlying RPC client
func (sc *SupervisorClient) Close() {
	sc.client.Close()
}

// GetFailsafeEnabled calls admin_getFailsafeEnabled on the supervisor
func (sc *SupervisorClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := sc.client.CallContext(ctx, &enabled, "admin_getFailsafeEnabled")
	return enabled, err
}
