package inspect

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

type PortMap map[string]descriptors.PortInfo

type ServiceMap map[string]PortMap

// InspectData represents a summary of the output of "kurtosis enclave inspect"
type InspectData struct {
	FileArtifacts []string
	UserServices  ServiceMap
}

type Inspector struct {
	enclaveID string
}

func NewInspector(enclaveID string) *Inspector {
	return &Inspector{enclaveID: enclaveID}
}

func (e *Inspector) ExtractData(ctx context.Context) (*InspectData, error) {
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		return nil, err
	}

	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, e.enclaveID)
	if err != nil {
		return nil, err
	}

	services, err := enclaveCtx.GetServices()
	if err != nil {
		return nil, err
	}

	artifacts, err := enclaveCtx.GetAllFilesArtifactNamesAndUuids(ctx)
	if err != nil {
		return nil, err
	}

	enclaveUUID := string(enclaveCtx.GetEnclaveUuid())

	data := &InspectData{
		UserServices:  make(ServiceMap),
		FileArtifacts: make([]string, len(artifacts)),
	}

	for i, artifact := range artifacts {
		data.FileArtifacts[i] = artifact.GetFileName()
	}

	for svc := range services {
		svc := string(svc)
		svcCtx, err := enclaveCtx.GetServiceContext(svc)
		if err != nil {
			return nil, err
		}
		svcUUID := string(svcCtx.GetServiceUUID())

		portMap := make(PortMap)

		for port, portSpec := range svcCtx.GetPublicPorts() {
			portMap[port] = descriptors.PortInfo{
				Host: svcCtx.GetMaybePublicIPAddress(),
				Port: int(portSpec.GetNumber()),
			}
		}

		for port, portSpec := range svcCtx.GetPrivatePorts() {
			// avoid non-mapped ports, we shouldn't have to use them.
			if p, ok := portMap[port]; ok {
				p.PrivatePort = int(portSpec.GetNumber())
				p.ReverseProxyHeader = http.Header{
					// This allows going through the kurtosis reverse proxy for each port
					"Host": []string{fmt.Sprintf("%d-%.12s-%.12s", p.PrivatePort, svcUUID, enclaveUUID)},
				}

				portMap[port] = p
			}
		}

		if len(portMap) != 0 {
			data.UserServices[svc] = portMap
		}

	}

	return data, nil
}
