package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := cli.NewApp()
	app.Name = "check-derivation"
	app.Usage = "Optimism derivation checker"
	app.Commands = []*cli.Command{
		{
			Name:  "detect-l2-reorg",
			Usage: "Detects unsafe block reorg",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "l2-rpc-url",
					Value:   "http://127.0.0.1:9545",
					Usage:   "L2 RPC URL",
					EnvVars: []string{"L2_RPC_URL"},
				},
				&cli.IntFlag{
					Name:  "polling-interval",
					Value: 500,
					Usage: "Polling interval (ms)",
				},
			},
			Action: detectL2Reorg,
		},
		{
			Name:  "check-consolidation",
			Usage: "Checks consolidation",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "l2-rpc-url",
					Value:   "http://127.0.0.1:9545",
					Usage:   "L2 RPC URL",
					EnvVars: []string{"L2_RPC_URL"},
				},
				&cli.IntFlag{
					Name:  "polling-interval",
					Value: 1000,
					Usage: "Polling interval (ms)",
				},
				&cli.StringFlag{
					Name:  "private-key",
					Value: "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
					Usage: "Private key for signing L2 transactions. " +
						"Default: devnet pre-funded account",
				},
				&cli.IntFlag{
					Name:  "tx-count",
					Value: 4,
					Usage: "Number of transactions to send. Minimum value is 4 for checking every tx type.",
				},
				&cli.Uint64Flag{
					Name:  "l2-chain-id",
					Value: 901,
					Usage: "L2 chain ID",
				},
			},
			Action: checkConsolidation,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error checking l2", "err", err)
	}
}

type clients struct {
	Client    *ethclient.Client
	RpcClient *rpc.Client
}

func newClientsFromContext(ctx context.Context, cliCtx *cli.Context) (*clients, error) {
	url := cliCtx.String("l2-rpc-url")
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("cannot dial ethclient: %w", err)
	}
	rpcClient, err := rpc.DialContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("cannot dial rpc client: %w", err)
	}
	return &clients{Client: ethClient, RpcClient: rpcClient}, nil
}

func getHead(ctx context.Context, client *rpc.Client, info interface{}) (eth.BlockID, common.Hash, error) {
	return retry.Do2(ctx, 10, &retry.FixedStrategy{Dur: 100 * time.Millisecond}, func() (eth.BlockID, common.Hash, error) {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		res := make(map[string]interface{})
		err := client.CallContext(ctx, &res, "eth_getBlockByNumber", info, false)
		if err != nil {
			return eth.BlockID{}, common.Hash{}, err
		}
		numStr, ok := res["number"].(string)
		if !ok {
			return eth.BlockID{}, common.Hash{}, errors.New("block number field invalid")
		}
		blockNum, err := hexutil.DecodeUint64(numStr)
		if err != nil {
			return eth.BlockID{}, common.Hash{}, fmt.Errorf("failed to decode RPC block number: %w", err)
		}
		hashStr, ok := res["hash"].(string)
		if !ok {
			return eth.BlockID{}, common.Hash{}, errors.New("hash field invalid")
		}
		hash := common.HexToHash(hashStr)
		parentHashStr, ok := res["parentHash"].(string)
		if !ok {
			return eth.BlockID{}, common.Hash{}, errors.New("parent hash field invalid")
		}
		parentHash := common.HexToHash(parentHashStr)
		return eth.BlockID{Hash: hash, Number: blockNum}, parentHash, nil
	})
}

func getUnsafeHead(ctx context.Context, client *rpc.Client) (eth.BlockID, common.Hash, error) {
	return getHead(ctx, client, "latest")
}

func getSafeHead(ctx context.Context, client *rpc.Client) (eth.BlockID, common.Hash, error) {
	return getHead(ctx, client, "safe")
}

func checkReorg(blockMap map[uint64]common.Hash, number uint64, hash common.Hash) {
	prevHash, ok := blockMap[number]
	if ok {
		if prevHash != hash {
			log.Error("Unsafe head reorg", "blockNum:", number,
				"prevHash", prevHash.String(), "currHash", hash.String())
		}
	}
}

