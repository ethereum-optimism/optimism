package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// callFrame represents the response returned from geth's
// `debug_traceTransaction` callTracer
type callFrame struct {
	Type    string      `json:"type"`
	From    string      `json:"from"`
	To      string      `json:"to,omitempty"`
	Value   string      `json:"value,omitempty"`
	Gas     string      `json:"gas"`
	GasUsed string      `json:"gasUsed"`
	Input   string      `json:"input"`
	Output  string      `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	Calls   []callFrame `json:"calls,omitempty"`
}

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "withdrawals",
		Usage: "submits pending withdrawals",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "l1-rpc-url",
				Value: "http://127.0.0.1:8545",
				Usage: "RPC URL for an L1 Node",
			},
			&cli.StringFlag{
				Name:  "l2-rpc-url",
				Value: "http://127.0.0.1:9545",
				Usage: "RPC URL for an L2 Node",
			},
			&cli.StringFlag{
				Name:  "optimism-portal-address",
				Usage: "Address of the OptimismPortal on L1",
			},
			&cli.StringFlag{
				Name:  "l1-crossdomain-messenger-address",
				Usage: "Address of the L1CrossDomainMessenger",
			},
			&cli.StringFlag{
				Name:  "ovm-messages",
				Usage: "Path to ovm-messages.json",
			},
			&cli.StringFlag{
				Name:  "evm-messages",
				Usage: "Path to evm-messages.json",
			},
			&cli.Uint64Flag{
				Name:  "bedrock-transition-block-number",
				Usage: "The blocknumber of the bedrock transition block",
			},
			&cli.StringFlag{
				Name:  "private-key",
				Usage: "Key to sign transactions with",
			},
		},
		Action: func(ctx *cli.Context) error {
			l1RpcURL := ctx.String("l1-rpc-url")
			l1Client, err := ethclient.Dial(l1RpcURL)
			if err != nil {
				return err
			}
			l1ChainID, err := l1Client.ChainID(context.Background())
			if err != nil {
				return err
			}

			log.Info("Set up L1 RPC Client", "chain-id", l1ChainID)

			l2RpcURL := ctx.String("l2-rpc-url")
			l2Client, err := ethclient.Dial(l2RpcURL)
			if err != nil {
				return err
			}
			l2ChainID, err := l2Client.ChainID(context.Background())
			if err != nil {
				return err
			}
			log.Info("Set up L2 RPC Client", "chain-id", l2ChainID)

			l1RpcClient, err := rpc.DialContext(context.Background(), l1RpcURL)
			if err != nil {
				return err
			}

			l2RpcClient, err := rpc.DialContext(context.Background(), l2RpcURL)
			if err != nil {
				return err
			}
			gclient := gethclient.New(l2RpcClient)
			log.Info("Set up L2 geth Client")
			log.Debug(fmt.Sprintf("OVM path: %v", ctx.String("ovm-messages")))
			log.Debug(fmt.Sprintf("EVM path: %v", ctx.String("evm-messages")))
			ovmMessages, err := migration.NewSentMessage(ctx.String("ovm-messages"))
			if err != nil {
				return err
			}
			evmMessages, err := migration.NewSentMessage(ctx.String("evm-messages"))
			if err != nil {
				return err
			}
			log.Info("Created evm/ovm messages successfully")

			optimismPortalAddress := ctx.String("optimism-portal-address")
			if len(optimismPortalAddress) == 0 {
				return errors.New("OptimismPortal address not configured")
			}
			optimismPortalAddr := common.HexToAddress(optimismPortalAddress)

			migrationData := migration.MigrationData{
				OvmMessages: ovmMessages,
				EvmMessages: evmMessages,
			}

			wds, err := migrationData.ToWithdrawals()
			if err != nil {
				return err
			}
			if len(wds) == 0 {
				return errors.New("no withdrawals")
			}
			log.Info("Converted migration data to withdrawals successfully")

			l1xdmAddress := ctx.String("l1-crossdomain-messenger-address")
			if l1xdmAddress == "" {
				return errors.New("Must pass in --l1-crossdomain-messenger-address")
			}
			l1xdmAddr := common.HexToAddress(l1xdmAddress)

			// TODO: temp, should iterate over all instead of taking the first
			wd := wds[11]
			log.Debug(fmt.Sprintf("wd addr: %v", wd.Target))

			withdrawal, err := crossdomain.MigrateWithdrawal(wd, &l1xdmAddr)
			if err != nil {
				return err
			}

			hash, err := withdrawal.Hash()
			if err != nil {
				return err
			}

			slot, err := withdrawal.StorageSlot()
			if err != nil {
				return err
			}
			log.Info("Migrated first withdrawal successfully")

			portal, err := bindings.NewOptimismPortal(optimismPortalAddr, l1Client)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Deployed OptimismPortal successfully @ %v", optimismPortalAddr))

			// -- snip --
			period, err := portal.FINALIZATIONPERIODSECONDS(&bind.CallOpts{})
			if err != nil {
				return err
			}
			blockNo, err := l1Client.BlockNumber(context.Background())
			if err != nil {
				return err
			}
			log.Debug(fmt.Sprintf("Finalization Period (seconds): %v", period))
			log.Debug(fmt.Sprintf("Current L1 block #: %v", blockNo))
			// -- snip --

			l2OracleAddr, err := portal.L2ORACLE(&bind.CallOpts{})
			if err != nil {
				return err
			}
			oracle, err := bindings.NewL2OutputOracleCaller(l2OracleAddr, l1Client)
			if err != nil {
				return nil
			}
			log.Debug(fmt.Sprintf("L2 Oracle Address: %v", l2OracleAddr))

			transitionBlockNumber := new(big.Int).SetUint64(ctx.Uint64("bedrock-transition-block-number"))
			l2OutputIndex, err := oracle.GetL2OutputIndexAfter(&bind.CallOpts{}, transitionBlockNumber)
			if err != nil {
				return err
			}
			l2Output, err := oracle.GetL2Output(&bind.CallOpts{}, l2OutputIndex)
			if err != nil {
				return err
			}
			log.Debug(fmt.Sprintf("L2 Output index: %v", l2OutputIndex.String()))
			log.Debug(fmt.Sprintf("L2 Output root: %v", common.Bytes2Hex(l2Output.OutputRoot[:])))
			log.Debug(fmt.Sprintf("L2 Bedrock Genesis block number: %v", transitionBlockNumber))
			log.Debug(fmt.Sprintf("L2 block number of output: %v", l2Output.L2BlockNumber))

			header, err := l2Client.HeaderByNumber(context.Background(), l2Output.L2BlockNumber)
			if err != nil {
				return err
			}
			log.Debug(fmt.Sprintf("Fetched header for L2 block #%v", l2Output.L2BlockNumber))

			proof, err := gclient.GetProof(context.Background(), predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, transitionBlockNumber)
			if err != nil {
				return err
			}
			if len(proof.StorageProof) != 1 {
				return errors.New("invalid amount of storage proofs")
			}
			trieNodes := make([][]byte, len(proof.StorageProof[0].Proof))
			for i, s := range proof.StorageProof[0].Proof {
				trieNodes[i] = common.FromHex(s)
			}
			log.Info("Generated proof and trie nodes successfully")

			outputRootProof := bindings.TypesOutputRootProof{
				Version:                  [32]byte{},
				StateRoot:                header.Root,
				MessagePasserStorageRoot: proof.StorageHash,
				LatestBlockhash:          header.Hash(),
			}

			localOutputRootHash := crypto.Keccak256Hash(
				outputRootProof.Version[:],
				outputRootProof.StateRoot[:],
				outputRootProof.MessagePasserStorageRoot[:],
				outputRootProof.LatestBlockhash[:],
			)
			log.Debug(fmt.Sprintf("Do output root hashes match? : %v", localOutputRootHash.Hex()[2:] == common.Bytes2Hex(l2Output.OutputRoot[:])))

			if ctx.String("private-key") == "" {
				return errors.New("No private key to transact with")
			}
			privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(ctx.String("private-key"), "0x"))
			if err != nil {
				return err
			}

			opts, err := bind.NewKeyedTransactorWithChainID(privateKey, l1ChainID)
			if err != nil {
				return err
			}

			wdTx := bindings.TypesWithdrawalTransaction{
				Nonce:    withdrawal.Nonce,
				Sender:   *withdrawal.Sender,
				Target:   *withdrawal.Target,
				Value:    withdrawal.Value,
				GasLimit: withdrawal.GasLimit,
				Data:     withdrawal.Data,
			}

			tx, err := portal.ProveWithdrawalTransaction(
				opts,
				wdTx,
				l2OutputIndex,
				outputRootProof,
				trieNodes,
			)

			if err != nil {
				return err
			}

			receipt, err := bind.WaitMined(context.Background(), l1Client, tx)
			if err != nil {
				return err
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				return errors.New("withdrawal proof unsuccessful")
			}

			log.Info(fmt.Sprintf("Withdrawal proven (txHash: %v | withdrawalHash: %v)", tx.Hash().Hex(), hash.Hex()))

			// Block the thread for 25s (`hardhat_setTimestamp` is not exposed)
			// The finalization period is 2s, so the extra buffer is just to ensure
			// that we don't try to finalize before the period has elapsed.
			time.Sleep(25 * time.Second)

			// Get the ETH balance of the withdrawal target *before* the finalization
			targetBalBefore, err := l1Client.BalanceAt(context.Background(), common.BytesToAddress(wd.Target.Bytes()), nil)
			if err != nil {
				return err
			}

			log.Debug(fmt.Sprintf("Target balance before finalization: %v", targetBalBefore))

			// Finalize withdrawal
			tx, err = portal.FinalizeWithdrawalTransaction(
				opts,
				wdTx,
			)
			if err != nil {
				return err
			}

			receipt, err = bind.WaitMined(context.Background(), l1Client, tx)
			if err != nil {
				return err
			}
			if receipt.Status != types.ReceiptStatusSuccessful {
				return errors.New("withdrawal finalize unsuccessful")
			}

			log.Info(fmt.Sprintf("Withdrawal Finalized (txHash: %v | withdrawalHash: %v)", tx.Hash(), hash))

			var finalizationTrace callFrame
			tracer := "callTracer"
			traceConfig := tracers.TraceConfig{
				Tracer: &tracer,
			}
			err = l1RpcClient.Call(&finalizationTrace, "debug_traceTransaction", receipt.TxHash, traceConfig)
			if err != nil {
				return err
			}

			traceJson, err := json.MarshalIndent(finalizationTrace, "", "    ")
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("Withdrawal Finalization Trace: %v", string(traceJson)))

			// Get the ETH balance of the withdrawal target *after* the finalization
			targetBalAfter, err := l1Client.BalanceAt(context.Background(), common.BytesToAddress(wd.Target.Bytes()), nil)
			if err != nil {
				return err
			}

			log.Debug(fmt.Sprintf("Target balance after finalization: %v", targetBalAfter))

			// Ensure that the target's balance was increased correctly
			wdValue, err := wd.Value()
			if err != nil {
				return err
			}

			log.Debug(fmt.Sprintf("Withdrawal value: %v", wdValue))
			log.Debug(fmt.Sprintf("Target balance diff: %v", new(big.Int).Sub(targetBalAfter, targetBalBefore)))

			if new(big.Int).Sub(targetBalAfter, targetBalBefore).Cmp(wdValue) != 0 {
				return errors.New("target balance was not increased correctly")
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}
