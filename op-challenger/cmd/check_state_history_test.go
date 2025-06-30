package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCheckStateHistoryCommand(t *testing.T) {
	// Test that the command is properly defined
	require.NotNil(t, CheckStateHistoryCommand)
	require.Equal(t, "check-state-history", CheckStateHistoryCommand.Name)
	require.Equal(t, "Check if sufficient state history is available for the challenger to operate", CheckStateHistoryCommand.Usage)
	require.NotNil(t, CheckStateHistoryCommand.Action)
	require.NotNil(t, CheckStateHistoryCommand.Flags)

	// Test that required flags are present
	var hasL1Flag, hasL2Flag, hasHistoryDepthFlag bool
	for _, flag := range CheckStateHistoryCommand.Flags {
		switch f := flag.(type) {
		case *cli.StringFlag:
			if f.Name == "l1-eth-rpc" {
				hasL1Flag = true
			}
			if f.Name == "l2-eth-rpc" {
				hasL2Flag = true
			}
		case *cli.Uint64Flag:
			if f.Name == "history-depth" {
				hasHistoryDepthFlag = true
				require.Equal(t, uint64(1000), f.Value) // Check default value
			}
		}
	}

	require.True(t, hasL1Flag, "should have l1-eth-rpc flag")
	require.True(t, hasL2Flag, "should have l2-eth-rpc flag")
	require.True(t, hasHistoryDepthFlag, "should have history-depth flag")
}

func TestCheckStateHistoryFlags(t *testing.T) {
	flags := checkStateHistoryFlags()
	require.Greater(t, len(flags), 3, "should have at least 4 flags")

	// Check that we have the basic required flags
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		switch f := flag.(type) {
		case *cli.StringFlag:
			flagNames[f.Name] = true
		case *cli.Uint64Flag:
			flagNames[f.Name] = true
		}
	}

	require.True(t, flagNames["l1-eth-rpc"], "should have l1-eth-rpc flag")
	require.True(t, flagNames["l2-eth-rpc"], "should have l2-eth-rpc flag")
	require.True(t, flagNames["history-depth"], "should have history-depth flag")
}
