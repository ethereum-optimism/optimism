package engine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kurtosis-tech/kurtosis/api/golang/kurtosis_version"
	"gopkg.in/yaml.v3"
)

// EngineManager handles running the Kurtosis engine
type EngineManager struct {
	kurtosisBinary string
	version        string
}

// Option configures an EngineManager
type Option func(*EngineManager)

// WithKurtosisBinary sets the path to the kurtosis binary
func WithKurtosisBinary(binary string) Option {
	return func(e *EngineManager) {
		e.kurtosisBinary = binary
	}
}

// WithVersion sets the engine version
func WithVersion(version string) Option {
	return func(e *EngineManager) {
		e.version = version
	}
}

// NewEngineManager creates a new EngineManager with the given options
func NewEngineManager(opts ...Option) *EngineManager {
	e := &EngineManager{
		kurtosisBinary: "kurtosis",                       // Default to expecting kurtosis in PATH
		version:        kurtosis_version.KurtosisVersion, // Default to library version
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EnsureRunning starts the Kurtosis engine with the configured version
func (e *EngineManager) EnsureRunning() error {
	cmd := exec.Command(e.kurtosisBinary, "engine", "start", "--version", e.version)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start kurtosis engine: %w", err)
	}
	return nil
}

// GetEngineType gets the type of the running engine (docker, kubernetes, etc)
func (e *EngineManager) GetEngineType() (string, error) {
	// First try to get the cluster name
	cmd := exec.Command(e.kurtosisBinary, "cluster", "get")
	output, err := cmd.Output()
	if err != nil {
		// Means there's no cluster set, so we're using the default cluster
		// Which is the first entry in kurtosis cluster ls
		cmd = exec.Command(e.kurtosisBinary, "cluster", "ls")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get cluster info: %w", err)
		}
		clusterName := strings.TrimSpace(string(output))
		return clusterName, nil
	}
	clusterName := strings.TrimSpace(string(output))

	cmd = exec.Command(e.kurtosisBinary, "config", "path")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get config path: %w", err)
	}
	configPath := strings.TrimSpace(string(output))

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config struct {
		KurtosisClusters map[string]struct {
			Type string `yaml:"type"`
		} `yaml:"kurtosis-clusters"`
	}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}

	cluster, exists := config.KurtosisClusters[clusterName]
	if !exists {
		// Means we're using the cluster definitions from the default config
		return clusterName, nil
	}

	return cluster.Type, nil
}
