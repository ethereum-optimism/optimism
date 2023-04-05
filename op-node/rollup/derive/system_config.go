package derive

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

var (
	// A left-padded uint256 equal to 32.
	oneWordUint = common.Hash{31: 32}
	// A left-padded uint256 equal to 64.
	twoWordUint = common.Hash{31: 64}
	// 24 zero bytes (the padding for a uint64 in a 32 byte word)
	uint64Padding = make([]byte, 24)
	// 12 zero bytes (the padding for an Ethereum address in a 32 byte word)
	addressPadding = make([]byte, 12)
)

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
		return fmt.Errorf("invalid SystemConfig update event: %s, expected %s", ev.Topics[0], ConfigUpdateEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != ConfigUpdateEventVersion0 {
		return fmt.Errorf("unrecognized SystemConfig update event version: %s", version)
	}
	// indexed 1
	updateType := ev.Topics[2]

	// Create a reader of the unindexed data
	reader := bytes.NewReader(ev.Data)

	// Counter for the number of bytes read from `reader` via `readWord`
	countReadBytes := 0

	// Helper function to read a word from the log data reader
	readWord := func() (b [32]byte) {
		if _, err := reader.Read(b[:]); err != nil {
			// If there is an error reading the next 32 bytes from the reader, return an empty
			// 32 byte array. We always check that the number of bytes read (`countReadBytes`)
			// is equal to the expected amount at the end of each switch case.
			return b
		}
		countReadBytes += 32
		return b
	}

	// Attempt to read unindexed data
	switch updateType {
	case SystemConfigUpdateBatcher:
		// Read the pointer, it should always equal 32.
		if word := readWord(); word != oneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should also always equal 32.
		if word := readWord(); word != oneWordUint {
			return fmt.Errorf("expected length to be 32 bytes, but got %s", word)
		}

		// Indexing `word` directly is always safe here, it is guaranteed to be 32 bytes in length.
		// Check that the batcher address is correctly zero-padded.
		word := readWord()
		if !bytes.Equal(word[:12], addressPadding) {
			return fmt.Errorf("expected version 0 batcher hash with zero padding, but got %x", word)
		}
		destSysCfg.BatcherAddr.SetBytes(word[12:])

		if countReadBytes != 32*3 {
			return NewCriticalError(fmt.Errorf("expected 32*3 bytes in batcher hash update, but got %d bytes", len(ev.Data)))
		}

		return nil
	case SystemConfigUpdateGasConfig:
		// Read the pointer, it should always equal 32.
		if word := readWord(); word != oneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should always equal 64.
		if word := readWord(); word != twoWordUint {
			return fmt.Errorf("expected length to be 64 bytes, but got %s", word)
		}

		// Set the system config's overhead and scalar values to the values read from the log
		destSysCfg.Overhead = readWord()
		destSysCfg.Scalar = readWord()

		if countReadBytes != 32*4 {
			return NewCriticalError(fmt.Errorf("expected 32*4 bytes in GPO params update data, but got %d", len(ev.Data)))
		}

		return nil
	case SystemConfigUpdateGasLimit:
		// Read the pointer, it should always equal 32.
		if word := readWord(); word != oneWordUint {
			return fmt.Errorf("expected offset to point to length location, but got %s", word)
		}

		// Read the length, it should also always equal 32.
		if word := readWord(); word != oneWordUint {
			return fmt.Errorf("expected length to be 32 bytes, but got %s", word)
		}

		// Indexing `word` directly is always safe here, it is guaranteed to be 32 bytes in length.
		// Check that the gas limit is correctly zero-padded.
		word := readWord()
		if !bytes.Equal(word[:24], uint64Padding) {
			return fmt.Errorf("expected zero padding for gaslimit, but got %x", word)
		}
		destSysCfg.GasLimit = binary.BigEndian.Uint64(word[24:])

		if countReadBytes != 32*3 {
			return NewCriticalError(fmt.Errorf("expected 32*3 bytes in gas limit update, but got %d bytes", len(ev.Data)))
		}

		return nil
	case SystemConfigUpdateUnsafeBlockSigner:
		// Ignored in derivation. This configurable applies to runtime configuration outside of the derivation.
		return nil
	default:
		return fmt.Errorf("unrecognized L1 sysCfg update type: %s", updateType)
	}
}
