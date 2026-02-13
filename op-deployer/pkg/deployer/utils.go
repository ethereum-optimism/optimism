package deployer

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
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

func CreateCacheDir(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}
	return nil
}

func ChainIDFromRPC(ctx context.Context, rpcURL string) (*big.Int, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return chainID, nil
}

// IsVersionAtLeast parses a semver string (e.g., "6.0.0" or "7.1.2") and checks if it's >= the target version
func IsVersionAtLeast(versionStr string, targetMajor, targetMinor, targetPatch int) (bool, error) {
	// Remove any "v" prefix if present
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Split version string by "."
	parts := strings.Split(versionStr, ".")
	if len(parts) < 2 {
		return false, fmt.Errorf("invalid version format: %s", versionStr)
	}

	// Parse major version
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("invalid major version: %s", parts[0])
	}

	// Parse minor version
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	// Parse patch version if present (optional)
	patch := 0
	if len(parts) >= 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return false, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	// Compare versions
	if major > targetMajor {
		return true, nil
	}
	if major < targetMajor {
		return false, nil
	}

	// major == targetMajor
	if minor > targetMinor {
		return true, nil
	}
	if minor < targetMinor {
		return false, nil
	}

	// major == targetMajor && minor == targetMinor
	return patch >= targetPatch, nil
}
