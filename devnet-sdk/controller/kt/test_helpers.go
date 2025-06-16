package kt

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/kurtosis_core_rpc_api_bindings"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/spf13/afero"
)

// testPortSpec implements interfaces.PortSpec
type testPortSpec struct {
	number uint16
}

func (m *testPortSpec) GetNumber() uint16 {
	return m.number
}

// testServiceContext implements interfaces.ServiceContext
type testServiceContext struct {
	publicPorts  map[string]interfaces.PortSpec
	privatePorts map[string]interfaces.PortSpec
}

func (m *testServiceContext) GetServiceUUID() services.ServiceUUID {
	return "mock-service-uuid"
}

func (m *testServiceContext) GetMaybePublicIPAddress() string {
	return "127.0.0.1"
}

func (m *testServiceContext) GetPublicPorts() map[string]interfaces.PortSpec {
	return m.publicPorts
}

func (m *testServiceContext) GetPrivatePorts() map[string]interfaces.PortSpec {
	return m.privatePorts
}

func (m *testServiceContext) GetLabels() map[string]string {
	return make(map[string]string)
}

// mockEnclaveFS implements fs.EnclaveContextIface for testing
type mockEnclaveFS struct {
	env *descriptors.DevnetEnvironment
}

func (m *mockEnclaveFS) GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error) {
	return nil, nil
}

func (m *mockEnclaveFS) DownloadFilesArtifact(ctx context.Context, name string) ([]byte, error) {
	return nil, nil
}

func (m *mockEnclaveFS) UploadFiles(pathToUpload string, artifactName string) (services.FilesArtifactUUID, services.FileArtifactName, error) {
	return "", "", nil
}

// newMockDevnetFS creates a new mock DevnetFS for testing
func newMockDevnetFS(env *descriptors.DevnetEnvironment) (*fs.DevnetFS, error) {
	mockEnclaveFS := &mockEnclaveFS{env: env}
	enclaveFS, err := fs.NewEnclaveFS(context.Background(), "test-enclave",
		fs.WithEnclaveCtx(mockEnclaveFS),
		fs.WithFs(afero.NewMemMapFs()),
	)
	if err != nil {
		return nil, err
	}
	return fs.NewDevnetFS(enclaveFS), nil
}

type mockEnclaveContext struct {
	services map[services.ServiceName]interfaces.ServiceContext
}

func (m *mockEnclaveContext) GetEnclaveUuid() enclaves.EnclaveUUID {
	return "mock-enclave-uuid"
}

func (m *mockEnclaveContext) GetService(serviceIdentifier string) (interfaces.ServiceContext, error) {
	if svc, ok := m.services[services.ServiceName(serviceIdentifier)]; ok {
		return svc, nil
	}
	return nil, nil
}

func (m *mockEnclaveContext) GetServices() (map[services.ServiceName]services.ServiceUUID, error) {
	result := make(map[services.ServiceName]services.ServiceUUID)
	for name, svc := range m.services {
		result[name] = svc.GetServiceUUID()
	}
	return result, nil
}

func (m *mockEnclaveContext) GetAllFilesArtifactNamesAndUuids(ctx context.Context) ([]*kurtosis_core_rpc_api_bindings.FilesArtifactNameAndUuid, error) {
	return nil, nil
}

func (m *mockEnclaveContext) RunStarlarkPackage(ctx context.Context, pkg string, serializedParams *starlark_run_config.StarlarkRunConfig) (<-chan interfaces.StarlarkResponse, string, error) {
	return nil, "", nil
}

func (m *mockEnclaveContext) RunStarlarkScript(ctx context.Context, script string, serializedParams *starlark_run_config.StarlarkRunConfig) error {
	return nil
}

// mockKurtosisContext implements interfaces.KurtosisContextInterface
type mockKurtosisContext struct {
	enclaves map[string]interfaces.EnclaveContext
}

func (m *mockKurtosisContext) CreateEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	if enclave, ok := m.enclaves[name]; ok {
		return enclave, nil
	}
	return nil, nil
}

func (m *mockKurtosisContext) GetEnclave(ctx context.Context, name string) (interfaces.EnclaveContext, error) {
	if enclave, ok := m.enclaves[name]; ok {
		return enclave, nil
	}
	return nil, nil
}

func (m *mockKurtosisContext) Clean(ctx context.Context, destroyAll bool) ([]interfaces.EnclaveNameAndUuid, error) {
	return []interfaces.EnclaveNameAndUuid{}, nil
}

func (m *mockKurtosisContext) GetEnclaveStatus(ctx context.Context, name string) (interfaces.EnclaveStatus, error) {
	return interfaces.EnclaveStatusRunning, nil
}

func (m *mockKurtosisContext) DestroyEnclave(ctx context.Context, name string) error {
	return nil
}
