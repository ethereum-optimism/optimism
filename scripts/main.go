package main

import (
	"context"
	"log"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
)

const (
	l1Url                    = "http://localhost:8545"
	l2Url                    = "http://localhost:9545"
	deployerAddr             = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	deployerPrivKey          = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	randomAddr               = "0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5"
	l1BlockL2ContractAddress = "0x4200000000000000000000000000000000000015"
)

func main() {
	app := cli.NewApp()
	app.Name = "Op devnet scripts"
	app.Description = "Collection of scripts to help analyse the op-stack devnet"

	app.Commands = []*cli.Command{
		{
			Name:  "l2-tx",
			Usage: "Sends a simple L2 Transaction",
			Action: func(_ *cli.Context) error {
				sendL2Tx()
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func sendL2Tx() {
	l2Client, err := ethclient.Dial(l2Url)
	if err != nil {
		log.Fatalf("Error dialing ethereum client: %v", err)
	}

	printDeployerBalance(l2Client)

	tx := signTransaction(randomAddr, "", deployerPrivKey, uint64(21400), uint64(1), l2Client)
	err = l2Client.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalln(err)
	}
	receipt := waitForTransaction(tx.Hash(), l2Client)
	log.Println("Tx block number: ", receipt.BlockNumber)
	printDeployerBalance(l2Client)
	l1Origin := getL1Origin(receipt.BlockHash, l2Client)
	log.Println("L1Origin: ", l1Origin)
}

func getL1Origin(l2BlockHash common.Hash, l2Client *ethclient.Client) uint64 {
	address := common.HexToAddress(l1BlockL2ContractAddress)
	instance, err := bindings.NewL1BlockCaller(address, l2Client)
	if err != nil {
		log.Fatal(err)
	}

	l1Origin, err := instance.Number(&bind.CallOpts{BlockHash: l2BlockHash})
	if err != nil {
		log.Fatal(err)
	}

	return l1Origin
}

func printDeployerBalance(client *ethclient.Client) {
	ctx := context.Background()
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("Block number: %v", blockNumber)

	balance, err := client.BalanceAt(ctx, common.HexToAddress(deployerAddr), big.NewInt(int64(blockNumber)))
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("Deployer balance: %v", ToDecimal(balance, 18))
}

// ToDecimal wei to decimals
func ToDecimal(ivalue interface{}, decimals int) decimal.Decimal {
	value := new(big.Int)
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, 10)
	case *big.Int:
		value = v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.Div(mul)

	return result
}
