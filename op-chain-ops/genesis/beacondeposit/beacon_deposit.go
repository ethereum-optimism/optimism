package beacondeposit

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

//go:embed l1_empty_beacon_deposit_contract.json
var l1EmptyBeaconDepositContractJSON []byte

func InsertEmptyBeaconDepositContract(gen *core.Genesis, addr common.Address) error {
	var beaconDepositContractAccount types.Account
	if err := json.Unmarshal(l1EmptyBeaconDepositContractJSON, &beaconDepositContractAccount); err != nil {
		return fmt.Errorf("failed to read beacon deposit contract definition: %w", err)
	}
	gen.Alloc[addr] = beaconDepositContractAccount
	return nil
}
