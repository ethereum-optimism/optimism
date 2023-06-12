package challenger

import (
	"testing"

	"github.com/stretchr/testify/require"

	eth "github.com/ethereum/go-ethereum"
	abi "github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
)

// TestBuildOutputLogFilter_Succeeds tests that the Output
// Log Filter is built correctly.
func TestBuildOutputLogFilter_Succeeds(t *testing.T) {
	// Create a mock event id
	event := abi.Event{
		ID: [32]byte{0x01},
	}

	filterQuery := eth.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	// Mock the ABI
	l2ooABI := abi.ABI{
		Events: map[string]abi.Event{
			"OutputProposed": event,
		},
	}

	// Build the filter
	query, err := BuildOutputLogFilter(&l2ooABI)
	require.Equal(t, filterQuery, query)
	require.NoError(t, err)
}

// TestBuildOutputLogFilter_Fails tests that the Output
// Log Filter fails when the event definition is missing.
func TestBuildOutputLogFilter_Fails(t *testing.T) {
	// Mock the ABI
	l2ooABI := abi.ABI{
		Events: map[string]abi.Event{},
	}

	// Build the filter
	_, err := BuildOutputLogFilter(&l2ooABI)
	require.Error(t, err)
	require.Equal(t, ErrMissingEvent, err)
}
