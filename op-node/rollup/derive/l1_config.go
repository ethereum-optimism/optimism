package derive

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-multierror"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var (
	ConfigUpdateEventABI      = "ConfigUpdate(uint256,uint8,bytes)"
	ConfigUpdateEventABIHash  = crypto.Keccak256Hash([]byte(ConfigUpdateEventABI))
	ConfigUpdateEventVersion0 = common.Hash{}
)

// UpdateL1Config filters all receipts to find config updates and applies the config updates to the given l1Cfg
func UpdateL1Config(l1Cfg *eth.L1ConfigData, receipts []*types.Receipt, cfg *rollup.Config) error {
	var result error
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == cfg.L1SystemConfigAddress && len(log.Topics) > 0 && log.Topics[0] == ConfigUpdateEventABIHash {
				if err := ProcessConfigUpdateLogEvent(l1Cfg, log); err != nil {
					result = multierror.Append(result, fmt.Errorf("malformatted L1 system l1Config log in receipt %d, log %d: %w", i, j, err))
				}
			}
		}
	}
	return result
}

// ProcessConfigUpdateLogEvent decodes an EVM log entry emitted by the system l1Config contract and applies it as a l1Config change.
//
// parse log data for:
//
//	event ConfigUpdate(
//	    uint256 indexed version,
//	    UpdateType indexed updateType,
//	    bytes data
//	);
func ProcessConfigUpdateLogEvent(destL1Config *eth.L1ConfigData, ev *types.Log) error {
	if len(ev.Topics) != 3 {
		return fmt.Errorf("expected 3 event topics (event identity, indexed version, indexed updateType), got %d", len(ev.Topics))
	}
	if ev.Topics[0] != ConfigUpdateEventABIHash {
		return fmt.Errorf("invalid deposit event selector: %s, expected %s", ev.Topics[0], DepositEventABIHash)
	}

	// indexed 0
	version := ev.Topics[1]
	if version != ConfigUpdateEventVersion0 {
		return fmt.Errorf("unrecognized L1 l1Config update event version: %s", version)
	}
	// indexed 1
	updateType := ev.Topics[2]
	// unindexed data
	switch updateType {
	case common.Hash{}:
		destL1Config.BatcherAddr.SetBytes(ev.Data)
		return nil
	case common.Hash{31: 0x01}: // left padded uint8
		if len(ev.Data) != 32*2 {
			return fmt.Errorf("expected 32*2 bytes in GPO params update data, but got %d", len(ev.Data))
		}
		copy(destL1Config.Overhead[:], ev.Data[:32])
		copy(destL1Config.Scalar[:], ev.Data[32:])
		return nil
	default:
		return fmt.Errorf("unrecognized L1 l1Config update type: %s", updateType)
	}
}
