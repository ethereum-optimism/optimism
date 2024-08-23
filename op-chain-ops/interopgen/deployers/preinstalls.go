package deployers

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type PreinstallsScript struct {
	SetPreinstalls func() error
}

func InsertPreinstalls(host *script.Host) error {
	l2GenesisScript, cleanupL2Genesis, err := script.WithScript[PreinstallsScript](host, "SetPreinstalls.s.sol", "SetPreinstalls")
	if err != nil {
		return fmt.Errorf("failed to load SetPreinstalls script: %w", err)
	}
	defer cleanupL2Genesis()

	if err := l2GenesisScript.SetPreinstalls(); err != nil {
		return fmt.Errorf("failed to set preinstalls: %w", err)
	}
	return nil
}
