package verify

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

var constructorArgSlots = map[string]int{
	"ProxyAdminAddress":              1,
	"OpcmAddress":                    28,
	"DelayedWETHImplAddress":         1,
	"OptimismPortalImplAddress":      2,
	"PreimageOracleSingletonAddress": 2,
	"MipsSingletonAddress":           1,
	"SuperchainConfigProxyAddress":   1,
	"PermissionedDisputeGameAddress": 12,
	"FaultDisputeGameAddress":        10,
}

func (v *Verifier) getConstructorArgs(ctx context.Context, address common.Address, contractName string) (string, error) {
	argSlots, ok := constructorArgSlots[contractName]
	if !ok {
		return "", nil
	}

	v.log.Info("Extracting constructor arguments", "address", address.Hex())
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
	v.log.Info("Successfully extracted constructor arguments", "address", address.Hex())

	return constructorArgs, nil
}
