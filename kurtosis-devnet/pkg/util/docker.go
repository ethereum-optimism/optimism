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

func DestroyDockerResources(ctx context.Context, enclave ...string) error {
	apiClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	fmt.Printf("Destroying docker resources for enclave: %s\n", enclave)
	// Create filter for kurtosis resources
	kurtosisFilter := filters.NewArgs()
	enclaveName := ""
	if len(enclave) > 0 {
		enclaveName = enclave[0]
		kurtosisFilter.Add("label", fmt.Sprintf("kurtosis.devnet.enclave=%s", enclaveName))
	} else {
		kurtosisFilter.Add("label", "kurtosis.devnet.enclave")
	}

	// Find networks
	networks, err := apiClient.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		// Check if network matches our criteria
		if (enclaveName != "" && strings.HasPrefix(network.Name, fmt.Sprintf("kt-%s-devnet", enclaveName))) ||
			(enclaveName == "" && strings.Contains(network.Name, "kt-")) {
			// Get containers using this network
			containers, err := apiClient.ContainerList(ctx, container.ListOptions{
				All:     true,
				Filters: kurtosisFilter,
			})
			if err != nil {
				return fmt.Errorf("failed to list containers: %w", err)
			}

			// Stop and remove containers
			for _, cont := range containers {
				// Stop the container if it's running
				if cont.State == "running" {
					timeoutSecs := int(10)
					if err := apiClient.ContainerStop(ctx, cont.ID, container.StopOptions{
						Timeout: &timeoutSecs,
					}); err != nil {
						return fmt.Errorf("failed to stop container %s: %w", cont.ID, err)
					}
				}

				// Remove the container
				if err := apiClient.ContainerRemove(ctx, cont.ID, container.RemoveOptions{
					RemoveVolumes: true,
					Force:         true,
				}); err != nil {
					return fmt.Errorf("failed to remove container %s: %w", cont.ID, err)
				}
			}

			// Remove volumes associated with kurtosis
			volumes, err := apiClient.VolumeList(ctx, volume.ListOptions{
				Filters: kurtosisFilter,
			})
			if err != nil {
				return fmt.Errorf("failed to list volumes: %w", err)
			}
			for _, volume := range volumes.Volumes {
				if err := apiClient.VolumeRemove(ctx, volume.Name, true); err != nil {
					return fmt.Errorf("failed to remove volume %s: %w", volume.Name, err)
				}
			}

			// Finally remove the network
			if err := apiClient.NetworkRemove(ctx, network.ID); err != nil {
				return fmt.Errorf("failed to remove network: %w", err)
			}
		}
	}

	return nil
}
