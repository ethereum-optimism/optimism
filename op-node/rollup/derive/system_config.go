package derive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/go-multierror"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
)

var (
	SystemConfigUpdateBatcher           = common.Hash{31: 0}
	SystemConfigUpdateGasConfig         = common.Hash{31: 1}
	SystemConfigUpdateGasLimit          = common.Hash{31: 2}
	SystemConfigUpdateUnsafeBlockSigner = common.Hash{31: 3}
	SystemConfigUpdateGasConfigEcotone  = common.Hash{31: 4}
)

var (
	ConfigUpdateEventABI      = "ConfigUpdate(uint256,uint8,bytes)"
	ConfigUpdateEventABIHash  = crypto.Keccak256Hash([]byte(ConfigUpdateEventABI))
	ConfigUpdateEventVersion0 = common.Hash{}

	empty24 = make([]byte, 24)
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
	case SystemConfigUpdateGasConfigEcotone:
		// TODO(optimism#8801): pull this deserialization logic out into a public handler for solidity
		// diff/fuzz testing
		if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
			return NewCriticalError(errors.New("invalid pointer field"))
		}
		if length, err := solabi.ReadUint64(reader); err != nil || length != 8 {
			return NewCriticalError(errors.New("invalid length field"))
		}
		packed := make([]byte, 8)
		_, err := io.ReadFull(reader, packed)
		if err != nil {
			return NewCriticalError(errors.New("invalid packed scalars field"))
		}
		// confirm there is 32-8=24 bytes of 0-padding left
		zeros := make([]byte, 24)
		_, err = io.ReadFull(reader, zeros)
		if err != nil {
			return NewCriticalError(errors.New("didn't find expected padding"))
		}
		if !bytes.Equal(zeros, empty24) {
			return NewCriticalError(fmt.Errorf("expected padding to be all zeros, got %x", zeros))
		}
		if !solabi.EmptyReader(reader) {
			return NewCriticalError(errors.New("too many bytes"))
		}
		destSysCfg.BasefeeScalar = binary.BigEndian.Uint32(packed[0:4])
		destSysCfg.BlobBasefeeScalar = binary.BigEndian.Uint32(packed[4:8])
		return nil
	default:
		return fmt.Errorf("unrecognized L1 sysCfg update type: %s", updateType)
	}
}
