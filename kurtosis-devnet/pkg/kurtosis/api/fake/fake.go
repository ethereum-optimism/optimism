package fake

import (
	"context"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
)

// KurtosisContext implements interfaces.KurtosisContextInterface for testing
type KurtosisContext struct {
	EnclaveCtx    *EnclaveContext
	GetErr        error
	CreateErr     error
	CleanErr      error
	DestroyErr    error
	Status        interfaces.EnclaveStatus
	StatusErr     error
	DestroyCalled bool
	CleanCalled   bool
}

func (f *KurtosisContext) CreateEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	if f.CreateErr != nil {
		return nil, f.CreateErr
	}
	return f.EnclaveCtx, nil
}

func (f *KurtosisContext) GetEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	return f.EnclaveCtx, nil
}

func (f *KurtosisContext) GetEnclaveStatus(ctx context.Context, name string) (interfaces.EnclaveStatus, error) {
	if f.StatusErr != nil {
		return "", f.StatusErr
	}
	return f.Status, nil
}

func (f *KurtosisContext) Clean(ctx context.Context, destroyAll bool) ([]interfaces.EnclaveNameAndUuid, error) {
	f.CleanCalled = true
	if f.CleanErr != nil {
		return nil, f.CleanErr
	}
	return []interfaces.EnclaveNameAndUuid{}, nil
}

func (f *KurtosisContext) DestroyEnclave(ctx context.Context, name string) error {
	f.DestroyCalled = true
	if f.DestroyErr != nil {
		return f.DestroyErr
	}
	return nil
}

// EnclaveContext implements interfaces.EnclaveContext for testing
type EnclaveContext struct {
	RunErr        error
	Responses     []interfaces.StarlarkResponse
	ArtifactNames []*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid
	ArtifactData  []byte
	UploadErr     error

	Services map[services.ServiceName]interfaces.ServiceContext
	Files    map[services.FileArtifactName]string
}

func (f *EnclaveContext) GetEnclaveUuid() enclaves.EnclaveUUID {
	return enclaves.EnclaveUUID("test-enclave-uuid")
}

func (f *EnclaveContext) GetServices() (map[services.ServiceName]services.ServiceUUID, error) {
	if f.Services == nil {
		return nil, nil
	}
	services := make(map[services.ServiceName]services.ServiceUUID)
	for name, svc := range f.Services {
		services[name] = svc.GetServiceUUID()
	}
	return services, nil
}

func (f *EnclaveContext) GetService(serviceIdentifier string) (interfaces.ServiceContext, error) {
	if f.Services == nil {
		return nil, nil
	}
	return f.Services[services.ServiceName(serviceIdentifier)], nil
}

