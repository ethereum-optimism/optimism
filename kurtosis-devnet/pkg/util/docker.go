package util

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	opClient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type TraefikHostTransport struct {
	base http.RoundTripper
	host string
}

func (t *TraefikHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())
	newReq.Host = t.host
	newReq.Header.Set("Host", t.host)
	return t.base.RoundTrip(newReq)
}

type ServiceWithoutLabels struct {
	Name        string
	ServiceUUID string
	EnclaveUUID string
	Ports       []ServicePort
}

type ServicePort struct {
	Name string
	Port int
}

type RPCEndpoint struct {
	Name        string
	Port        int
	UUID        string
	EnclaveUUID string
}

// NewDockerClient creates a new Docker client and checks if Docker is available
func NewDockerClient() (*client.Client, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	_, err = apiClient.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	return apiClient, nil
}

// createKurtosisFilter creates a filter for kurtosis resources
func createKurtosisFilter(enclave ...string) filters.Args {
	kurtosisFilter := filters.NewArgs()
	if len(enclave) > 0 {
		kurtosisFilter.Add("label", fmt.Sprintf("kurtosis.devnet.enclave=%s", enclave[0]))
	} else {
		kurtosisFilter.Add("label", "kurtosis.devnet.enclave")
	}
	return kurtosisFilter
}

// destroyContainers stops and removes containers matching the filter
func destroyContainers(ctx context.Context, apiClient *client.Client, filter filters.Args) error {
	containers, err := apiClient.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filter,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, cont := range containers {
		if cont.State == "running" {
			timeoutSecs := int(10)
			if err := apiClient.ContainerStop(ctx, cont.ID, container.StopOptions{
				Timeout: &timeoutSecs,
			}); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", cont.ID, err)
			}
		}

		if err := apiClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{
			RemoveVolumes: true,
			Force:         true,
		}); err != nil {
			return fmt.Errorf("failed to remove container %s: %w", cont.ID, err)
		}
	}
	return nil
}

// destroyVolumes removes volumes matching the filter
func destroyVolumes(ctx context.Context, apiClient *client.Client, filter filters.Args) error {
	volumes, err := apiClient.VolumeList(ctx, volume.ListOptions{
		Filters: filter,
	})
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	for _, volume := range volumes.Volumes {
		if err := apiClient.VolumeRemove(ctx, volume.Name, true); err != nil {
			return fmt.Errorf("failed to remove volume %s: %w", volume.Name, err)
		}
	}
	return nil
}

// destroyNetworks removes networks matching the filter
func destroyNetworks(ctx context.Context, apiClient *client.Client, enclaveName string) error {
	networks, err := apiClient.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if (enclaveName != "" && strings.HasPrefix(network.Name, fmt.Sprintf("kt-%s-devnet", enclaveName))) ||
			(enclaveName == "" && strings.Contains(network.Name, "kt-")) {
			if err := apiClient.NetworkRemove(ctx, network.ID); err != nil {
				return fmt.Errorf("failed to remove network: %w", err)
			}
		}
	}
	return nil
}

// DestroyDockerResources removes all Docker resources associated with the given enclave
func DestroyDockerResources(ctx context.Context, enclave ...string) error {
	apiClient, err := NewDockerClient()
	if err != nil {
		return err
	}

	enclaveName := ""
	if len(enclave) > 0 {
		enclaveName = enclave[0]
	}
	fmt.Printf("Destroying docker resources for enclave: %s\n", enclaveName)

	filter := createKurtosisFilter(enclave...)

	if err := destroyContainers(ctx, apiClient, filter); err != nil {
		fmt.Printf("failed to destroy containers: %v", err)
	}

	if err := destroyVolumes(ctx, apiClient, filter); err != nil {
		fmt.Printf("failed to destroy volumes: %v", err)
	}

	if err := destroyNetworks(ctx, apiClient, enclaveName); err != nil {
		fmt.Printf("failed to destroy networks: %v", err)
	}

	return nil
}

