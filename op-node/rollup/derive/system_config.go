package derive

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
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

	// Parse receipt log data
	version := ev.Topics[1]
	updateType := ev.Topics[2]
	reader := bytes.NewReader(ev.Data)

	// Switch on indexed 0 - the version
	switch version {
	case ConfigUpdateEventVersion0:
		return UpdateSystemConfigVersionZero(destSysCfg, updateType, reader)
	default:
		return fmt.Errorf("unrecognized SystemConfig update event version: %s", version)
	}
}

// UpdateSystemConfigVersionZero updates the given system config with the given updateType and data.
func UpdateSystemConfigVersionZero(destSysCfg *eth.SystemConfig, updateType common.Hash, reader *bytes.Reader) error {
	// Attempt to read unindexed data
	switch updateType {
	case SystemConfigUpdateBatcher:
		if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
			return NewCriticalError(errors.New("invalid pointer field"))
		}
		if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
			return NewCriticalError(errors.New("invalid length field"))
		}
		address, err := solabi.ReadAddress(reader)
		if err != nil {
			return NewCriticalError(errors.New("could not read address"))
		}
		if !solabi.EmptyReader(reader) {
			return NewCriticalError(errors.New("too many bytes"))
		}
		destSysCfg.BatcherAddr = address
		return nil
	case SystemConfigUpdateGasConfig:
		if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
			return NewCriticalError(errors.New("invalid pointer field"))
		}
		if length, err := solabi.ReadUint64(reader); err != nil || length != 64 {
			return NewCriticalError(errors.New("invalid length field"))
		}
		overhead, err := solabi.ReadEthBytes32(reader)
		if err != nil {
			return NewCriticalError(errors.New("could not read overhead"))
		}
		scalar, err := solabi.ReadEthBytes32(reader)
		if err != nil {
			return NewCriticalError(errors.New("could not read scalar"))
		}
		if !solabi.EmptyReader(reader) {
			return NewCriticalError(errors.New("too many bytes"))
		}
		destSysCfg.Overhead = overhead
		destSysCfg.Scalar = scalar
		return nil
	case SystemConfigUpdateGasLimit:
		if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
			return NewCriticalError(errors.New("invalid pointer field"))
		}
		if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
			return NewCriticalError(errors.New("invalid length field"))
		}
		gasLimit, err := solabi.ReadUint64(reader)
		if err != nil {
			return NewCriticalError(errors.New("could not read gas limit"))
		}
		if !solabi.EmptyReader(reader) {
			return NewCriticalError(errors.New("too many bytes"))
		}
		destSysCfg.GasLimit = gasLimit
		return nil
	case SystemConfigUpdateUnsafeBlockSigner:
		// Ignored in derivation. This configurable applies to runtime configuration outside of the derivation.
		return nil
	default:
		// Since we explicitly dispatch on version, this should never be reached.
		return nil
	}
}
