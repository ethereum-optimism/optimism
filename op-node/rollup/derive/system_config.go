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
	SystemConfigUpdateBatcher   = common.Hash{31: 0}
	SystemConfigUpdateGasConfig = common.Hash{31: 1}
	SystemConfigUpdateGasLimit  = common.Hash{31: 2}
)

var (
	ConfigUpdateEventABI      = "ConfigUpdate(uint256,uint8,bytes)"
	ConfigUpdateEventABIHash  = crypto.Keccak256Hash([]byte(ConfigUpdateEventABI))
	ConfigUpdateEventVersion0 = common.Hash{}
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
		return fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != ConfigUpdateEventVersion0 {
		return fmt.Errorf("unrecognized L1 sysCfg update event version: %s", version)
	}
	// indexed 1
	updateType := ev.Topics[2]
	// unindexed data
	switch updateType {
	case SystemConfigUpdateBatcher:
		if len(ev.Data) != 32*3 {
			return fmt.Errorf("expected 32*3 bytes in batcher hash update, but got %d bytes", len(ev.Data))
		}
		if x := common.BytesToHash(ev.Data[:32]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected offset to point to length location, but got %s", x)
		}
		if x := common.BytesToHash(ev.Data[32:64]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected length of 1 bytes32, but got %s", x)
		}
		if !bytes.Equal(ev.Data[64:64+12], make([]byte, 12)) {
			return fmt.Errorf("expected version 0 batcher hash with zero padding, but got %x", ev.Data)
		}
		destSysCfg.BatcherAddr.SetBytes(ev.Data[64+12:])
		return nil
	case SystemConfigUpdateGasConfig: // left padded uint8
		if len(ev.Data) != 32*4 {
			return fmt.Errorf("expected 32*4 bytes in GPO params update data, but got %d", len(ev.Data))
		}
		if x := common.BytesToHash(ev.Data[:32]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected offset to point to length location, but got %s", x)
		}
		if x := common.BytesToHash(ev.Data[32:64]); x != (common.Hash{31: 64}) {
			return fmt.Errorf("expected length of 2 bytes32, but got %s", x)
		}
		copy(destSysCfg.Overhead[:], ev.Data[64:96])
		copy(destSysCfg.Scalar[:], ev.Data[96:128])
		return nil
	case SystemConfigUpdateGasLimit:
		if len(ev.Data) != 32*3 {
			return fmt.Errorf("expected 32*3 bytes in gas limit update, but got %d bytes", len(ev.Data))
		}
		if x := common.BytesToHash(ev.Data[:32]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected offset to point to length location, but got %s", x)
		}
		if x := common.BytesToHash(ev.Data[32:64]); x != (common.Hash{31: 32}) {
			return fmt.Errorf("expected length of 1 bytes32, but got %s", x)
		}
		if !bytes.Equal(ev.Data[64:64+24], make([]byte, 24)) {
			return fmt.Errorf("expected zero padding for gaslimit, but got %x", ev.Data)
		}
		destSysCfg.GasLimit = binary.BigEndian.Uint64(ev.Data[64+24:])
		return nil
	default:
		return fmt.Errorf("unrecognized L1 sysCfg update type: %s", updateType)
	}
}
