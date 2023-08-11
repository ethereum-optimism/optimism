package alphabet

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestAlphabetUpdater tests the [alphabetUpdater].
func TestAlphabetUpdater(t *testing.T) {
	logger := testlog.Logger(t, log.LvlInfo)
	updater := NewOracleUpdater(logger)
	require.Nil(t, updater.UpdateOracle(context.Background(), types.PreimageOracleData{}))
}
