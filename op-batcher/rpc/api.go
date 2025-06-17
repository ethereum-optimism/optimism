package rpc

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
)

type BatcherDriver interface {
	StartBatchSubmitting() error
	StopBatchSubmitting(ctx context.Context) error
	SetThrottleController(controllerType string, pidConfig *config.PIDConfig) error
	GetThrottleControllerInfo() (config.ThrottleControllerInfo, error)
	ResetThrottleController() error
}

type adminAPI struct {
	*rpc.CommonAdminAPI
	b BatcherDriver
}

var _ apis.BatcherAdminServer = (*adminAPI)(nil)

func NewAdminAPI(dr BatcherDriver, log log.Logger) *adminAPI {
	return &adminAPI{
		CommonAdminAPI: rpc.NewCommonAdminAPI(log),
		b:              dr,
	}
}

func GetAdminAPI(api *adminAPI) gethrpc.API {
	return gethrpc.API{
		Namespace: "admin",
		Service:   api,
	}
}

func (a *adminAPI) StartBatcher(_ context.Context) error {
	return a.b.StartBatchSubmitting()
}

func (a *adminAPI) StopBatcher(ctx context.Context) error {
	return a.b.StopBatchSubmitting(ctx)
}

// SetThrottleController changes the throttle controller type at runtime
func (a *adminAPI) SetThrottleController(_ context.Context, controllerType string, pidConfig *config.PIDConfig) error {
	// Validate controller type
	validTypes := []string{string(config.StepControllerType), string(config.LinearControllerType), string(config.QuadraticControllerType), string(config.PIDControllerType)}
	isValid := false
	for _, valid := range validTypes {
		if controllerType == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid controller type '%s', must be one of: %v", controllerType, validTypes)
	}

	// Validate PID config if needed
	if controllerType == "pid" && pidConfig == nil {
		return fmt.Errorf("PID configuration required when setting controller type to 'pid'")
	}

	if controllerType == "pid" {
		if pidConfig.Kp < 0 || pidConfig.Ki < 0 || pidConfig.Kd < 0 {
			return fmt.Errorf("PID gains must be non-negative")
		}
		if pidConfig.IntegralMax <= 0 {
			return fmt.Errorf("PID IntegralMax must be positive")
		}
		if pidConfig.OutputMax <= 0 || pidConfig.OutputMax > 1 {
			return fmt.Errorf("PID OutputMax must be between 0 and 1")
		}
	}

	return a.b.SetThrottleController(controllerType, pidConfig)
}

// GetThrottleController returns current throttle controller information
func (a *adminAPI) GetThrottleController(_ context.Context) (config.ThrottleControllerInfo, error) {
	return a.b.GetThrottleControllerInfo()
}

// ResetThrottleController resets the current throttle controller state
func (a *adminAPI) ResetThrottleController(_ context.Context) error {
	return a.b.ResetThrottleController()
}
