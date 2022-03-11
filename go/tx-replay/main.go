package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli"

	"cloud.google.com/go/pubsub"

	ethereum "github.com/ethereum-optimism/optimism/l2geth"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	"github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum-optimism/optimism/l2geth/log"
	"github.com/ethereum-optimism/optimism/l2geth/params"
	"github.com/ethereum-optimism/optimism/l2geth/rlp"
)

var (
	GitVersion = ""
	GitCommit  = ""
	GitDate    = ""

	SubscriptionReceiveSettings = pubsub.ReceiveSettings{
		MaxOutstandingMessages: 100000,
		MaxOutstandingBytes:    1e9,
	}
)

func main() {
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	app := cli.NewApp()
	app.Flags = flags
	app.Version = fmt.Sprintf("%s-%s", GitVersion, params.VersionWithCommit(GitCommit, GitDate))
	app.Name = "tx-replay"
	app.Usage = "Transaction replay"
	app.Description = "Replay transactions from Google PubSub to a sequencer"
	app.Action = Main()
	if err := app.Run(os.Args); err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func Main() func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		client, err := pubsub.NewClient(context.Background(), ctx.GlobalString(GcpProjectFlag.Name))
		if err != nil {
			log.Error("Failed to create pubsub client", "msg", err)
			return err
		}
		defer client.Close()

		sub := client.Subscription(ctx.GlobalString(SubscriptionIDFlag.Name))
		sub.ReceiveSettings = SubscriptionReceiveSettings

		eclient, err := ethclient.DialContext(context.Background(), ctx.GlobalString(SequencerURLFlag.Name))
		if err != nil {
			log.Error("Failed to create ethclient", "msg", err)
			return err
		}

		// sanity check sub configs
		/*
			subConfig, err := sub.Config(context.Background())
			if err != nil {
				log.Error("Failed to retrieve subscription config", "msg", err)
				return err
			}
			if !subConfig.EnableMessageOrdering {
				return errors.New("invalid sub config: message ordering is not enabled")
			}
		*/

		for {
			err = sub.Receive(context.Background(), func(ctx context.Context, msg *pubsub.Message) {
				handleMessage(ctx, msg, eclient)
			})
			if err != nil {
				log.Error("unable to receive message", "msg", err)
			}
		}
	}
}

func handleMessage(ctx context.Context, msg *pubsub.Message, eclient *ethclient.Client) {
	var tx types.Transaction
	if err := rlp.DecodeBytes(msg.Data, &tx); err != nil {
		msg.Nack()
		log.Error("invalid transaction in queue", "msg", err)
		// TODO: Backoff for a bit?
	}

	txHash := tx.Hash()
	_, _, err := eclient.TransactionByHash(ctx, txHash)
	if err != nil && err != ethereum.NotFound {
		log.Error("unable to retrieve transaction", "hash", txHash.String(), "msg", err)
		msg.Nack()
		return // retry later
	}
	if err == ethereum.NotFound {
		log.Info("Skipping transaction", "hash", txHash.String())
		msg.Ack()
		return
	}

	jason, _ := tx.MarshalJSON()
	log.Info("Replaying transaction", "hash", txHash.String(), "tx", string(jason))

	if err := eclient.SendTransaction(ctx, &tx); err != nil {
		msg.Nack()
		log.Error("Failed to replay transaction", "hash", txHash.String(), "msg", err)
		return
	}
	msg.Ack()

}
