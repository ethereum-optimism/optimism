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

	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"

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

				addressManager, err := bindings.NewAddressManager(addresses.AddressManager, clients.L1Client)
				if err != nil {
					return err
				}

				for {
					shutoffBlock, err := addressManager.GetAddress(&bind.CallOpts{}, "DTL_SHUTOFF_BLOCK")
					if err != nil {
						return err
					}
					if shutoffBlock.Big().Cmp(common.Big0) != 0 {
						break
					}
					time.Sleep(3 * time.Second)
				}

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

				log.Info("Searching backwards for final deposit")
				blockNumber, err := clients.L2Client.BlockNumber(context.Background())
				if err != nil {
					return err
				}

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

				ctc, err := legacy_bindings.NewCanonicalTransactionChain(addresses.CanonicalTransactionChain, clients.L1Client)
				if err != nil {
					return err
				}

				scc, err := legacy_bindings.NewStateCommitmentChain(addresses.StateCommitmentChain, clients.L1Client)
				if err != nil {
					return err
				}

				var wg sync.WaitGroup

				log.Info("Waiting for CanonicalTransactionChain")
				wg.Add(1)
				go waitForTotalElements(wg, ctc, clients.L2Client)

				log.Info("Waiting for StateCommitmentChain")
				wg.Add(1)
				go waitForTotalElements(wg, scc, clients.L2Client)

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
func waitForTotalElements(wg sync.WaitGroup, contract RollupContract, client *ethclient.Client) error {
	defer wg.Done()

	for {
		bn, err := client.BlockNumber(context.Background())
		if err != nil {
			return err
		}
		totalElements, err := contract.GetTotalElements(&bind.CallOpts{})
		if err != nil {
			return err
		}

		if totalElements.Cmp(bn) == 0 {
			return nil
		}
		log.Info("Waiting for elements to be submitted", "count", totalElements.Uint64()-bn.Uint64(), "height", bn.Uint64(), "total-elements", totalElements.Uint64())

		time.Sleep(3 * time.Second)
	}

	return nil
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

// toBlockNumArg is able to handle the conversion between a big.Int and a
// string blocktag. This function should be used in JSON RPC interactions
// when fetching data by block number.
func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	finalized := big.NewInt(int64(rpc.FinalizedBlockNumber))
	if number.Cmp(finalized) == 0 {
		return "finalized"
	}
	safe := big.NewInt(int64(rpc.SafeBlockNumber))
	if number.Cmp(safe) == 0 {
		return "safe"
	}
	return hexutil.EncodeBig(number)
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
