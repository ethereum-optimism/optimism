package verify

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (v *Verifier) getConstructorArgs(ctx context.Context, address common.Address, artifact *contractArtifact) (string, error) {
	argSlots := 0
	for _, arg := range artifact.ConstructorArgs {
		argSlots += calculateTypeSlots(arg.Type)
	}
	if argSlots == 0 {
		return "", nil
	}

	v.log.Info("Extracting constructor args from initcode", "address", address.Hex(), "argSlots", argSlots)
	txHash, err := v.etherscan.getContractCreation(address)
	if err != nil {
		return "", fmt.Errorf("failed to get contract creation tx: %w", err)
	}
	v.log.Info("Contract creation tx hash", "txHash", txHash.Hex())

	tx, isPending, err := v.l1Client.TransactionByHash(ctx, txHash)
	if err != nil {
		return "", fmt.Errorf("failed to get transaction: %w", err)
	}

	if isPending {
		return "", fmt.Errorf("transaction is still pending")
	}

	// tx.Data contains bytecode + constructor args, so we strip the
	// constructor args off of the end
	txInput := hex.EncodeToString(tx.Data())
	constructorArgs := txInput[len(txInput)-(argSlots*64):]
	v.log.Info("Successfully extracted constructor args", "address", address.Hex())

	return constructorArgs, nil
}

// Helper function to calculate slots needed for a abi.Type, handling nested tuples
func calculateTypeSlots(t abi.Type) int {
	if t.String() == "string" {
		return 3 // 1 slot each for: offset, length, value (assuming value only takes 1 slot)
	} else if strings.HasPrefix(t.String(), "(") {
		// loop through nested tuple elements
		totalSlots := 0
		for _, elem := range t.TupleElems {
			totalSlots += calculateTypeSlots(*elem)
		}
		return totalSlots
	} else {
		return 1
	}
}
