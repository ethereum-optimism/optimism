package main

import (
	"context"
	"errors"
	"math/big"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/l2geth/crypto"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

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

			l2RpcClient, err := rpc.DialContext(context.Background(), l2RpcURL)
			if err != nil {
				return err
			}
			gclient := gethclient.New(l2RpcClient)
			log.Info("Set up L2 geth Client")

			ovmMessages, err := migration.NewSentMessage(ctx.String("ovm-messages"))
			if err != nil {
				return err
			}
			evmMessages, err := migration.NewSentMessage(ctx.String("evm-messages"))
			if err != nil {
				return err
			}

			optimismPortalAddress := ctx.String("optimism-portal-address")
			if optimismPortalAddress == "" {
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

			l1xdmAddress := ctx.String("l1-crossdomain-messenger-address")
			if l1xdmAddress == "" {
				return errors.New("Must pass in --l1-crossdomain-messenger-address")
			}
			l1xdmAddr := common.HexToAddress(l1xdmAddress)

			// TODO: temp, should iterate over all instead of taking the first
			wd := wds[0]
			withdrawal, err := crossdomain.MigrateWithdrawal(wd, &l1xdmAddr)
			if err != nil {
				return err
			}

			hash, err := withdrawal.Hash()
			if err != nil {
				return err
			}

			slot, err := withdrawal.StorageSlot()
			if err != nil {
				return err
			}

			transitionBlockNumber := new(big.Int).SetUint64(ctx.Uint64("bedrock-transition-block-number"))
			bn, err := withdrawals.WaitForFinalizationPeriod(context.Background(), l1Client, optimismPortalAddr, transitionBlockNumber)
			if err != nil {
				return err
			}
			header, err := l2Client.HeaderByNumber(context.Background(), new(big.Int).SetUint64(bn))
			if err != nil {
				return err
			}
			proof, err := gclient.GetProof(context.Background(), predeploys.L2ToL1MessagePasserAddr, []string{slot.String()}, header.Number)
			if err != nil {
				return err
			}
			if len(proof.StorageProof) != 1 {
				return errors.New("invalid amount of storage proofs")
			}
			trieNodes := make([][]byte, len(proof.StorageProof[0].Proof))
			for i, s := range proof.StorageProof[0].Proof {
				trieNodes[i] = common.FromHex(s)
			}

			portal, err := bindings.NewOptimismPortal(optimismPortalAddr, l1Client)
			if err != nil {
				return err
			}

			l2OracleAddr, err := portal.L2ORACLE(&bind.CallOpts{})
			oracle, err := bindings.NewL2OutputOracleCaller(l2OracleAddr, l1Client)
			if err != nil {
				return nil
			}

			l2OutputIndex, err := oracle.GetL2OutputIndexAfter(&bind.CallOpts{}, header.Number)

			if ctx.String("private-key") == "" {
				return errors.New("No private key to transact with")
			}
			privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(ctx.String("private-key"), "0x"))
			if err != nil {
				return err
			}

			opts, err := bind.NewKeyedTransactorWithChainID(privateKey, l1ChainID)
			if err != nil {
				return err
			}

			tx, err := portal.ProveWithdrawalTransaction(
				opts,
				bindings.TypesWithdrawalTransaction{
					Nonce:    withdrawal.Nonce,
					Sender:   *withdrawal.Sender,
					Target:   *withdrawal.Target,
					Value:    withdrawal.Value,
					GasLimit: withdrawal.GasLimit,
					Data:     withdrawal.Data,
				},
				l2OutputIndex,
				bindings.TypesOutputRootProof{
					Version:                  [32]byte{},
					StateRoot:                header.Root,
					MessagePasserStorageRoot: proof.StorageHash,
					LatestBlockhash:          header.Hash(),
				},
				trieNodes,
			)

			if err != nil {
				return err
			}

			log.Info("Withdrawal proven", "tx-hash", tx.Hash(), "withdrawal-hash", hash)

			// TODO: - warp forward the L1 timestamp
			//       - finalize the tx

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error in migration", "err", err)
	}
}
