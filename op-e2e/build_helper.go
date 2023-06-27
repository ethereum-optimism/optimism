package op_e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BuildOpProgramClient builds the `op-program` client executable and returns the path to the resulting executable
func BuildOpProgramClient(t *testing.T) string {
	t.Log("Building op-program-client")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "make", "op-program-client")
	cmd.Dir = "../op-program"
	cmd.Stdout = os.Stdout // for debugging
	cmd.Stderr = os.Stderr // for debugging
	require.NoError(t, cmd.Run(), "Failed to build op-program-client")
	t.Log("Built op-program-client successfully")
	return "../op-program/bin/op-program-client"
}
