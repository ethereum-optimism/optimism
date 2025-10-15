package deployer

import (
	"fmt"
	"log"
	"os"
	"path"
)

type DeploymentTarget string

const (
	DeploymentTargetLive     DeploymentTarget = "live"
	DeploymentTargetGenesis  DeploymentTarget = "genesis"
	DeploymentTargetCalldata DeploymentTarget = "calldata"
	DeploymentTargetNoop     DeploymentTarget = "noop"
)

func NewDeploymentTarget(s string) (DeploymentTarget, error) {
	switch s {
	case string(DeploymentTargetLive):
		return DeploymentTargetLive, nil
	case string(DeploymentTargetGenesis):
		return DeploymentTargetGenesis, nil
	case string(DeploymentTargetCalldata):
		return DeploymentTargetCalldata, nil
	case string(DeploymentTargetNoop):
		return DeploymentTargetNoop, nil
	default:
		return "", fmt.Errorf("invalid deployment target: %s", s)
	}
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

func DefaultCacheDir() string {
	var cacheDir string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		cacheDir = ".op-deployer/cache"
		log.Printf("error getting user home directory: %v, using fallback directory: %s\n", err, cacheDir)
	} else {
		cacheDir = path.Join(homeDir, ".op-deployer/cache")
	}

	return cacheDir
}

func CreateCacheDir(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}
	return nil
}
