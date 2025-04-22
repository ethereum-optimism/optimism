package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// NewDockerClient creates a new Docker client and checks if Docker is available
func NewDockerClient() (*client.Client, error) {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// Test the connection to verify Docker is available
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