// detectL2Reorg polls safe heads and detects l2 unsafe block reorg.
func detectL2Reorg(cliCtx *cli.Context) error {
	ctx := context.Background()
	clients, err := newClientsFromContext(ctx, cliCtx)
	if err != nil {
		return err
	}
	var pollingInterval = time.Duration(cliCtx.Int("polling-interval")) * time.Millisecond
	// blockMap maps blockNumber to blockHash
	blockMap := make(map[uint64]common.Hash)
	var prevUnsafeHeadNum uint64
	for {
		unsafeHeadBlockId, parentHash, err := getUnsafeHead(ctx, clients.RpcClient)
		if err != nil {
			return fmt.Errorf("failed to fetch unsafe head: %w", err)
		}
		checkReorg(blockMap, unsafeHeadBlockId.Number-1, parentHash)
		checkReorg(blockMap, unsafeHeadBlockId.Number, unsafeHeadBlockId.Hash)

		if unsafeHeadBlockId.Number > prevUnsafeHeadNum {
			log.Info("Fetched Unsafe block", "blockNum", unsafeHeadBlockId.Number, "hash", unsafeHeadBlockId.Hash.String())
		}

		blockMap[unsafeHeadBlockId.Number-1] = parentHash
		blockMap[unsafeHeadBlockId.Number] = unsafeHeadBlockId.Hash
		prevUnsafeHeadNum = unsafeHeadBlockId.Number
		time.Sleep(pollingInterval)
	}
}

// getRandomAddress returns vanity address of the form 0x000000000000000000000000[random 32 bits][prefix]
// example: 0x00000000000000000000000030bd3402deadbeef
func getRandomAddress(rng *rand.Rand, prefix uint64) common.Address {
	var vanity uint64 = prefix + (uint64(rng.Uint32()) << 32)
	return common.HexToAddress(fmt.Sprintf("0x%X", vanity))
}

func getPrivateKey(cliCtx *cli.Context) (*ecdsa.PrivateKey, error) {
	privateKey, err := crypto.HexToECDSA(cliCtx.String("private-key"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	return privateKey, nil
}

func getSenderAddress(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}
	from := crypto.PubkeyToAddress(*publicKeyECDSA)
	return from, nil
}

// getRandomSignedTransaction returns signed tx which sends 1 wei to random address
func getRandomSignedTransaction(ctx context.Context, ethClient *ethclient.Client, rng *rand.Rand, from common.Address, privateKey *ecdsa.PrivateKey, chainId *big.Int, txType int, protected bool) (*types.Transaction, error) {
	randomAddress := getRandomAddress(rng, 0xDEADBEEF)
	amount := big.NewInt(1)
	nonce, err := ethClient.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest gas price: %w", err)
	}
	data := testutils.RandomData(rng, 10)
	var txData types.TxData
	switch txType {
	case types.LegacyTxType:
		gasLimit, err := core.IntrinsicGas(data, nil, false, true, true, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get intrinsicGas: %w", err)
		}
		txData = &types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       &randomAddress,
			Value:    amount,
			Data:     data,
		}
	case types.AccessListTxType:
		accessList := types.AccessList{types.AccessTuple{
			Address:     randomAddress,
			StorageKeys: []common.Hash{common.HexToHash("0x1234")},
		}}
		gasLimit, err := core.IntrinsicGas(data, accessList, false, true, true, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get intrinsicGas: %w", err)
		}
		txData = &types.AccessListTx{
			ChainID:    chainId,
			Nonce:      nonce,
			GasPrice:   gasPrice,
			Gas:        gasLimit,
			To:         &randomAddress,
			Value:      amount,
			AccessList: accessList,
			Data:       data,
		}
	case types.DynamicFeeTxType:
		gasLimit, err := core.IntrinsicGas(data, nil, false, true, true, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get intrinsicGas: %w", err)
		}
		gasTipCap, err := ethClient.SuggestGasTipCap(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get gas tip cap: %w", err)
		}
		txData = &types.DynamicFeeTx{
			ChainID:   chainId,
			Nonce:     nonce,
			GasTipCap: gasTipCap,
			GasFeeCap: gasPrice,
			Gas:       gasLimit,
			To:        &randomAddress,
			Value:     amount,
			Data:      data,
		}
	default:
		return nil, fmt.Errorf("unsupported tx type: %d", txType)
	}

	tx := types.NewTx(txData)

	signer := types.NewLondonSigner(chainId)
	if !protected {
		if txType == types.LegacyTxType {
			signer = types.HomesteadSigner{}
		} else {
			return nil, errors.New("typed tx cannot be unprotected")
		}
	}

	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return signedTx, nil
}

// confirmTransaction polls receipts to confirm transaction is included in the block.
func confirmTransaction(ctx context.Context, ethClient *ethclient.Client, l2BlockTime uint64, txHash common.Hash) (eth.BlockID, error) {
	var retryCount uint64
	for {
		receipt, _ := ethClient.TransactionReceipt(ctx, txHash)
		if retryCount > 30 {
			return eth.BlockID{}, fmt.Errorf("transaction confirmation failure: txHash: %s", txHash.String())
		}
		if receipt == nil {
			log.Info("Waiting for transaction receipt", "txHash", txHash.String())
			retryCount++
			// wait at least l2 block time
			time.Sleep(time.Duration(l2BlockTime) * time.Second)
			continue
		}
		block := eth.BlockID{
			Hash:   receipt.BlockHash,
			Number: receipt.BlockNumber.Uint64(),
		}
		log.Info("Transaction receipt found", "block", block, "status", receipt.Status)
		return block, nil
	}
}

