package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	client, err := ethclient.Dial("http://localhost:7545")
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	header, err := client.HeaderByNumber(ctx, nil)
	fmt.Println(header.Number)
}