func (f *EnclaveContext) GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error) {
	artifacts := make([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, 0, len(f.Files))
	for name, uuid := range f.Files {
		artifacts = append(artifacts, &kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid{
			FileName: string(name),
			FileUuid: string(uuid),
		})
	}
	return artifacts, nil
}

func (f *EnclaveContext) RunStarlarkPackage(ctx context.Context, pkg string, params *starlark_run_config.StarlarkRunConfig) (<-chan interfaces.StarlarkResponse, string, error) {
	if f.RunErr != nil {
		return nil, "", f.RunErr
	}

	// Create a channel and send all responses
	ch := make(chan interfaces.StarlarkResponse)
	go func() {
		defer close(ch)
		for _, resp := range f.Responses {
			ch <- resp
		}
	}()
	return ch, "", nil
}

func (f *EnclaveContext) RunStarlarkScript(ctx context.Context, script string, params *starlark_run_config.StarlarkRunConfig) error {
	return f.RunErr
}

func (f *EnclaveContext) DownloadFilesArtifact(ctx context.Context, name string) ([]byte, error) {
	return f.ArtifactData, nil
}

func (f *EnclaveContext) UploadFiles(pathToUpload string, artifactName string) (services.FilesArtifactUUID, services.FileArtifactName, error) {
	if f.UploadErr != nil {
		return "", "", f.UploadErr
	}
	return "test-uuid", services.FileArtifactName(artifactName), nil
}

type ServiceContext struct {
	ServiceUUID  services.ServiceUUID
	PublicIP     string
	PrivateIP    string
	PublicPorts  map[string]*services.PortSpec
	PrivatePorts map[string]*services.PortSpec
}

func (f *ServiceContext) GetServiceUUID() services.ServiceUUID {
	return f.ServiceUUID
}

func (f *ServiceContext) GetMaybePublicIPAddress() string {
	return f.PublicIP
}

func (f *ServiceContext) GetPublicPorts() map[string]*services.PortSpec {
	return f.PublicPorts
}

func (f *ServiceContext) GetPrivatePorts() map[string]*services.PortSpec {
	return f.PrivatePorts
}

// StarlarkResponse implements interfaces.StarlarkResponse for testing
type StarlarkResponse struct {
	Err          interfaces.StarlarkError
	ProgressMsg  []string
	Instruction  string
	IsSuccessful bool
	Warning      string
	Info         string
	Result       string
	HasResult    bool // tracks whether result was explicitly set
}

func (f *StarlarkResponse) GetError() interfaces.StarlarkError {
	return f.Err
}

func (f *StarlarkResponse) GetProgressInfo() interfaces.ProgressInfo {
	if f.ProgressMsg != nil {
		return &ProgressInfo{Info: f.ProgressMsg}
	}
	return nil
}

func (f *StarlarkResponse) GetInstruction() interfaces.Instruction {
	if f.Instruction != "" {
		return &Instruction{Desc: f.Instruction}
	}
	return nil
}

func (f *StarlarkResponse) GetRunFinishedEvent() interfaces.RunFinishedEvent {
	return &RunFinishedEvent{IsSuccessful: f.IsSuccessful}
}

func (f *StarlarkResponse) GetWarning() interfaces.Warning {
	if f.Warning != "" {
		return &Warning{Msg: f.Warning}
	}
	return nil
}

func (f *StarlarkResponse) GetInfo() interfaces.Info {
	if f.Info != "" {
		return &Info{Msg: f.Info}
	}
	return nil
}

func (f *StarlarkResponse) GetInstructionResult() interfaces.InstructionResult {
	if !f.HasResult {
		return nil
	}
	return &InstructionResult{Result: f.Result}
}

// ProgressInfo implements ProgressInfo for testing
type ProgressInfo struct {
	Info []string
}

func (f *ProgressInfo) GetCurrentStepInfo() []string {
	return f.Info
}

// Instruction implements Instruction for testing
type Instruction struct {
	Desc string
}

func (f *Instruction) GetDescription() string {
	return f.Desc
}

// StarlarkError implements StarlarkError for testing
type StarlarkError struct {
	InterpretationErr error
	ValidationErr     error
	ExecutionErr      error
}

func (f *StarlarkError) GetInterpretationError() error {
	return f.InterpretationErr
}

func (f *StarlarkError) GetValidationError() error {
	return f.ValidationErr
}

func (f *StarlarkError) GetExecutionError() error {
	return f.ExecutionErr
}

// RunFinishedEvent implements RunFinishedEvent for testing
type RunFinishedEvent struct {
	IsSuccessful bool
}

func (f *RunFinishedEvent) GetIsRunSuccessful() bool {
	return f.IsSuccessful
}

// Warning implements Warning for testing
type Warning struct {
	Msg string
}

func (f *Warning) GetMessage() string {
	return f.Msg
}

// Info implements Info for testing
type Info struct {
	Msg string
}

func (f *Info) GetMessage() string {
	return f.Msg
}

// InstructionResult implements InstructionResult for testing
type InstructionResult struct {
	Result string
}

func (f *InstructionResult) GetSerializedInstructionResult() string {
	return f.Result
}
