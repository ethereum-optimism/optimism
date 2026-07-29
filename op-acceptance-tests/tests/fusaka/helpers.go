package fusaka

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// L1GethOption resolves a mise-installed geth binary for the L1 EL. This avoids mutating
// process-global env vars, which would otherwise block package-wide test parallelism.
func L1GethOption() presets.Option {
	cmd := exec.Command("mise", "which", "geth")
	buf := bytes.NewBuffer([]byte{})
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to find mise-installed geth: %v", err))
	}
	execPath := strings.TrimSpace(buf.String())
	return presets.WithL1Geth(execPath)
}
