package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var (
	SystemConfigUpdateBatcher           = common.Hash{31: 0}
	SystemConfigUpdateGasConfig         = common.Hash{31: 1}
	SystemConfigUpdateGasLimit          = common.Hash{31: 2}
	SystemConfigUpdateUnsafeBlockSigner = common.Hash{31: 3}
)

var (
	ConfigUpdateEventABI      = "ConfigUpdate(uint256,uint8,bytes)"
	ConfigUpdateEventABIHash  = crypto.Keccak256Hash([]byte(ConfigUpdateEventABI))
	ConfigUpdateEventVersion0 = common.Hash{}
)

// A left-padded uint256 equal to 32.
var OneWordUint = common.Hash{31: 32}

// 24 zero bytes (the padding for a uint64 in a 32 byte word)
var Uint64Padding = make([]byte, 24)

// 12 zero bytes (the padding for an Ethereum address in a 32 byte word)
var AddressPadding = make([]byte, 12)

var logger = log.New("derive", "system_config")

// UpdateSystemConfigWithL1Receipts filters all L1 receipts to find config updates and applies the config updates to the given sysCfg
func UpdateSystemConfigWithL1Receipts(sysCfg *eth.SystemConfig, receipts []*types.Receipt, cfg *rollup.Config) error {
	var result error
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == cfg.L1SystemConfigAddress && len(log.Topics) > 0 && log.Topics[0] == ConfigUpdateEventABIHash {
				if err := ProcessSystemConfigUpdateLogEvent(sysCfg, log); err != nil {
					result = multierror.Append(result, fmt.Errorf("malformatted L1 system sysCfg log in receipt %d, log %d: %w", i, j, err))
				}
			}
		}
	}
	return result
}

// ProcessSystemConfigUpdateLogEvent decodes an EVM log entry emitted by the system config contract and applies it as a system config change.
//
// parse log data for:
//
//	event ConfigUpdate(
//	    uint256 indexed version,
//	    UpdateType indexed updateType,
//	    bytes data
//	);
func ProcessSystemConfigUpdateLogEvent(destSysCfg *eth.SystemConfig, ev *types.Log) error {
	if len(ev.Topics) != 3 {
		return fmt.Errorf("expected 3 event topics (event identity, indexed version, indexed updateType), got %d", len(ev.Topics))
	}
	if ev.Topics[0] != ConfigUpdateEventABIHash {
		return fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != ConfigUpdateEventVersion0 {
		return fmt.Errorf("unrecognized L1 sysCfg update event version: %s", version)
	}
	// indexed 1
	updateType := ev.Topics[2]

	// Create a reader of the unindexed data
	reader := bytes.NewReader(ev.Data)

	// Helper function to prevent code duplication.
	readWord := func() (b [32]byte) {
		if _, err := reader.Read(b[:]); err != nil {
			// The possible error returned by `Read` is ignored due to the length check of the unindexed data in
			// all cases of the below switch statement. While we don't panic here, this log should *never* be emitted.
			logger.Crit("failed to read word from unindexed log data")
		}
		return b
	}

	// Attempt to read unindexed data
	switch updateType {
	case SystemConfigUpdateBatcher:
		if len(ev.Data) != 32*3 {
			return fmt.Errorf("expected 32*3 bytes in batcher hash update, but got %d bytes", len(ev.Data))
		}

		// Read the pointer, it should always equal 32.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should also always equal 32.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected length to be 32 bytes, but got %s", word)
		}

		// Indexing `word` directly is always safe here, it is guaranteed to be 32 bytes in length.
		// Check that the batcher address is correctly zero-padded.
		word := readWord()
		if !bytes.Equal(word[:12], AddressPadding) {
			return fmt.Errorf("expected version 0 batcher hash with zero padding, but got %x", word)
		}
		destSysCfg.BatcherAddr.SetBytes(word[12:])
		return nil
	case SystemConfigUpdateGasConfig:
		if len(ev.Data) != 32*4 {
			return fmt.Errorf("expected 32*4 bytes in GPO params update data, but got %d", len(ev.Data))
		}

		// Read the pointer, it should always equal 32.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should always equal 64.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected length to be 64 bytes, but got %s", word)
		}

		// Set the system config's overhead and scalar values to the values read from the log
		destSysCfg.Overhead = readWord()
		destSysCfg.Scalar = readWord()
		return nil
	case SystemConfigUpdateGasLimit:
		if len(ev.Data) != 32*3 {
			return fmt.Errorf("expected 32*3 bytes in gas limit update, but got %d bytes", len(ev.Data))
		}

		// Read the pointer, it should always equal 32.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should also always equal 32.
		if word := readWord(); common.BytesToHash(word[:]) != OneWordUint {
			return fmt.Errorf("expected length to be 32 bytes, but got %s", word)
		}

		// Indexing `word` directly is always safe here, it is guaranteed to be 32 bytes in length.
		// Check that the gas limit is correctly zero-padded.
		word := readWord()
		if !bytes.Equal(word[:24], Uint64Padding) {
			return fmt.Errorf("expected zero padding for gaslimit, but got %x", word)
		}
		destSysCfg.GasLimit = binary.BigEndian.Uint64(word[24:])
		return nil
	case SystemConfigUpdateUnsafeBlockSigner:
		// Ignored in derivation. This configurable applies to runtime configuration outside of the derivation.
		return nil
	default:
		return fmt.Errorf("unrecognized L1 sysCfg update type: %s", updateType)
	}
}
