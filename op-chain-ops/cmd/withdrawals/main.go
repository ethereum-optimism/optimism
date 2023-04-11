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

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// abiTrue represents the storage representation of the boolean
// value true.
var abiTrue = common.Hash{31: 0x01}

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

// BigValue turns a 0x prefixed string into a `big.Int`
func (c *callFrame) BigValue() *big.Int {
	v := strings.TrimPrefix(c.Value, "0x")
	b, _ := new(big.Int).SetString(v, 16)
	return b
}

// suspiciousWithdrawal represents a pending withdrawal that failed for some
// reason after the migration. These are written to disk so that they can
// be manually inspected.
type suspiciousWithdrawal struct {
	Withdrawal *crossdomain.Withdrawal       `json:"withdrawal"`
	Legacy     *crossdomain.LegacyWithdrawal `json:"legacy"`
	Trace      callFrame                     `json:"trace"`
	Index      int                           `json:"index"`
	Reason     string                        `json:"reason"`
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
			clients, err := util.NewClients(ctx)
			if err != nil {
				return err
			}

			// initialize the contract bindings
			contracts, err := newContracts(ctx, clients.L1Client, clients.L2Client)
			if err != nil {
				return err
			}
			l1xdmAddr := common.HexToAddress(ctx.String("l1-crossdomain-messenger-address"))

			l1ChainID, err := clients.L1Client.ChainID(context.Background())
			if err != nil {
				return err
			}

			// create the set of withdrawals
			wds, err := newWithdrawals(ctx, l1ChainID)
			if err != nil {
				return err
			}

			period, err := contracts.L2OutputOracle.FINALIZATIONPERIODSECONDS(&bind.CallOpts{})
			if err != nil {
				return err
			}

			bedrockStartingBlockNumber, err := contracts.L2OutputOracle.StartingBlockNumber(&bind.CallOpts{})
			if err != nil {
				return err
			}

			bedrockStartingBlock, err := clients.L2Client.BlockByNumber(context.Background(), bedrockStartingBlockNumber)
			if err != nil {
				return err
			}

			log.Info("Withdrawal config", "finalization-period", period, "bedrock-starting-block-number", bedrockStartingBlockNumber, "bedrock-starting-block-hash", bedrockStartingBlock.Hash().Hex())

			if !bytes.Equal(bedrockStartingBlock.Extra(), genesis.BedrockTransitionBlockExtraData) {
				return errors.New("genesis block mismatch")
			}

			outfile := ctx.String("bad-withdrawals-out")
			f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o755)
			if err != nil {
				return err
			}

			// create a transactor
			opts, err := newTransactor(ctx)
			if err != nil {
				return err
			}

			// Need this to compare in event parsing
			l1StandardBridgeAddress := common.HexToAddress(ctx.String("l1-standard-bridge-address"))

			// iterate over all of the withdrawals and submit them
			for i, wd := range wds {
				log.Info("Processing withdrawal", "index", i)

				// migrate the withdrawal
				withdrawal, err := crossdomain.MigrateWithdrawal(wd, &l1xdmAddr)
				if err != nil {
					return err
				}

				// Pass to Portal
				hash, err := withdrawal.Hash()
				if err != nil {
					return err
				}

				lcdm := wd.CrossDomainMessage()
				legacyXdmHash, err := lcdm.Hash()
				if err != nil {
					return err
				}

				// check to see if the withdrawal has already been successfully
				// relayed or received
				isSuccess, err := contracts.L1CrossDomainMessenger.SuccessfulMessages(&bind.CallOpts{}, legacyXdmHash)
				if err != nil {
					return err
				}
				isFailed, err := contracts.L1CrossDomainMessenger.FailedMessages(&bind.CallOpts{}, legacyXdmHash)
				if err != nil {
					return err
				}

				xdmHash := crypto.Keccak256Hash(withdrawal.Data)
				if err != nil {
					return err
				}

				isSuccessNew, err := contracts.L1CrossDomainMessenger.SuccessfulMessages(&bind.CallOpts{}, xdmHash)
				if err != nil {
					return err
				}
				isFailedNew, err := contracts.L1CrossDomainMessenger.FailedMessages(&bind.CallOpts{}, xdmHash)
				if err != nil {
					return err
				}

				log.Info("cross domain messenger status", "hash", legacyXdmHash.Hex(), "success", isSuccess, "failed", isFailed, "is-success-new", isSuccessNew, "is-failed-new", isFailedNew)

				// compute the storage slot
				slot, err := withdrawal.StorageSlot()
				if err != nil {
					return err
				}
				// successful messages can be skipped, received messages failed
				// their execution and should be replayed
				if isSuccessNew {
					log.Info("Message already relayed", "index", i, "hash", hash, "slot", slot)
					continue
				}

				// check the storage value of the slot to ensure that it is in
				// the L2 storage. Without this check, the proof will fail
				storageValue, err := clients.L2Client.StorageAt(context.Background(), predeploys.L2ToL1MessagePasserAddr, slot, nil)
				if err != nil {
					return err
				}
				log.Debug("L2ToL1MessagePasser status", "value", common.Bytes2Hex(storageValue))

				// the value should be set to a boolean in storage
				if !bytes.Equal(storageValue, abiTrue.Bytes()) {
					return fmt.Errorf("storage slot %x not found in state", slot)
				}

				legacySlot, err := wd.StorageSlot()
				if err != nil {
					return err
				}
				legacyStorageValue, err := clients.L2Client.StorageAt(context.Background(), predeploys.LegacyMessagePasserAddr, legacySlot, nil)
				if err != nil {
					return err
				}
				log.Debug("LegacyMessagePasser status", "value", common.Bytes2Hex(legacyStorageValue))

				// check to see if its already been proven
				proven, err := contracts.OptimismPortal.ProvenWithdrawals(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}

				// if it has not been proven, then prove it
				if proven.Timestamp.Cmp(common.Big0) == 0 {
					log.Info("Proving withdrawal to OptimismPortal")
					if err := proveWithdrawalTransaction(contracts, clients, opts, withdrawal, bedrockStartingBlockNumber, period); err != nil {
						return err
					}
				} else {
					log.Info("Withdrawal already proven to OptimismPortal")
				}

				// check to see if the withdrawal has been finalized already
				isFinalized, err := contracts.OptimismPortal.FinalizedWithdrawals(&bind.CallOpts{}, hash)
				if err != nil {
					return err
				}

				if !isFinalized {
					// Get the ETH balance of the withdrawal target *before* the finalization
					targetBalBefore, err := clients.L1Client.BalanceAt(context.Background(), wd.XDomainTarget, nil)
					if err != nil {
						return err
					}
					log.Debug("Balance before finalization", "balance", targetBalBefore, "account", wd.XDomainTarget)

					log.Info("Finalizing withdrawal")
					receipt, err := finalizeWithdrawalTransaction(contracts, clients, opts, wd, withdrawal)
					if err != nil {
						return err
					}
					log.Info("withdrawal finalized", "tx-hash", receipt.TxHash, "withdrawal-hash", hash)

					finalizationTrace, err := callTrace(clients, receipt)
					if err != nil {
						return nil
					}

					isSuccessNewPost, err := contracts.L1CrossDomainMessenger.SuccessfulMessages(&bind.CallOpts{}, xdmHash)
					if err != nil {
						return err
					}

					// This would indicate that there is a replayability problem
					if isSuccess && isSuccessNewPost {
						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "should revert"); err != nil {
							return err
						}
						panic("DOUBLE PLAYED DEPOSIT ALLOWED")
					}

					callFrame := findWithdrawalCall(&finalizationTrace, wd, l1xdmAddr)
					if callFrame == nil {
						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "cannot find callframe"); err != nil {
							return err
						}
						continue
					}

					traceJson, err := json.MarshalIndent(callFrame, "", "    ")
					if err != nil {
						return err
					}
					log.Debug(fmt.Sprintf("%v", string(traceJson)))

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
							if err := handleFinalizeERC20Withdrawal(args, receipt, l1StandardBridgeAddress); err != nil {
								return err
							}
						case "finalizeETHWithdrawal":
							if err := handleFinalizeETHWithdrawal(args); err != nil {
								return err
							}
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
						log.Info("unknown method", "to", wd.XDomainTarget, "data", hexutil.Encode(wd.XDomainData))
						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "unknown method"); err != nil {
							return err
						}
					}

					// check that the user's intents are actually executed
					if common.HexToAddress(callFrame.To) != wd.XDomainTarget {
						log.Info("target mismatch", "index", i)

						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "target mismatch"); err != nil {
							return err
						}
						continue
					}
					if !bytes.Equal(hexutil.MustDecode(callFrame.Input), wd.XDomainData) {
						log.Info("calldata mismatch", "index", i)

						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "calldata mismatch"); err != nil {
							return err
						}
						continue
					}
					if callFrame.BigValue().Cmp(wdValue) != 0 {
						log.Info("value mismatch", "index", i)
						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "value mismatch"); err != nil {
							return err
						}
						continue
					}

					// Get the ETH balance of the withdrawal target *after* the finalization
					targetBalAfter, err := clients.L1Client.BalanceAt(context.Background(), wd.XDomainTarget, nil)
					if err != nil {
						return err
					}

					diff := new(big.Int).Sub(targetBalAfter, targetBalBefore)
					log.Debug("balances", "before", targetBalBefore, "after", targetBalAfter, "diff", diff)

					isSuccessNewPost, err = contracts.L1CrossDomainMessenger.SuccessfulMessages(&bind.CallOpts{}, xdmHash)
					if err != nil {
						return err
					}

					if diff.Cmp(wdValue) != 0 && isSuccessNewPost && isSuccess {
						log.Info("native eth balance diff mismatch", "index", i, "diff", diff, "val", wdValue)
						if err := writeSuspicious(f, withdrawal, wd, finalizationTrace, i, "balance mismatch"); err != nil {
							return err
						}
						continue
					}
				} else {
					log.Info("Already finalized")
				}
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}

