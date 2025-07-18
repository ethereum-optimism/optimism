package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/gnosis"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var operationMap = map[string]uint8{
	"call":     0,
	"delegate": 1,
	"create":   2,
}

func SendGnosisTransactionCLI(cliCtx *cli.Context) error {
	lgr := oplog.NewLogger(oplog.AppOut(cliCtx), oplog.ReadCLIConfig(cliCtx))

	// Parse CLI args
	calldataHex := cliCtx.String(CalldataFlag.Name)
	safeAddress := common.HexToAddress(cliCtx.String(SafeAddressFlag.Name))
	rpcUrl := cliCtx.String(L1RpcUrlFlag.Name)
	toAddress := common.HexToAddress(cliCtx.String(ToAddressFlag.Name))
	operation, exists := operationMap[strings.ToLower(cliCtx.String(OperationFlag.Name))]
	if !exists {
		// Build list of valid operations
		validOps := make([]string, 0, len(operationMap))
		for op := range operationMap {
			validOps = append(validOps, op)
		}
		return fmt.Errorf("invalid operation: %s. Must be one of: %s", cliCtx.String(OperationFlag.Name), strings.Join(validOps, ", "))
	}

	privateKeysRaw := cliCtx.StringSlice(PrivateKeysFlag.Name)
	var privateKeys []string
	for _, key := range privateKeysRaw {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			privateKeys = append(privateKeys, trimmed)
		}
	}
	if len(privateKeys) == 0 {
		return fmt.Errorf("no valid private keys provided")
	}

	if safeAddress == (common.Address{}) {
		return fmt.Errorf("safe address cannot be 0x000...000")
	}

	if toAddress == (common.Address{}) {
		return fmt.Errorf("to address cannot be 0x000...000")
	}

	calldata := common.FromHex(calldataHex)
	if len(calldata) == 0 {
		return fmt.Errorf("invalid calldata: %s", calldataHex)
	}

	gnosisClient, err := gnosis.NewGnosisClient(lgr, rpcUrl, privateKeys, safeAddress,
		gnosis.WithCustomTxMgr(func(cfg *txmgr.CLIConfig) { cfg.NumConfirmations = 1 }),
	)
	if err != nil {
		return fmt.Errorf("failed to create gnosis client: %w", err)
	}
	defer gnosisClient.Close()

	lgr.Info("sending transaction", "calldataSize", len(calldata), "to", toAddress.Hex(), "operation", operation)
	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)
	receipt, err := gnosisClient.SendTransaction(ctx, toAddress, big.NewInt(0), calldata, operation)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	lgr.Info("transaction sent successfully", "txHash", receipt.TxHash.Hex())
	return nil
}
