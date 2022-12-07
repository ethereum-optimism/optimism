package main

import (
	"bytes"
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

	"github.com/ethereum-optimism/optimism/l2geth/common/hexutil"
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

func findWithdrawalCall(trace *callFrame, wd *crossdomain.LegacyWithdrawal, l1xdm common.Address) *callFrame {
	isCall := trace.Type == "CALL"
	isTarget := common.HexToAddress(trace.To) == *wd.Target
	isFrom := common.HexToAddress(trace.From) == l1xdm
	if isCall && isTarget && isFrom {
		return trace
	}
	for _, subcall := range trace.Calls {
		if call := findWithdrawalCall(&subcall, wd, l1xdm); call != nil {
			return call
		}
	}
	return nil
}

func createOutput(
	withdrawal *crossdomain.Withdrawal,
	oracle *bindings.L2OutputOracle,
	blockNumber *big.Int,
	l2Client bind.ContractBackend,
	l2GethClient *gethclient.Client,
) (*big.Int, bindings.TypesOutputRootProof, [][]byte, error) {
	slot, err := withdrawal.StorageSlot()
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}

	l2OutputIndex, err := oracle.GetL2OutputIndexAfter(&bind.CallOpts{}, blockNumber)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}
	l2Output, err := oracle.GetL2Output(&bind.CallOpts{}, l2OutputIndex)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}

	log.Debug(
		"L2 output",
		"index", l2OutputIndex,
		"root", common.Bytes2Hex(l2Output.OutputRoot[:]),
		"l2-blocknumber", l2Output.L2BlockNumber,
		"timestamp", l2Output.Timestamp,
	)

	header, err := l2Client.HeaderByNumber(context.Background(), l2Output.L2BlockNumber)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}

	proof, err := l2GethClient.GetProof(context.Background(), predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, blockNumber)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}
	if count := len(proof.StorageProof); count != 1 {
		return nil, bindings.TypesOutputRootProof{}, nil, fmt.Errorf("invalid amount of storage proofs: %d", count)
	}
	trieNodes := make([][]byte, len(proof.StorageProof[0].Proof))
	for i, s := range proof.StorageProof[0].Proof {
		trieNodes[i] = common.FromHex(s)
	}

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

	if l2Output.OutputRoot != localOutputRootHash {
		return nil, bindings.TypesOutputRootProof{}, nil, fmt.Errorf("mismatch in output root hashes", "got", localOutputRootHash, "expect", l2Output.OutputRoot)
	}
	log.Info(
		"output root proof",
		"version", common.Hash(outputRootProof.Version),
		"state-root", common.Hash(outputRootProof.StateRoot),
		"storage-root", common.Hash(outputRootProof.MessagePasserStorageRoot),
		"block-hash", common.Hash(outputRootProof.LatestBlockhash),
	)

	return l2OutputIndex, outputRootProof, trieNodes, nil
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
				Name:  "l1-standard-bridge-address",
				Usage: "Address of the L1StandardBridge",
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
			&cli.StringFlag{
				Name:  "bad-withdrawals-out",
				Value: "bad-withdrawals.json",
				Usage: "Path to write JSON file of bad withdrawals to manually inspect",
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

			ovmMsgs := ctx.String("ovm-messages")
			evmMsgs := ctx.String("evm-messages")

			log.Debug("Migration data", "ovm-path", ovmMsgs, "evm-messages", evmMsgs)
			ovmMessages, err := migration.NewSentMessage(ovmMsgs)
			if err != nil {
				return err
			}
			evmMessages, err := migration.NewSentMessage(evmMsgs)
			if err != nil {
				return err
			}

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
			log.Info("Converted migration data to withdrawals successfully", "count", len(wds))

			l1xdmAddress := ctx.String("l1-crossdomain-messenger-address")
			if l1xdmAddress == "" {
				return errors.New("Must pass in --l1-crossdomain-messenger-address")
			}
			l1xdmAddr := common.HexToAddress(l1xdmAddress)

			l1CrossDomainMessenger, err := bindings.NewL1CrossDomainMessenger(l1xdmAddr, l1Client)
			if err != nil {
				return err
			}

			portal, err := bindings.NewOptimismPortal(optimismPortalAddr, l1Client)
			if err != nil {
				return err
			}

			l2OracleAddr, err := portal.L2ORACLE(&bind.CallOpts{})
			if err != nil {
				return err
			}
			oracle, err := bindings.NewL2OutputOracle(l2OracleAddr, l1Client)
			if err != nil {
				return nil
			}

			log.Info(
				"Addresses",
				"l1-crossdomain-messenger", l1xdmAddr,
				"optimism-portal", optimismPortalAddr,
				"output-oracle", l2OracleAddr,
			)

			period, err := portal.FINALIZATIONPERIODSECONDS(&bind.CallOpts{})
			if err != nil {
				return err
			}

			transitionBlockNumber := new(big.Int).SetUint64(ctx.Uint64("bedrock-transition-block-number"))
			log.Info("Withdrawal config", "finalization-period", period, "bedrock-transition-block-number", transitionBlockNumber)

			if ctx.String("private-key") == "" {
				return errors.New("No private key to transact with")
			}
			privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(ctx.String("private-key"), "0x"))
			if err != nil {
				return err
			}

			type badWithdrawal struct {
				Withdrawal *crossdomain.Withdrawal       `json:"withdrawal"`
				Legacy     *crossdomain.LegacyWithdrawal `json:"legacy"`
				Trace      callFrame                     `json:"trace"`
				Index      int                           `json"index"`
			}

			badWithdrawals := make([]badWithdrawal, 0)

			for i, wd := range wds {
				log.Info("Processing withdrawal", "index", i)

				withdrawal, err := crossdomain.MigrateWithdrawal(wd, &l1xdmAddr)
				if err != nil {
					return err
				}

				hash, err := withdrawal.Hash()
				if err != nil {
					return err
				}

				isSuccess, err := l1CrossDomainMessenger.SuccessfulMessages(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}
				isReceived, err := l1CrossDomainMessenger.ReceivedMessages(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}

				slot, err := withdrawal.StorageSlot()
				if err != nil {
					return err
				}

				log.Info("cross domain messenger status", "hash", hash.Hex(), "success", isSuccess, "received", isReceived, "slot", slot.Hex())

				if isSuccess {
					log.Info("Message already relayed", "index", i, "hash", hash, "slot", slot)
					continue
				}

				l2OutputIndex, outputRootProof, trieNodes, err := createOutput(withdrawal, oracle, transitionBlockNumber, l2Client, gclient)
				if err != nil {
					return err
				}

				opts, err := bind.NewKeyedTransactorWithChainID(privateKey, l1ChainID)
				if err != nil {
					return err
				}

				proven, err := portal.ProvenWithdrawals(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}

				wdTx := withdrawal.WithdrawalTransaction()

				// check to see if its been proven
				// if it has not been proven, then prove it
				if proven.Timestamp.Cmp(common.Big0) == 0 {
					log.Info("Proving withdrawal to OptimismPortal")

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

					log.Info("withdrawal proved", "tx-hash", tx.Hash(), "withdrawal-hash", hash)

					block, err := l1Client.BlockByHash(context.Background(), receipt.BlockHash)
					if err != nil {
						return err
					}
					initialTime := block.Time()
					for {
						log.Info("waiting for finalization")
						if block.Time() >= initialTime+period.Uint64() {
							log.Info("can be finalized")
							break
						}
						time.Sleep(1 * time.Second)
						block, err = l1Client.BlockByNumber(context.Background(), nil)
						if err != nil {
							return err
						}
					}
				} else {
					log.Info("Withdrawal already proven to OptimismPortal")
				}

				isFinalized, err := portal.FinalizedWithdrawals(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}

				if !isFinalized {
					log.Info("Finalizing withdrawal")

					// Get the ETH balance of the withdrawal target *before* the finalization
					targetBalBefore, err := l1Client.BalanceAt(context.Background(), common.BytesToAddress(wd.Target.Bytes()), nil)
					if err != nil {
						return err
					}

					log.Debug(fmt.Sprintf("Target balance before finalization: %v", targetBalBefore))

					// Finalize withdrawal
					tx, err := portal.FinalizeWithdrawalTransaction(
						opts,
						wdTx,
					)
					if err != nil {
						return err
					}

					receipt, err := bind.WaitMined(context.Background(), l1Client, tx)
					if err != nil {
						return err
					}
					if receipt.Status != types.ReceiptStatusSuccessful {
						return errors.New("withdrawal finalize unsuccessful")
					}

					log.Info("withdrawal finalized", "tx-hash", tx.Hash(), "withdrawal-hash", hash)

					// fetch the call trace
					var finalizationTrace callFrame
					tracer := "callTracer"
					traceConfig := tracers.TraceConfig{
						Tracer: &tracer,
					}
					err = l1RpcClient.Call(&finalizationTrace, "debug_traceTransaction", receipt.TxHash, traceConfig)
					if err != nil {
						return err
					}

					callFrame := findWithdrawalCall(&finalizationTrace, wd, l1xdmAddr)
					if callFrame == nil {
						return errors.New("cannot find callframe")
					}

					traceJson, err := json.MarshalIndent(callFrame, "", "    ")
					if err != nil {
						return err
					}
					log.Info(fmt.Sprintf("%v", string(traceJson)))

					erc20Abi, err := bindings.ERC20MetaData.GetAbi()
					if err != nil {
						return err
					}
					transferEvent := erc20Abi.Events["Transfer"]

					abi, err := bindings.L1StandardBridgeMetaData.GetAbi()
					if err != nil {
						return err
					}

					calldata := hexutil.MustDecode(callFrame.Input)

					// this must be the L1 standard bridge
					method, err := abi.MethodById(calldata)
					// Handle L1StandardBridge specific logic
					if err == nil {
						args, err := method.Inputs.Unpack(calldata[4:])
						if err != nil {
							return err
						}

						log.Info("decoded calldata", "name", method.Name)

						switch method.Name {
						case "finalizeERC20Withdrawal":
							l1Token, ok := args[0].(common.Address)
							if !ok {
								return fmt.Errorf("")
							}
							l2Token, ok := args[1].(common.Address)
							if !ok {
								return fmt.Errorf("")
							}
							from, ok := args[2].(common.Address)
							if !ok {
								return fmt.Errorf("")
							}
							to, ok := args[3].(common.Address)
							if !ok {
								return fmt.Errorf("")
							}
							amount, ok := args[4].(*big.Int)
							if !ok {
								return fmt.Errorf("")
							}
							extraData, ok := args[5].([]byte)
							if !ok {
								return fmt.Errorf("")
							}

							log.Info(
								"decoded calldata",
								"l1Token", l1Token,
								"l2Token", l2Token,
								"from", from,
								"to", to,
								"amount", amount,
								"extraData", extraData,
							)

							// Look for the ERC20 token transfer topic
							for _, l := range receipt.Logs {
								topic := l.Topics[0]
								if topic == transferEvent.ID {
									a, _ := transferEvent.Inputs.Unpack(l.Data)
									// TODO: add a check here for balance diff as
									// expected
									log.Info("EVENT FOUND", "args", a)
								}
								log.Info("receipt topic", "hex", topic.Hex())
							}

						case "finalizeETHWithdrawal":
							from, ok := args[0].(common.Address)
							if !ok {
								return fmt.Errorf("invalid type: from")
							}
							to, ok := args[1].(common.Address)
							if !ok {
								return fmt.Errorf("invalid type: to")
							}
							amount, ok := args[2].(*big.Int)
							if !ok {
								return fmt.Errorf("invalid type: amount")
							}
							extraData, ok := args[3].([]byte)
							if !ok {
								return fmt.Errorf("invalid type: extraData")
							}

							log.Info(
								"decoded calldata",
								"from", from,
								"to", to,
								"amount", amount,
								"extraData", extraData,
							)

						default:
							log.Info("Unhandled method", "name", method.Name)
						}
					}

					// Ensure that the target's balance was increasedData correctly
					wdValue, err := wd.Value()
					if err != nil {
						return err
					}
					if method != nil {
						log.Info("withdrawal action", "function", method.Name, "value", wdValue)
					} else {
						log.Info("unknown method", "to", wd.Target, "data", hexutil.Encode(wd.Data))

						badWithdrawals = append(badWithdrawals, badWithdrawal{
							Withdrawal: withdrawal,
							Legacy:     wd,
							Trace:      finalizationTrace,
							Index:      i,
						})

					}

					if common.HexToAddress(callFrame.To) != *wd.Target {
						badWithdrawals = append(badWithdrawals, badWithdrawal{
							Withdrawal: withdrawal,
							Legacy:     wd,
							Trace:      finalizationTrace,
							Index:      i,
						})

						log.Info("target mismatch", "index", i)

						continue
					}
					if !bytes.Equal(hexutil.MustDecode(callFrame.Input), wd.Data) {
						badWithdrawals = append(badWithdrawals, badWithdrawal{
							Withdrawal: withdrawal,
							Legacy:     wd,
							Trace:      finalizationTrace,
							Index:      i,
						})

						log.Info("calldata mismatch", "index", i)

						continue
					}
					if callFrame.Value != "0x"+wdValue.Text(16) {
						badWithdrawals = append(badWithdrawals, badWithdrawal{
							Withdrawal: withdrawal,
							Legacy:     wd,
							Trace:      finalizationTrace,
							Index:      i,
						})

						log.Info("value mismatch", "index", i)

						continue
					}

					// Get the ETH balance of the withdrawal target *after* the finalization
					targetBalAfter, err := l1Client.BalanceAt(context.Background(), *wd.Target, nil)
					if err != nil {
						return err
					}

					diff := new(big.Int).Sub(targetBalAfter, targetBalBefore)
					log.Debug("balances", "before", targetBalBefore, "after", targetBalAfter, "diff", diff)

					if diff.Cmp(wdValue) != 0 {
						badWithdrawals = append(badWithdrawals, badWithdrawal{
							Withdrawal: withdrawal,
							Legacy:     wd,
							Trace:      finalizationTrace,
							Index:      i,
						})

						log.Info("native eth balance diff mismatch", "index", i)

						continue
					}
				}
			}

			// Write the stuff to disk
			if err := writeJSON(ctx.String("bad-withdrawals-out"), badWithdrawals); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

func writeJSON(outfile string, input interface{}) error {
	f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(input)
}
