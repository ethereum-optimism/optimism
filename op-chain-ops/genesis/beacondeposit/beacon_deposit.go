package beacondeposit

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	gstate "github.com/ethereum/go-ethereum/core/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
)

//go:embed l1_empty_beacon_deposit_contract.json
var l1EmptyBeaconDepositContractJSON []byte

func InsertEmptyBeaconDepositContract(stateDB *state.MemoryStateDB, addr common.Address) error {
	var beaconDepositContractAccount gstate.DumpAccount
	if err := json.Unmarshal(l1EmptyBeaconDepositContractJSON, &beaconDepositContractAccount); err != nil {
		return fmt.Errorf("failed to read beacon deposit contract definition: %w", err)
	}
	stateDB.CreateAccount(addr)
	stateDB.SetCode(addr, beaconDepositContractAccount.Code)
	stateDB.SetNonce(addr, beaconDepositContractAccount.Nonce)
	for k, v := range beaconDepositContractAccount.Storage {
		stateDB.SetState(addr, k, common.HexToHash(v))
	}
	return nil
}
