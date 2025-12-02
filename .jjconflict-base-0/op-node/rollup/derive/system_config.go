package derive

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/solabi"
)

var (
	SystemConfigUpdateBatcher              = common.Hash{31: 0}
	SystemConfigUpdateFeeScalars           = common.Hash{31: 1}
	SystemConfigUpdateGasLimit             = common.Hash{31: 2}
	SystemConfigUpdateUnsafeBlockSigner    = common.Hash{31: 3}
	SystemConfigUpdateEIP1559Params        = common.Hash{31: 4}
	SystemConfigUpdateOperatorFeeParams    = common.Hash{31: 5}
	SystemConfigUpdateMinBaseFee           = common.Hash{31: 6}
	SystemConfigUpdateDAFootprintGasScalar = common.Hash{31: 7}
)

var (
	ConfigUpdateEventABI      = "ConfigUpdate(uint256,uint8,bytes)"
	ConfigUpdateEventABIHash  = crypto.Keccak256Hash([]byte(ConfigUpdateEventABI))
	ConfigUpdateEventVersion0 = common.Hash{}
)

var (
	ErrUnknownEventVersion  = errors.New("unknown SystemConfig event version")
	ErrUnknownEventType     = errors.New("unknown SystemConfig event type")
	ErrInvalidEIP1559Params = errors.New("invalid EIP-1559 parameters")
)

