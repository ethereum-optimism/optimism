package challenger

import (
	"testing"

	"github.com/stretchr/testify/require"

	eth "github.com/ethereum/go-ethereum"
	abi "github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
)

// TestBuildDisputeGameLogFilter_Succeeds tests that the DisputeGame
// Log Filter is built correctly.
func TestBuildDisputeGameLogFilter_Succeeds(t *testing.T) {
	event := abi.Event{
		ID: [32]byte{0x01},
	}

	filterQuery := eth.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	dgfABI := abi.ABI{
		Events: map[string]abi.Event{
			"DisputeGameCreated": event,
		},
	}

	query, err := BuildDisputeGameLogFilter(&dgfABI)
	require.Equal(t, filterQuery, query)
	require.NoError(t, err)
}

// TestBuildDisputeGameLogFilter_Fails tests that the DisputeGame
// Log Filter fails when the event definition is missing.
func TestBuildDisputeGameLogFilter_Fails(t *testing.T) {
	dgfABI := abi.ABI{
		Events: map[string]abi.Event{},
	}

	_, err := BuildDisputeGameLogFilter(&dgfABI)
	require.ErrorIs(t, ErrMissingFactoryEvent, err)
}
