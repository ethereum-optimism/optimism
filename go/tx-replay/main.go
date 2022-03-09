package main

import (
	"context"
	"flag"
	"log"

	"cloud.google.com/go/pubsub"

	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/l2geth/rlp"
)

var (
	project      = flag.String("gcp-project", "", "Google project-name")
	topic        = flag.String("topic", "", "PUB/SUB topic name to subscribe to")
	sequencerURL = flag.String("sequencer-url", "http://0.0.0.0:8545", "sequencer URL to replay txs")
	subscription = flag.String("sub", "", "subscription ID")
)

func init() {
	flag.Parse()
}

func main() {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, *project)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	sub := client.Subscription(*subscription)

	eclient, err := ethclient.DialContext(ctx, *sequencerURL)
	if err != nil {
		log.Fatal(err)
	}

	cb := func(ctx context.Context, msg *pubsub.Message) {
		var tx types.Transaction
		if err := rlp.DecodeBytes(msg.Data, &tx); err != nil {
			log.Fatalf("invalid transaction in queue: %v", err)
		}
		jason, _ := tx.MarshalJSON()
		log.Printf("Replaying transaction: %s\n", string(jason))

		if err := eclient.SendTransaction(ctx, &tx); err != nil {
			log.Fatalf("Failed to resend transaction: %v", err)
		}
		msg.Ack()
	}
	for {
		err = sub.Receive(ctx, cb)
		if err != nil {
			log.Fatalf("unable to receive msg: %v", err)
		}
	}
}
