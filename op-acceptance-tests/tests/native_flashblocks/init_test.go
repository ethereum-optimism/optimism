package native_flashblocks

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

func TestMain(m *testing.M) {
	if os.Getenv("OP_RETH_EXEC_PATH") == "" {
		fmt.Println("skipping native_flashblocks tests: OP_RETH_EXEC_PATH not set")
		os.Exit(0)
	}
	presets.DoMain(m, presets.WithSingleChainSystemWithNativeFlashblocks())
}
