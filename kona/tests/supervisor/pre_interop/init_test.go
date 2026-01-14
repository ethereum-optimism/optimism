package preinterop

// todo: add tests
import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	spresets "github.com/op-rs/kona/supervisor/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// sleep to ensure the backend is ready

	presets.DoMain(m,
		spresets.WithSimpleInteropMinimal(),
		presets.WithSuggestedInteropActivationOffset(30),
		presets.WithInteropNotAtGenesis())

}