func findRPCEndpoints(ctx context.Context, apiClient *client.Client) ([]RPCEndpoint, error) {
	userFilters := filters.NewArgs()
	userFilters.Add("label", "com.kurtosistech.container-type=user-service")

	containers, err := apiClient.ContainerList(ctx, container.ListOptions{
		All:     false,
		Filters: userFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var endpoints []RPCEndpoint
	seen := make(map[string]bool)

	for _, c := range containers {
		serviceName := strings.TrimPrefix(c.Names[0], "/")
		serviceUUID := c.Labels["com.kurtosistech.guid"]
		enclaveUUID := c.Labels["com.kurtosistech.enclave-id"]

		for _, port := range c.Ports {
			if port.PrivatePort == 8545 && port.PublicPort != 0 {
				key := fmt.Sprintf("%s-%s", serviceName, serviceUUID)
				if !seen[key] {
					seen[key] = true
					endpoints = append(endpoints, RPCEndpoint{
						Name:        serviceName,
						Port:        8545,
						UUID:        serviceUUID,
						EnclaveUUID: enclaveUUID,
					})
				}
			}
		}
	}

	return endpoints, nil
}

func testRPCEndpoint(endpoint RPCEndpoint) error {
	shortUUID := shortenedUUIDString(endpoint.UUID)
	shortEnclaveUUID := shortenedUUIDString(endpoint.EnclaveUUID)

	hostHeader := fmt.Sprintf("%d-%s-%s", endpoint.Port, shortUUID, shortEnclaveUUID)
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &TraefikHostTransport{
			base: http.DefaultTransport,
			host: hostHeader,
		},
	}

	rpcClient, err := rpc.DialOptions(context.Background(), "http://127.0.0.1:9730", rpc.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	defer rpcClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if strings.Contains(endpoint.Name, "supervisor") {
		return testSupervisor(ctx, rpcClient)
	}
	if strings.Contains(endpoint.Name, "test-sequencer") {
		// TODO: No public or unauthenticated health/status API exists for test-sequencer yet.
		//    Admin API is still in progress — skip readiness check until it's available.
		return nil
	}
	return testEthNode(ctx, rpcClient)
}

func testEthNode(ctx context.Context, rpcClient *rpc.Client) error {
	baseClient := opClient.NewBaseRPCClient(rpcClient)

	ethConfig := &sources.EthClientConfig{
		MaxRequestsPerBatch:   1,
		MaxConcurrentRequests: 1,
		ReceiptsCacheSize:     1,
		TransactionsCacheSize: 1,
		HeadersCacheSize:      1,
		PayloadsCacheSize:     1,
		BlockRefsCacheSize:    1,
		TrustRPC:              true,
		MustBePostMerge:       false,
		RPCProviderKind:       sources.RPCKindStandard,
		MethodResetDuration:   time.Minute,
	}

	ethClient, err := sources.NewEthClient(baseClient, log.Root(), nil, ethConfig)
	if err != nil {
		return fmt.Errorf("failed to create EthClient: %w", err)
	}

	blockRef, err := ethClient.BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	if blockRef.Number == 0 && blockRef.Hash.String() == "0x0000000000000000000000000000000000000000000000000000000000000000" {
		return fmt.Errorf("received invalid block reference")
	}

	blockInfo, err := ethClient.InfoByNumber(ctx, blockRef.Number)
	if err != nil {
		return fmt.Errorf("failed to get block info by number: %w", err)
	}

	if blockInfo.Hash() != blockRef.Hash {
		return fmt.Errorf("block hash mismatch: expected %s, got %s", blockRef.Hash, blockInfo.Hash())
	}

	return nil
}

func testSupervisor(ctx context.Context, rpcClient *rpc.Client) error {
	var syncStatus interface{}
	err := rpcClient.CallContext(ctx, &syncStatus, "supervisor_syncStatus")
	if err != nil {
		return fmt.Errorf("failed to call supervisor_syncStatus: %w", err)
	}

	if syncStatus == nil {
		return fmt.Errorf("supervisor_syncStatus returned nil")
	}

	return nil
}

// SetReverseProxyConfig recreates the Traefik container with correct configuration for service routing
func SetReverseProxyConfig(ctx context.Context) error {
	apiClient, err := NewDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	fmt.Printf("Fixing Traefik routing by recreating container\n")

	traefikFilters := filters.NewArgs()
	traefikFilters.Add("name", "kurtosis-reverse-proxy")

	traefikContainers, err := apiClient.ContainerList(ctx, container.ListOptions{
		All:     false,
		Filters: traefikFilters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var traefikContainer *types.Container
	for _, c := range traefikContainers {
		for _, name := range c.Names {
			if strings.Contains(name, "kurtosis-reverse-proxy") {
				traefikContainer = &c
				break
			}
		}
		if traefikContainer != nil {
			break
		}
	}

	if traefikContainer == nil {
		return fmt.Errorf("traefik container (kurtosis-reverse-proxy) not found, recreate it by restarting kurtosis (kurtosis engine restart)")
	}

	fmt.Printf("Found Traefik container: %s\n", traefikContainer.ID[:12])

	containerInfo, err := apiClient.ContainerInspect(ctx, traefikContainer.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}
	containerName := strings.TrimPrefix(containerInfo.Name, "/")
	containerImage := containerInfo.Config.Image
	var portBindings []string
	for containerPort, hostBindings := range containerInfo.HostConfig.PortBindings {
		for _, binding := range hostBindings {
			portBindings = append(portBindings, fmt.Sprintf("%s:%s", binding.HostPort, containerPort))
		}
	}
	var networks []string
	for networkName := range containerInfo.NetworkSettings.Networks {
		if networkName != "bridge" {
			networks = append(networks, networkName)
		}
	}
	var correctNetworkID string
	for networkName, network := range containerInfo.NetworkSettings.Networks {
		if networkName != "bridge" && strings.Contains(networkName, "kt-") {
			correctNetworkID = network.NetworkID
			break
		}
	}

	if correctNetworkID == "" {
		return fmt.Errorf("traefik container is not connected to any kurtosis networks")
	}

	tempDir, err := createTempConfigDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to create temp config directory: %w", err)
	}
	defer func() {
		if err := removeTempDir(tempDir); err != nil {
			fmt.Printf("Warning: Failed to clean up temp directory %s: %v\n", tempDir, err)
		}
	}()

	fmt.Printf("Stopping current Traefik container\n")
	timeout := int(10)
	if err := apiClient.ContainerStop(ctx, traefikContainer.ID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := apiClient.ContainerRemove(ctx, traefikContainer.ID, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	newContainer, err := recreateTraefikContainer(ctx, apiClient, containerName, containerImage, portBindings, tempDir, containerInfo.Config.Labels)
	if err != nil {
		return fmt.Errorf("failed to recreate container: %w", err)
	}

	fmt.Printf("Created new Traefik container: %s\n", newContainer.ID[:12])
	for _, networkName := range networks {
		if err := apiClient.NetworkConnect(ctx, networkName, newContainer.ID, nil); err != nil {
			return fmt.Errorf("failed to connect to network %s: %w", networkName, err)
		}
	}

	if err := waitForContainerRunning(ctx, apiClient, newContainer.ID, 30*time.Second); err != nil {
		return fmt.Errorf("container failed to start within timeout: %w", err)
	}

	if err := waitForTraefikReady(ctx, 30*time.Second); err != nil {
		fmt.Printf("Warning: Traefik API not ready within timeout: %v\n", err)
	}

	if err := TestRPCEndpoints(ctx, apiClient); err != nil {
		fmt.Printf("RPC access test failed: %v\n", err)
	}

	fmt.Printf("Traefik routing fix completed successfully\n")

	return nil
}

// waitForContainerRunning polls the container status until it's running or timeout
func waitForContainerRunning(ctx context.Context, apiClient *client.Client, containerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to start")
		case <-ticker.C:
			containerInfo, err := apiClient.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}
			if containerInfo.State.Running {
				return nil
			}
			if containerInfo.State.Status == "exited" {
				return fmt.Errorf("container exited unexpectedly: %s", containerInfo.State.Error)
			}
		}
	}
}

// waitForTraefikReady waits for Traefik API to be accessible
func waitForTraefikReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Traefik API to be ready")
		case <-ticker.C:
			resp, err := client.Get("http://127.0.0.1:9731/api/rawdata")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

func createTempConfigDir(ctx context.Context) (string, error) {
	tempDir, err := os.MkdirTemp("", "traefik-fix-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	apiClient, err := NewDockerClient()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer apiClient.Close()

	networks, err := apiClient.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	var traefikConfig strings.Builder
	traefikConfig.WriteString(`# Traefik configuration with corrected network ID
api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":9730"
  traefik:
    address: ":9731"

providers:
`)

	for _, net := range networks {
		// Only include Kurtosis networks (skip bridge, host, none)
		switch net.Name {
		case "bridge", "host", "none":
			continue
		}
		traefikConfig.WriteString(fmt.Sprintf(`  docker:
    endpoint: "unix:///var/run/docker.sock"
    network: "%s"
    exposedByDefault: true
`, net.ID))
		break
	}

	traefikConfig.WriteString(`  file:
    directory: /etc/traefik/dynamic
    watch: true

log:
  level: INFO
`)
	configPath := filepath.Join(tempDir, "traefik.yml")
	if err := os.WriteFile(configPath, []byte(traefikConfig.String()), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write traefik config: %w", err)
	}

	dynamicDir := filepath.Join(tempDir, "dynamic")
	if err := os.MkdirAll(dynamicDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to create dynamic directory: %w", err)
	}

	servicesWithoutLabels, err := discoverServicesWithoutTraefikLabels(ctx, apiClient)
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to discover services without Traefik labels: %v\n", err)
		servicesWithoutLabels = []ServiceWithoutLabels{}
	}
	var dynamicConfig strings.Builder
	dynamicConfig.WriteString("# Dynamic Traefik configuration for services without Traefik labels\n")
	dynamicConfig.WriteString("# Generated by SetReverseProxyConfig - do not edit manually\n")
	dynamicConfig.WriteString("http:\n")
	dynamicConfig.WriteString("  routers:\n")

	for _, service := range servicesWithoutLabels {
		addServiceRouters(&dynamicConfig, service)
	}
	dynamicConfig.WriteString("  services:\n")
	for _, service := range servicesWithoutLabels {
		addServiceServices(&dynamicConfig, service)
	}
	dynamicPath := filepath.Join(dynamicDir, "l1-routing.yml")
	if err := os.WriteFile(dynamicPath, []byte(dynamicConfig.String()), 0644); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to write dynamic config: %w", err)
	}
	fmt.Printf("Created dynamic routing rules for %d services\n", len(servicesWithoutLabels))
	return tempDir, nil
}

func removeTempDir(tempDir string) error {
	return os.RemoveAll(tempDir)
}

func recreateTraefikContainer(ctx context.Context, apiClient *client.Client, containerName, containerImage string, portBindings []string, tempDir string, labels map[string]string) (*container.CreateResponse, error) {
	exposedPorts := make(nat.PortSet)
	portBindingsMap := make(nat.PortMap)

	for _, binding := range portBindings {
		parts := strings.Split(binding, ":")
		if len(parts) == 2 {
			hostPort := parts[0]
			containerPortStr := parts[1]

			if strings.Contains(containerPortStr, "/") {
				containerPortStr = strings.Split(containerPortStr, "/")[0]
			}

			containerPort, err := nat.NewPort("tcp", containerPortStr)
			if err != nil {
				return nil, fmt.Errorf("invalid container port %s: %w", containerPortStr, err)
			}
			exposedPorts[containerPort] = struct{}{}
			var portBindings []nat.PortBinding
			portBindings = append(portBindings, nat.PortBinding{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			})
			portBindingsMap[containerPort] = portBindings
		}
	}

	mounts := []string{
		fmt.Sprintf("%s/traefik.yml:/etc/traefik/traefik.yml:ro", tempDir),
		fmt.Sprintf("%s/dynamic:/etc/traefik/dynamic:ro", tempDir),
		"/var/run/docker.sock:/var/run/docker.sock:ro",
	}

	config := &container.Config{
		Image:        containerImage,
		ExposedPorts: exposedPorts,
		Labels:       labels,
	}
	hostConfig := &container.HostConfig{
		Binds:        mounts,
		PortBindings: portBindingsMap,
	}
	resp, err := apiClient.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	if err := apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	return &resp, nil
}

func TestRPCEndpoints(ctx context.Context, apiClient *client.Client) error {
	endpoints, err := findRPCEndpoints(ctx, apiClient)
	if err != nil {
		return fmt.Errorf("failed to find RPC endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		fmt.Printf("No RPC endpoints found\n")
		return nil
	}

	fmt.Printf("Found %d RPC endpoint(s)\n", len(endpoints))

	var lastError error
	successCount := 0
	for _, endpoint := range endpoints {
		fmt.Printf("Testing %s", endpoint.Name)
		if err := testRPCEndpoint(endpoint); err != nil {
			fmt.Printf(" - Failed: %v\n", err)
			lastError = err
		} else {
			fmt.Printf(" - Success\n")
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("all RPC endpoints failed, last error: %w", lastError)
	}

	fmt.Printf("RPC access test passed (%d/%d endpoints working)\n", successCount, len(endpoints))
	return nil
}

// shortenedUUIDString returns the first 12 characters of a UUID
func shortenedUUIDString(fullUUID string) string {
	lengthToTrim := 12
	if lengthToTrim > len(fullUUID) {
		lengthToTrim = len(fullUUID)
	}
	return fullUUID[:lengthToTrim]
}

// discoverServicesWithoutTraefikLabels discovers services that need Traefik routing rules
func discoverServicesWithoutTraefikLabels(ctx context.Context, apiClient *client.Client) ([]ServiceWithoutLabels, error) {
	userFilters := filters.NewArgs()
	userFilters.Add("label", "com.kurtosistech.container-type=user-service")

	containers, err := apiClient.ContainerList(ctx, container.ListOptions{
		All:     false,
		Filters: userFilters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var servicesWithoutLabels []ServiceWithoutLabels

	for _, c := range containers {
		serviceName := strings.TrimPrefix(c.Names[0], "/")
		serviceUUID := c.Labels["com.kurtosistech.guid"]
		enclaveUUID := c.Labels["com.kurtosistech.enclave-id"]

		containerDetails, err := apiClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			fmt.Printf("failed to inspect container %s: %v\n", serviceName, err)
			continue
		}

		var portsWithoutLabels []ServicePort
		processedPorts := make(map[int]bool)

		for portSpec := range containerDetails.Config.ExposedPorts {
			port := portSpec.Port()
			portNum := 0
			if _, err := fmt.Sscanf(port, "%d", &portNum); err == nil {
				if processedPorts[portNum] {
					continue
				}
				processedPorts[portNum] = true
				hasTraefikLabelForPort := false
				for labelKey := range c.Labels {
					if strings.Contains(labelKey, "traefik.http.routers.") && strings.Contains(labelKey, ".rule") {
						if strings.Contains(labelKey, fmt.Sprintf("-%d", portNum)) {
							hasTraefikLabelForPort = true
							break
						}
					}
				}
				if !hasTraefikLabelForPort {
					portsWithoutLabels = append(portsWithoutLabels, ServicePort{
						Name: port,
						Port: portNum,
					})
				}
			}
		}

		if len(portsWithoutLabels) > 0 {
			servicesWithoutLabels = append(servicesWithoutLabels, ServiceWithoutLabels{
				Name:        serviceName,
				ServiceUUID: serviceUUID,
				EnclaveUUID: enclaveUUID,
				Ports:       portsWithoutLabels,
			})
		}
	}

	fmt.Printf("Discovered %d services with ports needing Traefik labels\n", len(servicesWithoutLabels))
	return servicesWithoutLabels, nil
}

// addServiceRouters adds Traefik router rules for a service and its ports
func addServiceRouters(dynamicConfig *strings.Builder, service ServiceWithoutLabels) {
	shortServiceUUID := shortenedUUIDString(service.ServiceUUID)
	shortEnclaveUUID := shortenedUUIDString(service.EnclaveUUID)
	for _, port := range service.Ports {
		routerName := fmt.Sprintf("%s-%s", service.Name, port.Name)
		serviceName := fmt.Sprintf("%s-%s", service.Name, port.Name)
		dynamicConfig.WriteString(fmt.Sprintf("    %s:\n", routerName))
		dynamicConfig.WriteString(fmt.Sprintf("      rule: \"HostRegexp(`{^name:%d-%s-%s-?.*$}`)\"\n", port.Port, shortServiceUUID, shortEnclaveUUID))
		dynamicConfig.WriteString(fmt.Sprintf("      service: \"%s\"\n", serviceName))
	}
}

// addServiceServices adds Traefik service definitions for a service and its ports
func addServiceServices(dynamicConfig *strings.Builder, service ServiceWithoutLabels) {
	for _, port := range service.Ports {
		serviceName := fmt.Sprintf("%s-%s", service.Name, port.Name)
		dynamicConfig.WriteString(fmt.Sprintf("    %s:\n", serviceName))
		dynamicConfig.WriteString("      loadBalancer:\n")
		dynamicConfig.WriteString("        servers:\n")
		dynamicConfig.WriteString(fmt.Sprintf("          - url: \"http://%s:%d\"\n", service.Name, port.Port))
	}
}
