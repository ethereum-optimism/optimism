package supervisor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigIsValid(t *testing.T) {
	cfg := DefaultCLIConfig()
	require.NoError(t, cfg.Check())
}
