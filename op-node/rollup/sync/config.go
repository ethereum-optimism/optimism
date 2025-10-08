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

type SafeSource int

// SafeSource determines where the safe head comes from:
//  1. In L1 safe source, the op-node derives the safe head from L1 data (default, trustless).
//  2. In L2 safe source, the op-node trusts another L2 node's safe head via RPC (fast follower mode).
const (
	SafeSourceL1 SafeSource = iota
	SafeSourceL2
)

const (
	SafeSourceL1String string = "l1"
	SafeSourceL2String string = "l2"
)

var SafeSources = []SafeSource{SafeSourceL1, SafeSourceL2}
var SafeSourceStrings = []string{SafeSourceL1String, SafeSourceL2String}

func StringToSafeSource(s string) (SafeSource, error) {
	switch strings.ToLower(s) {
	case SafeSourceL1String:
		return SafeSourceL1, nil
	case SafeSourceL2String:
		return SafeSourceL2, nil
	default:
		return 0, fmt.Errorf("unknown safe source: %s", s)
	}
}

func (s SafeSource) String() string {
	switch s {
	case SafeSourceL1:
		return SafeSourceL1String
	case SafeSourceL2:
		return SafeSourceL2String
	default:
		return "unknown"
	}
}

func (s *SafeSource) Set(value string) error {
	v, err := StringToSafeSource(value)
	if err != nil {
		return err
	}
	*s = v
	return nil
}

func (s *SafeSource) Clone() any {
	cpy := *s
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

	SupportsPostFinalizationELSync bool `json:"supports_post_finalization_elsync"`

	// SafeSource determines where the safe head comes from (L1 derivation or remote L2 RPC).
	SafeSource SafeSource `json:"safe_source"`
	// SafeSourceL2RPC is the RPC endpoint of the L2 node to use as safe source (required when SafeSource=SafeSourceL2).
	SafeSourceL2RPC string `json:"safe_source_l2_rpc,omitempty"`
}
