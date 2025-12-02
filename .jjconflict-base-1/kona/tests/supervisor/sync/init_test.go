package sync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

const (
	// SLEEP_BACKEND_READY is the time to wait for the backend to be ready
	SLEEP_BACKEND_READY = 90 * time.Second
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// sleep to ensure the backend is ready
	time.Sleep(SLEEP_BACKEND_READY)

	presets.DoMain(m, presets.WithSimpleInterop())
}
