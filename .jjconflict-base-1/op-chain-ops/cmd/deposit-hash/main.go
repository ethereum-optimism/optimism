package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	var rpcURL, txHash string
	flag.StringVar(&rpcURL, "rpc", "", "L1 RPC URL")
	flag.StringVar(&txHash, "tx", "", "Deposit transaction hash on L1")
	flag.Parse()

	// Validate required parameters
	if rpcURL == "" {
		log.Crit("RPC URL is required. Use --rpc flag to specify L1 RPC URL")
	}
	if txHash == "" {
		log.Crit("Transaction hash is required. Use --tx flag to specify deposit transaction hash")
	}

	depositLogTopic := common.HexToHash("0xb3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32")

	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Crit("Error creating RPC client", "rpc", rpcURL, "err", err)
	}

	// Use proper context with timeout instead of context.TODO()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	l1Receipt, err := ethClient.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		log.Crit("Error fetching transaction receipt", "txHash", txHash, "err", err)
	}

	var found bool
	for _, ethLog := range l1Receipt.Logs {
		if ethLog.Topics[0].String() == depositLogTopic.String() {
			found = true

			reconstructedDep, err := derive.UnmarshalDepositLogEvent(ethLog)
			if err != nil {
				log.Crit("Failed to parse deposit event", "err", err)
			}
			tx := types.NewTx(reconstructedDep)
			fmt.Println("L2 Tx Hash", tx.Hash().String())
		}
	}

	if !found {
		log.Crit("No deposit event found in transaction", "txHash", txHash)
	}
}
