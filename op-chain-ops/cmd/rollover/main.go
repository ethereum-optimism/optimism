package main

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/op-chain-ops/util"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "rollover"
	app.Usage = "Commands for assisting in the rollover of the system"

	var flags []cli.Flag
	flags = append(flags, util.ClientsFlags...)
	flags = append(flags, util.AddressesFlags...)

	app.Commands = []*cli.Command{
		{
			Name:  "deposits",
			Usage: "Ensures that all deposits have been ingested into L2",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				clients, err := util.NewClients(cliCtx)
				if err != nil {
					return err
				}

				addresses, err := util.NewAddresses(cliCtx)
				if err != nil {
					return err
				}

				log.Info("Connecting to AddressManager", "address", addresses.AddressManager)
				addressManager, err := bindings.NewAddressManager(addresses.AddressManager, clients.L1Client)
				if err != nil {
					return err
				}

				for {
					shutoffBlock, err := addressManager.GetAddress(&bind.CallOpts{}, "DTL_SHUTOFF_BLOCK")
					if err != nil {
						return err
					}
					if num := shutoffBlock.Big(); num.Cmp(common.Big0) != 0 {
						log.Info("DTL_SHUTOFF_BLOCK is set", "number", num.Uint64())
						break
					}
					log.Info("DTL_SHUTOFF_BLOCK not set yet")
					time.Sleep(3 * time.Second)
				}

				log.Info("Connecting to CanonicalTransactionChain", "address", addresses.CanonicalTransactionChain)
				ctc, err := legacy_bindings.NewCanonicalTransactionChain(addresses.CanonicalTransactionChain, clients.L1Client)
				if err != nil {
					return err
				}

				queueLength, err := ctc.GetQueueLength(&bind.CallOpts{})
				if err != nil {
					return err
				}

				totalElements, err := ctc.GetTotalElements(&bind.CallOpts{})
				if err != nil {
					return err
				}

				totalBatches, err := ctc.GetTotalBatches(&bind.CallOpts{})
				if err != nil {
					return err
				}

				pending, err := ctc.GetNumPendingQueueElements(&bind.CallOpts{})
				if err != nil {
					return err
				}

				log.Info(
					"CanonicalTransactionChain",
					"address", addresses.CanonicalTransactionChain,
					"queue-length", queueLength,
					"total-elements", totalElements,
					"total-batches", totalBatches,
					"pending", pending,
				)

				blockNumber, err := clients.L2Client.BlockNumber(context.Background())
				if err != nil {
					return err
				}
				log.Info("Searching backwards for final deposit", "start", blockNumber)

				for {
					bn := new(big.Int).SetUint64(blockNumber)
					log.Info("Checking L2 block", "number", bn)

					block, err := clients.L2Client.BlockByNumber(context.Background(), bn)
					if err != nil {
						return err
					}

					if length := len(block.Transactions()); length != 1 {
						return fmt.Errorf("unexpected number of transactions in block: %d", length)
					}

					tx := block.Transactions()[0]
					hash := tx.Hash()
					json, err := legacyTransactionByHash(clients.L2RpcClient, hash)
					if err != nil {
						return err
					}
					if json.QueueOrigin == "l1" {
						if json.QueueIndex == nil {
							// This should never happen
							return errors.New("queue index is nil")
						}
						queueIndex := uint64(*json.QueueIndex)
						if queueIndex == queueLength.Uint64()-1 {
							log.Info("Found final deposit in l2geth", "queue-index", queueIndex)
							break
						}
						if queueIndex < queueLength.Uint64() {
							return errors.New("missed final deposit")
						}
					}
					blockNumber--
				}

				finalPending, err := ctc.GetNumPendingQueueElements(&bind.CallOpts{})
				if err != nil {
					return err
				}
				log.Info("Remaining deposits that must be submitted", "count", finalPending)
				return nil
			},
		},
		{
			Name:  "batches",
			Usage: "Ensures that all batches have been submitted to L1",
			Flags: flags,
			Action: func(cliCtx *cli.Context) error {
				clients, err := util.NewClients(cliCtx)
				if err != nil {
					return err
				}

				addresses, err := util.NewAddresses(cliCtx)
				if err != nil {
					return err
				}

				log.Info("Connecting to CanonicalTransactionChain", "address", addresses.CanonicalTransactionChain)
				ctc, err := legacy_bindings.NewCanonicalTransactionChain(addresses.CanonicalTransactionChain, clients.L1Client)
				if err != nil {
					return err
				}

				log.Info("Connecting to StateCommitmentChain", "address", addresses.StateCommitmentChain)
				scc, err := legacy_bindings.NewStateCommitmentChain(addresses.StateCommitmentChain, clients.L1Client)
				if err != nil {
					return err
				}

				var wg sync.WaitGroup

				log.Info("Waiting for CanonicalTransactionChain")
				wg.Add(1)
				go waitForTotalElements(&wg, ctc, clients.L2Client)

				log.Info("Waiting for StateCommitmentChain")
				wg.Add(1)
				go waitForTotalElements(&wg, scc, clients.L2Client)

				wg.Wait()
				log.Info("All batches have been submitted")

				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// RollupContract represents a legacy rollup contract interface that
// exposes the GetTotalElements function. Both the StateCommitmentChain
// and the CanonicalTransactionChain implement this interface.
type RollupContract interface {
	GetTotalElements(opts *bind.CallOpts) (*big.Int, error)
}

// waitForTotalElements will poll to see
func waitForTotalElements(wg *sync.WaitGroup, contract RollupContract, client *ethclient.Client) {
	defer wg.Done()

	for {
		bn, err := client.BlockNumber(context.Background())
		if err != nil {
			log.Error("cannot fetch blocknumber", "error", err)
			time.Sleep(3 * time.Second)
			continue
		}
		totalElements, err := contract.GetTotalElements(&bind.CallOpts{})
		if err != nil {
			log.Error("cannot fetch total elements", "error", err)
			time.Sleep(3 * time.Second)
			continue
		}

		if totalElements.Uint64() == bn {
			return
		}
		log.Info("Waiting for elements to be submitted", "count", totalElements.Uint64()-bn, "height", bn, "total-elements", totalElements.Uint64())

		time.Sleep(3 * time.Second)
	}
}

// legacyTransactionByHash will fetch a transaction by hash and be sure to decode
// the additional fields added to legacy transactions.
func legacyTransactionByHash(client *rpc.Client, hash common.Hash) (*RPCTransaction, error) {
	var json *RPCTransaction
	err := client.CallContext(context.Background(), &json, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	}
	return json, nil
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a
// transaction. This handles the extra legacy fields added to transactions.
type RPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
	QueueOrigin      string          `json:"queueOrigin"`
	L1TxOrigin       *common.Address `json:"l1TxOrigin"`
	L1BlockNumber    *hexutil.Big    `json:"l1BlockNumber"`
	L1Timestamp      hexutil.Uint64  `json:"l1Timestamp"`
	Index            *hexutil.Uint64 `json:"index"`
	QueueIndex       *hexutil.Uint64 `json:"queueIndex"`
	RawTransaction   hexutil.Bytes   `json:"rawTransaction"`
}