// UpdateSystemConfigWithL1Receipts filters all L1 receipts to find config updates and applies the config updates to the given sysCfg
// Updates are applied individually, and any malformed or invalid updates are ignored.
// Any errors encountered during the update process are returned as a multierror.
func UpdateSystemConfigWithL1Receipts(sysCfg *eth.SystemConfig, receipts []*types.Receipt, cfg *rollup.Config, l1Time uint64) error {
	var result error
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			// copy sysConfig to an update structure to preserve the original in case of error
			updated := *sysCfg
			if log.Address == cfg.L1SystemConfigAddress && len(log.Topics) > 0 && log.Topics[0] == ConfigUpdateEventABIHash {
				err := ProcessSystemConfigUpdateLogEvent(&updated, log, cfg, l1Time)
				if err == nil {
					// apply the updated structure
					*sysCfg = updated
				} else {
					// or append the error to the result
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
func ProcessSystemConfigUpdateLogEvent(destSysCfg *eth.SystemConfig, ev *types.Log, rollupCfg *rollup.Config, l1Time uint64) error {
	if len(ev.Topics) != 3 {
		return fmt.Errorf("expected 3 event topics (event identity, indexed version, indexed updateType), got %d", len(ev.Topics))
	}
	if ev.Topics[0] != ConfigUpdateEventABIHash {
		return fmt.Errorf("invalid SystemConfig update event: %s, expected %s", ev.Topics[0], ConfigUpdateEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != ConfigUpdateEventVersion0 {
		return fmt.Errorf("%w: %s", ErrUnknownEventVersion, version)
	}
	// indexed 1
	updateType := ev.Topics[2]

	// Attempt to read unindexed data
	switch updateType {
	case SystemConfigUpdateBatcher:
		addr, err := parseSystemConfigUpdateBatcher(ev.Data)
		if err != nil {
			return err
		}
		destSysCfg.BatcherAddr = addr
		return nil
	case SystemConfigUpdateFeeScalars:
		overhead, scalar, err := parseSystemConfigUpdateFeeScalars(ev.Data)
		if err != nil {
			return err
		}
		if rollupCfg.IsEcotone(l1Time) {
			if err := eth.CheckEcotoneL1SystemConfigScalar(scalar); err != nil {
				return nil // ignore invalid scalars, retain the old system-config scalar
			}
			// retain the scalar data in encoded form
			destSysCfg.Scalar = scalar
			// zero out the overhead, it will not affect the state-transition after Ecotone
			destSysCfg.Overhead = eth.Bytes32{}
		} else {
			destSysCfg.Overhead = overhead
			destSysCfg.Scalar = scalar
		}
		return nil
	case SystemConfigUpdateGasLimit:
		gasLimit, err := parseSystemConfigUpdateGasLimit(ev.Data)
		if err != nil {
			return err
		}
		destSysCfg.GasLimit = gasLimit
		return nil
	case SystemConfigUpdateEIP1559Params:
		params, err := parseSystemConfigUpdateEIP1559Params(ev.Data)
		if err != nil {
			return err
		}
		copy(destSysCfg.EIP1559Params[:], params[24:32])
		return nil
	case SystemConfigUpdateOperatorFeeParams:
		params, err := parseSystemConfigUpdateOperatorFeeParams(ev.Data)
		if err != nil {
			return err
		}
		destSysCfg.OperatorFeeParams = params
		return nil
	case SystemConfigUpdateUnsafeBlockSigner:
		// Ignored in derivation. This configurable applies to runtime configuration outside of the derivation.
		return nil
	case SystemConfigUpdateMinBaseFee:
		minBaseFee, err := parseSystemConfigUpdateMinBaseFee(ev.Data)
		if err != nil {
			return err
		}
		destSysCfg.MinBaseFee = minBaseFee
		return nil
	case SystemConfigUpdateDAFootprintGasScalar:
		daFootprintGasScalar, err := parseSystemConfigUpdateDAFootprintGasScalar(ev.Data)
		if err != nil {
			return err
		}
		destSysCfg.DAFootprintGasScalar = daFootprintGasScalar
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrUnknownEventType, updateType)
	}
}

var ErrParsingSystemConfig = errors.New("error parsing system config")

func parseSystemConfigUpdateBatcher(data []byte) (common.Address, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return common.Address{}, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return common.Address{}, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	address, err := solabi.ReadAddress(reader)
	if err != nil {
		return common.Address{}, fmt.Errorf("%w: could not read address", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return common.Address{}, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return address, nil
}

func parseSystemConfigUpdateFeeScalars(data []byte) (eth.Bytes32, eth.Bytes32, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 64 {
		return eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	overhead, err := solabi.ReadEthBytes32(reader)
	if err != nil {
		return eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("%w: could not read overhead", ErrParsingSystemConfig)
	}
	scalar, err := solabi.ReadEthBytes32(reader)
	if err != nil {
		return eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("%w: could not read scalar", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return eth.Bytes32{}, eth.Bytes32{}, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return overhead, scalar, nil
}

func parseSystemConfigUpdateGasLimit(data []byte) (uint64, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return 0, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return 0, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	gasLimit, err := solabi.ReadUint64(reader)
	if err != nil {
		return 0, fmt.Errorf("%w: could not read gas limit", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return 0, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return gasLimit, nil
}

func parseSystemConfigUpdateEIP1559Params(data []byte) (eth.Bytes32, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return eth.Bytes32{}, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return eth.Bytes32{}, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	params, err := solabi.ReadEthBytes32(reader)
	if err != nil {
		return eth.Bytes32{}, fmt.Errorf("%w: could not read eip-1559 params", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return eth.Bytes32{}, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	// Validate the EIP-1559 params (last 8 bytes of the 32-byte value)
	if err := eip1559.ValidateHolocene1559Params(params[24:32]); err != nil {
		return eth.Bytes32{}, fmt.Errorf("%w: %w", ErrInvalidEIP1559Params, err)
	}
	return params, nil
}

func parseSystemConfigUpdateOperatorFeeParams(data []byte) (eth.Bytes32, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return eth.Bytes32{}, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return eth.Bytes32{}, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	params, err := solabi.ReadEthBytes32(reader)
	if err != nil {
		return eth.Bytes32{}, fmt.Errorf("%w: could not read operator fee params", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return eth.Bytes32{}, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return params, nil
}

func parseSystemConfigUpdateMinBaseFee(data []byte) (uint64, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return 0, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return 0, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	minBaseFee, err := solabi.ReadUint64(reader)
	if err != nil {
		return 0, fmt.Errorf("%w: could not read minBaseFee", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return 0, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return minBaseFee, nil
}

func parseSystemConfigUpdateDAFootprintGasScalar(data []byte) (uint16, error) {
	reader := bytes.NewReader(data)
	if pointer, err := solabi.ReadUint64(reader); err != nil || pointer != 32 {
		return 0, fmt.Errorf("%w: invalid pointer field", ErrParsingSystemConfig)
	}
	if length, err := solabi.ReadUint64(reader); err != nil || length != 32 {
		return 0, fmt.Errorf("%w: invalid length field", ErrParsingSystemConfig)
	}
	daFootprintGasScalar, err := solabi.ReadUint16(reader)
	if err != nil {
		return 0, fmt.Errorf("%w: could not read DA footprint gas scalar", ErrParsingSystemConfig)
	}
	if !solabi.EmptyReader(reader) {
		return 0, fmt.Errorf("%w: too many bytes", ErrParsingSystemConfig)
	}
	return daFootprintGasScalar, nil
}
