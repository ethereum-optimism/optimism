package sync

import (
	"fmt"
	"strings"
)

type Mode int

// There are two kinds of sync mode that the op-node does:
//  1. In consensus-layer (CL) sync, the op-node fully drives the execution client and imports unsafe blocks &
//     fetches unsafe blocks that it has missed.
//  2. In execution-layer (EL) sync, the op-node tells the execution client to sync towards the tip of the chain.
//     It will consolidate the chain as usual. This allows execution clients to snap sync if they are capable of it.
const (
	CLSync Mode = iota
	ELSync Mode = iota
)

const (
	CLSyncString string = "consensus-layer"
	ELSyncString string = "execution-layer"
)

var Modes = []Mode{CLSync, ELSync}
var ModeStrings = []string{CLSyncString, ELSyncString}

func StringToMode(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case CLSyncString:
		return CLSync, nil
	case ELSyncString:
		return ELSync, nil
	default:
		return 0, fmt.Errorf("unknown sync mode: %s", s)
	}
}

func (m Mode) String() string {
	switch m {
	case CLSync:
		return CLSyncString
	case ELSync:
		return ELSyncString
	default:
		return "unknown"
	}
}

func (m *Mode) Set(value string) error {
	v, err := StringToMode(value)
	if err != nil {
		return err
	}
	*m = v
	return nil
}

func (m *Mode) Clone() any {
	cpy := *m
	return &cpy
}

type Config struct {
	// SyncMode is defined above.
	SyncMode Mode `json:"syncmode"`
	// SkipSyncStartCheck skip the sanity check of consistency of L1 origins of the unsafe L2 blocks when determining the sync-starting point.
	// This defers the L1-origin verification, and is recommended to use in when utilizing --syncmode=execution-layer on op-node and --syncmode=snap on op-geth
	// Warning: This will be removed when we implement proper checkpoints.
	// Note: We probably need to detect the condition that snap sync has not complete when we do a restart prior to running sync-start if we are doing
	// snap sync with a genesis finalization data.
	SkipSyncStartCheck bool `json:"skip_sync_start_check"`

	// SequencerFinalityLookback defines the amount of L1<>L2 relations to track for finalization purposes, one per L1 block.
	//
	// When L1 finalizes blocks, it finalizes finalityLookback blocks behind the L1 head.
	// Non-finality may take longer, but when it does finalize again, it is within this range of the L1 head.
	// Thus we only need to retain the L1<>L2 derivation relation data of this many L1 blocks.
	//
	// In the event of older finalization signals, misconfiguration, or insufficient L1<>L2 derivation relation data,
	// then we may miss the opportunity to finalize more L2 blocks.
	// This does not cause any divergence, it just causes lagging finalization status.
	//
	// The beacon chain on mainnet has 32 slots per epoch,
	// and new finalization events happen at most 4 epochs behind the head.
	// And then we add 1 to make pruning easier by leaving room for a new item without pruning the 32*4.
	SequencerFinalityLookback uint64 `json:"sequencer_finality_lookback"`
}
