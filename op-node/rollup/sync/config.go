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
	ELSync
	ForceELSync
)

const (
	CLSyncString      string = "consensus-layer"
	ELSyncString      string = "execution-layer"
	ForceELSyncString string = "force-execution-layer"
)

var Modes = []Mode{CLSync, ELSync, ForceELSync}
var ModeStrings = []string{CLSyncString, ELSyncString, ForceELSyncString}

func StringToMode(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case CLSyncString:
		return CLSync, nil
	case ELSyncString:
		return ELSync, nil
	case ForceELSyncString:
		return ForceELSync, nil
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
	case ForceELSync:
		return ForceELSyncString
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

func (m Mode) IsELSync() bool {
	return m == ELSync || m == ForceELSync
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
}
