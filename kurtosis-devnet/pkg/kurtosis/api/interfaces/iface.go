package interfaces

import (
	"context"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
)

// Interfaces for Kurtosis SDK types to make testing easier
type StarlarkError interface {
	GetInterpretationError() error
	GetValidationError() error
	GetExecutionError() error
}

type ProgressInfo interface {
	GetCurrentStepInfo() []string
}

type Instruction interface {
	GetDescription() string
}

type RunFinishedEvent interface {
	GetIsRunSuccessful() bool
}

type Warning interface {
	GetMessage() string
}

type Info interface {
	GetMessage() string
}

type InstructionResult interface {
	GetSerializedInstructionResult() string
}

type StarlarkResponse interface {
	GetError() StarlarkError
	GetProgressInfo() ProgressInfo
	GetInstruction() Instruction
	GetRunFinishedEvent() RunFinishedEvent
	GetWarning() Warning
	GetInfo() Info
	GetInstructionResult() InstructionResult
}

type PortSpec interface {
	GetNumber() uint16
}

type ServiceContext interface {
	GetServiceUUID() services.ServiceUUID
	GetMaybePublicIPAddress() string
	GetPublicPorts() map[string]PortSpec
	GetPrivatePorts() map[string]PortSpec
}

type EnclaveContext interface {
	GetEnclaveUuid() enclaves.EnclaveUUID

	GetService(serviceIdentifier string) (ServiceContext, error)
	GetServices() (map[services.ServiceName]services.ServiceUUID, error)

	GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error)

	RunStarlarkPackage(context.Context, string, *starlark_run_config.StarlarkRunConfig) (<-chan StarlarkResponse, string, error)
	RunStarlarkScript(context.Context, string, *starlark_run_config.StarlarkRunConfig) error
}

type EnclaveNameAndUuid interface {
	GetName() string
	GetUuid() string
}

type EnclaveStatus string

const (
	EnclaveStatusEmpty   EnclaveStatus = "empty"
	EnclaveStatusRunning EnclaveStatus = "running"
	EnclaveStatusStopped EnclaveStatus = "stopped"
)

type KurtosisContextInterface interface {
	CreateEnclave(context.Context, string) (EnclaveContext, error)
	GetEnclave(context.Context, string) (EnclaveContext, error)
	GetEnclaveStatus(context.Context, string) (EnclaveStatus, error)
	DestroyEnclave(context.Context, string) error
	Clean(context.Context, bool) ([]EnclaveNameAndUuid, error)
}
