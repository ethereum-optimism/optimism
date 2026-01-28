package preinterop

// todo: add tests
import (
	"testing"

	spresets "github.com/ethereum-optimism/optimism/kona/tests/supervisor/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// sleep to ensure the backend is ready

	presets.DoMain(m,
		spresets.WithSimpleInteropMinimal(),
		presets.WithSuggestedInteropActivationOffset(30),
		presets.WithInteropNotAtGenesis())

}
