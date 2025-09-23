package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/util"
)

// InspectService handles the core inspection functionality
type InspectService struct {
	cfg *Config
	log log.Logger
}

func NewInspectService(cfg *Config, log log.Logger) *InspectService {
	return &InspectService{
		cfg: cfg,
		log: log,
	}
}

func (s *InspectService) Run(ctx context.Context) error {
	if s.cfg.FixTraefik {
		return s.fixTraefik(ctx)
	}

	return s.inspect(ctx)
}

func (s *InspectService) fixTraefik(ctx context.Context) error {
	s.log.Info("Fixing Traefik network configuration...")
	fmt.Println("🔧 Fixing Traefik network configuration...")

	if err := util.SetReverseProxyConfig(ctx); err != nil {
		return fmt.Errorf("error setting reverse proxy config: %w", err)
	}

	s.log.Info("Traefik network configuration fixed")
	fmt.Println("✅ Traefik network configuration fixed!")
	return nil
}

func (s *InspectService) inspect(ctx context.Context) error {
	inspector := NewInspector(s.cfg.EnclaveID)

	data, err := inspector.ExtractData(ctx)
	if err != nil {
		return fmt.Errorf("error inspecting enclave: %w", err)
	}

	conductorConfig, err := ExtractConductorConfig(ctx, s.cfg.EnclaveID)
	if err != nil {
		s.log.Warn("Error extracting conductor configuration", "error", err)
	}

	s.displayResults(data, conductorConfig)

	if err := s.writeFiles(data, conductorConfig); err != nil {
		return fmt.Errorf("error writing output files: %w", err)
	}

	return nil
}

func (s *InspectService) displayResults(data *InspectData, conductorConfig *ConductorConfig) {
	fmt.Println("File Artifacts:")
	for _, artifact := range data.FileArtifacts {
		fmt.Printf("  %s\n", artifact)
	}

	fmt.Println("\nServices:")
	for name, svc := range data.UserServices {
		fmt.Printf("  %s:\n", name)
		for portName, portInfo := range svc.Ports {
			host := portInfo.Host
			if host == "" {
				host = "localhost"
			}
			fmt.Printf("    %s: %s:%d\n", portName, host, portInfo.Port)
		}
	}

	if conductorConfig != nil {
		fmt.Println("\nConductor Configuration:")
		fmt.Println("========================")

		if err := toml.NewEncoder(os.Stdout).Encode(conductorConfig); err != nil {
			s.log.Error("Error marshaling conductor config to TOML", "error", err)
		}
	}
}

func (s *InspectService) writeFiles(data *InspectData, conductorConfig *ConductorConfig) error {
	if s.cfg.ConductorConfigPath != "" {
		if conductorConfig == nil {
			s.log.Info("No conductor services found, skipping conductor config generation")
		} else {
			if err := s.writeConductorConfig(s.cfg.ConductorConfigPath, conductorConfig); err != nil {
				return fmt.Errorf("error writing conductor config file: %w", err)
			}
			fmt.Printf("Conductor configuration saved to: %s\n", s.cfg.ConductorConfigPath)
		}
	}

	if s.cfg.EnvironmentPath != "" {
		if err := s.writeEnvironment(s.cfg.EnvironmentPath, data); err != nil {
			return fmt.Errorf("error writing environment file: %w", err)
		}
		fmt.Printf("Environment data saved to: %s\n", s.cfg.EnvironmentPath)
	}

	return nil
}

func (s *InspectService) writeConductorConfig(path string, config *ConductorConfig) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating conductor config file: %w", err)
	}
	defer out.Close()

	encoder := toml.NewEncoder(out)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("error encoding conductor config as TOML: %w", err)
	}

	return nil
}

func (s *InspectService) writeEnvironment(path string, data *InspectData) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating environment file: %w", err)
	}
	defer out.Close()

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("error encoding environment: %w", err)
	}

	return nil
}
