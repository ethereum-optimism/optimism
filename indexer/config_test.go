package indexer_test

import (
	"fmt"
	"testing"

	indexer "github.com/ethereum-optimism/optimism/indexer"
	"github.com/stretchr/testify/require"
)

var validateConfigTests = []struct {
	name   string
	cfg    indexer.Config
	expErr error
}{
	{
		name: "bad log level",
		cfg: indexer.Config{
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
			err := indexer.ValidateConfig(&test.cfg)
			require.Equal(t, err, test.expErr)
		})
	}
}