// callTrace will call `debug_traceTransaction` on a remote node
func callTrace(c *util.Clients, receipt *types.Receipt) (callFrame, error) {
	var finalizationTrace callFrame
	tracer := "callTracer"
	traceConfig := tracers.TraceConfig{
		Tracer: &tracer,
	}
	err := c.L1RpcClient.Call(&finalizationTrace, "debug_traceTransaction", receipt.TxHash, traceConfig)
	if err != nil {
		return finalizationTrace, err
	}
	return finalizationTrace, err
}

// handleFinalizeETHWithdrawal will ensure that the calldata is correct
func handleFinalizeETHWithdrawal(args []any) error {
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

	return nil
}

// handleFinalizeERC20Withdrawal will look at the receipt logs and make
// assertions that the values are correct
func handleFinalizeERC20Withdrawal(args []any, receipt *types.Receipt, l1StandardBridgeAddress common.Address) error {
	erc20Abi, err := bindings.ERC20MetaData.GetAbi()
	if err != nil {
		return err
	}
	transferEvent := erc20Abi.Events["Transfer"]

	// Handle logic for ERC20 withdrawals
	l1Token, ok := args[0].(common.Address)
	if !ok {
		return fmt.Errorf("invalid abi")
	}
	l2Token, ok := args[1].(common.Address)
	if !ok {
		return fmt.Errorf("invalid abi")
	}
	from, ok := args[2].(common.Address)
	if !ok {
		return fmt.Errorf("invalid abi")
	}
	to, ok := args[3].(common.Address)
	if !ok {
		return fmt.Errorf("invalid abi")
	}
	amount, ok := args[4].(*big.Int)
	if !ok {
		return fmt.Errorf("invalid abi")
	}
	extraData, ok := args[5].([]byte)
	if !ok {
		return fmt.Errorf("invalid abi")
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
			if l.Address == l1Token {
				a, _ := transferEvent.Inputs.Unpack(l.Data)
				if len(l.Topics) < 3 {
					return fmt.Errorf("")
				}

				_from := common.BytesToAddress(l.Topics[1].Bytes())
				_to := common.BytesToAddress(l.Topics[2].Bytes())

				// from the L1StandardBridge
				if _from != l1StandardBridgeAddress {
					return fmt.Errorf("from mismatch: %x - %x", _from, l1StandardBridgeAddress)
				}
				if to != _to {
					return fmt.Errorf("to mismatch: %x - %x", to, _to)
				}
				_amount, ok := a[0].(*big.Int)
				if !ok {
					return fmt.Errorf("invalid abi in transfer event")
				}
				if amount.Cmp(_amount) != 0 {
					return fmt.Errorf("amount mismatch: %d - %d", amount, _amount)
				}
			}
		}
	}

	return nil
}

