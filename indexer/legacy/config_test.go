package legacy_test

import (
	"fmt"
	"testing"

	legacy "github.com/ethereum-optimism/optimism/indexer/legacy"
	"github.com/stretchr/testify/require"
)

var validateConfigTests = []struct {
	name   string
	cfg    legacy.Config
	expErr error
}{
	{
		name: "bad log level",
		cfg: legacy.Config{
			LogLevel: "unknown",
		},
		expErr: fmt.Errorf("unknown level: unknown"),
	},
}

// TestValidateConfig asserts the behavior of ValidateConfig by testing expected
// error and success configurations.
func TestValidateConfig(t *testing.T) {
	for _, test := range validateConfigTests {
		t.Run(test.name, func(t *testing.T) {
			err := legacy.ValidateConfig(&test.cfg)
			require.Equal(t, err, test.expErr)
		})
	}
}
