package base

import (
	"os"
	"testing"
)

// TestCLAdvance_Supernode runs the same advance test using the supernode L2CL implementation.
func TestCLAdvance_Supernode(t *testing.T) {
	prev := os.Getenv("DEVSTACK_L2CL_KIND")
	_ = os.Setenv("DEVSTACK_L2CL_KIND", "supernode")
	defer func() { _ = os.Setenv("DEVSTACK_L2CL_KIND", prev) }()
	TestCLAdvance(t)
}