// proveWithdrawalTransaction will build the data required for proving a
// withdrawal and then send the transaction and make sure that it is included
// and successful and then wait for the finalization period to elapse.
func proveWithdrawalTransaction(c *contracts, cl *util.Clients, opts *bind.TransactOpts, withdrawal *crossdomain.Withdrawal, bn, finalizationPeriod *big.Int) error {
	l2OutputIndex, outputRootProof, trieNodes, err := createOutput(withdrawal, c.L2OutputOracle, bn, cl)
	if err != nil {
		return err
	}

	hash, err := withdrawal.Hash()
	if err != nil {
		return err
	}
	wdTx := withdrawal.WithdrawalTransaction()

	tx, err := c.OptimismPortal.ProveWithdrawalTransaction(
		opts,
		wdTx,
		l2OutputIndex,
		outputRootProof,
		trieNodes,
	)

	if err != nil {
		return err
	}

	receipt, err := bind.WaitMined(context.Background(), cl.L1Client, tx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return errors.New("withdrawal proof unsuccessful")
	}

	log.Info("withdrawal proved", "tx-hash", tx.Hash(), "withdrawal-hash", hash)

	block, err := cl.L1Client.BlockByHash(context.Background(), receipt.BlockHash)
	if err != nil {
		return err
	}
	initialTime := block.Time()
	for {
		log.Info("waiting for finalization")
		if block.Time() >= initialTime+finalizationPeriod.Uint64() {
			log.Info("can be finalized")
			break
		}
		time.Sleep(1 * time.Second)
		block, err = cl.L1Client.BlockByNumber(context.Background(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func finalizeWithdrawalTransaction(
	c *contracts,
	cl *util.Clients,
	opts *bind.TransactOpts,
	wd *crossdomain.LegacyWithdrawal,
	withdrawal *crossdomain.Withdrawal,
) (*types.Receipt, error) {
	if wd.XDomainTarget == (common.Address{}) {
		return nil, errors.New("withdrawal target is nil, should never happen")
	}

	wdTx := withdrawal.WithdrawalTransaction()

	// Finalize withdrawal
	tx, err := c.OptimismPortal.FinalizeWithdrawalTransaction(
		opts,
		wdTx,
	)
	if err != nil {
		return nil, err
	}

	receipt, err := bind.WaitMined(context.Background(), cl.L1Client, tx)
	if err != nil {
		return nil, err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("withdrawal finalize unsuccessful")
	}
	return receipt, nil
}

// contracts represents a set of bound contracts
type contracts struct {
	OptimismPortal         *bindings.OptimismPortal
	L1CrossDomainMessenger *bindings.L1CrossDomainMessenger
	L2OutputOracle         *bindings.L2OutputOracle
}

// newContracts will create a contracts struct with the contract bindings
// preconfigured
func newContracts(ctx *cli.Context, l1Backend, l2Backend bind.ContractBackend) (*contracts, error) {
	optimismPortalAddress := ctx.String("optimism-portal-address")
	if len(optimismPortalAddress) == 0 {
		return nil, errors.New("OptimismPortal address not configured")
	}
	optimismPortalAddr := common.HexToAddress(optimismPortalAddress)

	portal, err := bindings.NewOptimismPortal(optimismPortalAddr, l1Backend)
	if err != nil {
		return nil, err
	}

	l1xdmAddress := ctx.String("l1-crossdomain-messenger-address")
	if l1xdmAddress == "" {
		return nil, errors.New("L1CrossDomainMessenger address not configured")
	}
	l1xdmAddr := common.HexToAddress(l1xdmAddress)

	l1CrossDomainMessenger, err := bindings.NewL1CrossDomainMessenger(l1xdmAddr, l1Backend)
	if err != nil {
		return nil, err
	}

	l2OracleAddr, err := portal.L2ORACLE(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	oracle, err := bindings.NewL2OutputOracle(l2OracleAddr, l1Backend)
	if err != nil {
		return nil, err
	}

	log.Info(
		"Addresses",
		"l1-crossdomain-messenger", l1xdmAddr,
		"optimism-portal", optimismPortalAddr,
		"l2-output-oracle", l2OracleAddr,
	)

	return &contracts{
		OptimismPortal:         portal,
		L1CrossDomainMessenger: l1CrossDomainMessenger,
		L2OutputOracle:         oracle,
	}, nil
}

// newWithdrawals will create a set of legacy withdrawals
func newWithdrawals(ctx *cli.Context, l1ChainID *big.Int) ([]*crossdomain.LegacyWithdrawal, error) {
	ovmMsgs := ctx.String("ovm-messages")
	evmMsgs := ctx.String("evm-messages")

	log.Debug("Migration data", "ovm-path", ovmMsgs, "evm-messages", evmMsgs)
	ovmMessages, err := crossdomain.NewSentMessageFromJSON(ovmMsgs)
	if err != nil {
		return nil, err
	}

	// use empty ovmMessages if its not mainnet. The mainnet messages are
	// committed to in git.
	if l1ChainID.Cmp(common.Big1) != 0 {
		log.Info("not using ovm messages because its not mainnet")
		ovmMessages = []*crossdomain.SentMessage{}
	}

	evmMessages, err := crossdomain.NewSentMessageFromJSON(evmMsgs)
	if err != nil {
		return nil, err
	}

	migrationData := crossdomain.MigrationData{
		OvmMessages: ovmMessages,
		EvmMessages: evmMessages,
	}

	wds, _, err := migrationData.ToWithdrawals()
	if err != nil {
		return nil, err
	}
	if len(wds) == 0 {
		return nil, errors.New("no withdrawals")
	}
	log.Info("Converted migration data to withdrawals successfully", "count", len(wds))

	return wds, nil
}

// newTransactor creates a new transact context given a cli context
func newTransactor(ctx *cli.Context) (*bind.TransactOpts, error) {
	if ctx.String("private-key") == "" {
		return nil, errors.New("No private key to transact with")
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(ctx.String("private-key"), "0x"))
	if err != nil {
		return nil, err
	}

	l1RpcURL := ctx.String("l1-rpc-url")
	l1Client, err := ethclient.Dial(l1RpcURL)
	if err != nil {
		return nil, err
	}
	l1ChainID, err := l1Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, l1ChainID)
	if err != nil {
		return nil, err
	}
	return opts, nil
}

// findWithdrawalCall will find the call frame for the call that
// represents the user's intent.
func findWithdrawalCall(trace *callFrame, wd *crossdomain.LegacyWithdrawal, l1xdm common.Address) *callFrame {
	isCall := trace.Type == "CALL"
	isTarget := common.HexToAddress(trace.To) == wd.XDomainTarget
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

// createOutput will create the data required to send a withdrawal transaction.
func createOutput(
	withdrawal *crossdomain.Withdrawal,
	oracle *bindings.L2OutputOracle,
	blockNumber *big.Int,
	clients *util.Clients,
) (*big.Int, bindings.TypesOutputRootProof, [][]byte, error) {
	// compute the storage slot that the withdrawal is stored in
	slot, err := withdrawal.StorageSlot()
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}

	// find the output index that the withdrawal was committed to in
	l2OutputIndex, err := oracle.GetL2OutputIndexAfter(&bind.CallOpts{}, blockNumber)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}
	// fetch the output the commits to the withdrawal using the index
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

	// get the block header committed to in the output
	header, err := clients.L2Client.HeaderByNumber(context.Background(), l2Output.L2BlockNumber)
	if err != nil {
		return nil, bindings.TypesOutputRootProof{}, nil, err
	}

	// get the storage proof for the withdrawal's storage slot
	proof, err := clients.L2GethClient.GetProof(context.Background(), predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, blockNumber)

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

	// create an output root proof
	outputRootProof := bindings.TypesOutputRootProof{
		Version:                  [32]byte{},
		StateRoot:                header.Root,
		MessagePasserStorageRoot: proof.StorageHash,
		LatestBlockhash:          header.Hash(),
	}

	// TODO(mark): import the function from `op-node` to compute the hash
	// instead of doing this. Will update when testing against mainnet.
	localOutputRootHash := crypto.Keccak256Hash(
		outputRootProof.Version[:],
		outputRootProof.StateRoot[:],
		outputRootProof.MessagePasserStorageRoot[:],
		outputRootProof.LatestBlockhash[:],
	)

	// ensure that the locally computed hash matches
	if l2Output.OutputRoot != localOutputRootHash {
		return nil, bindings.TypesOutputRootProof{}, nil, fmt.Errorf("mismatch in output root hashes, got 0x%x expected 0x%x", localOutputRootHash, l2Output.OutputRoot)
	}
	log.Info(
		"output root proof",
		"version", common.Hash(outputRootProof.Version),
		"state-root", common.Hash(outputRootProof.StateRoot),
		"storage-root", common.Hash(outputRootProof.MessagePasserStorageRoot),
		"block-hash", common.Hash(outputRootProof.LatestBlockhash),
		"trie-node-count", len(trieNodes),
	)

	return l2OutputIndex, outputRootProof, trieNodes, nil
}

// writeSuspicious will create a suspiciousWithdrawal and then append it to a
// JSONL file. Each line is its own JSON where there is a newline separating them.
func writeSuspicious(
	f *os.File,
	withdrawal *crossdomain.Withdrawal,
	wd *crossdomain.LegacyWithdrawal,
	finalizationTrace callFrame,
	i int,
	reason string,
) error {
	bad := suspiciousWithdrawal{
		Withdrawal: withdrawal,
		Legacy:     wd,
		Trace:      finalizationTrace,
		Index:      i,
		Reason:     reason,
	}

	data, err := json.Marshal(bad)
	if err != nil {
		return err
	}
	_, err = f.WriteString(string(data) + "\n")
	return err
}
