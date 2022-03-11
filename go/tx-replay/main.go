package main

import (
	"context"
	"flag"
	"log"

	"cloud.google.com/go/pubsub"

	ethereum "github.com/ethereum-optimism/optimism/l2geth"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/l2geth/rlp"
)

var (
	project      = flag.String("gcp-project", "", "Google project-name")
	topic        = flag.String("topic", "", "PUB/SUB topic name to subscribe to")
	sequencerURL = flag.String("sequencer-url", "http://0.0.0.0:8545", "sequencer URL to replay txs")
	subscription = flag.String("sub", "", "subscription ID")

	SubscriptionReceiveSettings = pubsub.ReceiveSettings{
		MaxOutstandingMessages: 100000,
		MaxOutstandingBytes:    1e9,
	}
)

func init() {
	flag.Parse()
}

func main() {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, *project)
	if err != nil {
		log.Fatalf("Failed to create pubsub client: %v", err)
	}
	defer client.Close()

	sub := client.Subscription(*subscription)
	sub.ReceiveSettings = SubscriptionReceiveSettings

	eclient, err := ethclient.DialContext(ctx, *sequencerURL)
	if err != nil {
		log.Fatalf("Failed to create ethclient: %v", err)
	}

	// sanity check sub configs
	subConfig, err := sub.Config(ctx)
	if err != nil {
		log.Fatalf("Failed to retrieve subscription config: %v", err)
	}
	if !subConfig.EnableMessageOrdering {
		log.Fatal("invalid sub config: message ordering is not enabled")
	}

	cb := func(ctx context.Context, msg *pubsub.Message) {
		var tx types.Transaction
		if err := rlp.DecodeBytes(msg.Data, &tx); err != nil {
			msg.Nack()
			log.Fatalf("invalid transaction in queue: %v", err)
		}

		txHash := tx.Hash()
		rtx, _, err := eclient.TransactionByHash(ctx, txHash)
		if err != nil && err != ethereum.NotFound {
			log.Printf("ERROR: unable to retrieve transaction hash %s: %v", txHash.String(), err)
			msg.Nack()
			return // retry later
		}
		if rtx != nil || err == ethereum.NotFound {
			log.Printf("Skipping transaction hash %s", txHash.String())
			msg.Ack()
			return
		}

		jason, _ := tx.MarshalJSON()
		log.Printf("Replaying transaction %s: %s\n", txHash.String(), string(jason))

		if err := eclient.SendTransaction(ctx, &tx); err != nil {
			msg.Nack()
			log.Fatalf("Failed to replay transaction %s: %v", txHash.String(), err)
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