// checkConsolidation sends transactions and ensures them to be included in unsafe block.
// Then polls safe head to check unsafe blocks which includes sent tx are consolidated.
func checkConsolidation(cliCtx *cli.Context) error {
	ctx := context.Background()
	clients, err := newClientsFromContext(ctx, cliCtx)
	if err != nil {
		return err
	}
	var pollingInterval = time.Duration(cliCtx.Int("polling-interval")) * time.Millisecond
	privateKey, err := getPrivateKey(cliCtx)
	if err != nil {
		return err
	}
	from, err := getSenderAddress(privateKey)
	if err != nil {
		return err
	}
	txCount := cliCtx.Int("tx-count")
	if txCount < 4 {
		return fmt.Errorf("tx count %d is too low. requires minimum 4 txs to test all tx types", txCount)
	}
	l2ChainID := new(big.Int).SetUint64(cliCtx.Uint64("l2-chain-id"))
	l2BlockTime := uint64(2)
	rollupCfg, err := rollup.LoadOPStackRollupConfig(l2ChainID.Uint64())
	if err == nil {
		l2BlockTime = rollupCfg.BlockTime
	} else {
		log.Warn("Superchain config not loaded", "l2-chain-id", l2ChainID)
		log.Warn("Using default config", "l2-block-time", l2BlockTime)
	}
	rng := rand.New(rand.NewSource(1337))
	// txMap maps txHash to blockID
	txMap := make(map[common.Hash]eth.BlockID)
	for i := 0; i < txCount; i++ {
		var tx *types.Transaction
		switch i % 4 {
		case 0:
			tx, err = getRandomSignedTransaction(ctx, clients.Client, rng, from, privateKey, l2ChainID, types.LegacyTxType, false)
		case 1:
			tx, err = getRandomSignedTransaction(ctx, clients.Client, rng, from, privateKey, l2ChainID, types.LegacyTxType, true)
		case 2:
			tx, err = getRandomSignedTransaction(ctx, clients.Client, rng, from, privateKey, l2ChainID, types.AccessListTxType, true)
		case 3:
			tx, err = getRandomSignedTransaction(ctx, clients.Client, rng, from, privateKey, l2ChainID, types.DynamicFeeTxType, true)
		}
		if err != nil {
			return err
		}
		err = clients.Client.SendTransaction(ctx, tx)
		if err != nil {
			return fmt.Errorf("failed to send transaction: %w", err)
		}
		txHash := tx.Hash()
		blockId, err := confirmTransaction(ctx, clients.Client, l2BlockTime, txHash)
		if err != nil {
			return err
		}
		txMap[txHash] = blockId
	}
	lastSafeHeadNum := uint64(0)
	numChecked := 0
	failed := false
	for {
		safeHeadBlockId, _, err := getSafeHead(ctx, clients.RpcClient)
		if err != nil {
			return fmt.Errorf("failed to fetch safe head: %w", err)
		}
		log.Info("Fetched Safe head", "blockNum", safeHeadBlockId.Number, "hash", safeHeadBlockId.Hash.String(), "remainingTxCount", txCount-numChecked)

		for txHash, blockId := range txMap {
			if lastSafeHeadNum < blockId.Number && safeHeadBlockId.Number >= blockId.Number {
				safeBlockHash := safeHeadBlockId.Hash
				if safeHeadBlockId.Number != blockId.Number {
					safeBlock, err := retry.Do(ctx, 10, &retry.FixedStrategy{Dur: 100 * time.Millisecond}, func() (*types.Block, error) {
						return clients.Client.BlockByNumber(ctx, new(big.Int).SetUint64(blockId.Number))
					})
					if err != nil {
						return fmt.Errorf("failed to fetch block by number: %w", err)
					}
					safeBlockHash = safeBlock.Hash()
				}
				if safeBlockHash == blockId.Hash {
					log.Info("Transaction included at safe block", "block", blockId, "txHash", txHash.String())
				} else {
					log.Error("Transaction included block is reorged", "blockNum", blockId.Number, "prevHash", blockId.Hash, "currBlock", safeBlockHash, "txHash", txHash.String())
					failed = true
				}
				numChecked++
			}
		}
		if numChecked == txCount {
			if failed {
				log.Error("Failed")
			} else {
				log.Info("Succeeded")
			}
			break
		}
		lastSafeHeadNum = safeHeadBlockId.Number
		time.Sleep(pollingInterval)
	}
	return nil
}
