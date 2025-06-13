package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	rpcURL      = flag.String("rpc", "", "RPC endpoint URL")
	mnemonic    = flag.String("mnemonic", "", "Mnemonic phrase to derive keys from")
	maxAccounts = flag.Int("max-accounts", 10, "Maximum number of accounts to check")
	minBalance  = flag.String("min-balance", "0.1", "Minimum balance in ETH to trigger sweep")
	gasReserve  = flag.String("gas-reserve", "0.01", "ETH to leave for gas fees")
	refundAddr  = flag.String("refund-to", "", "Address to send swept funds to (if empty, uses first derived address)")
	dryRun      = flag.Bool("dry-run", false, "Only show what would be done without executing")
	chainID     = flag.Int64("chain-id", 0, "Chain ID for transactions")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	if *rpcURL == "" {
		log.Fatal("RPC URL is required")
	}
	if *mnemonic == "" {
		log.Fatal("Mnemonic is required")
	}
	if *chainID == 0 {
		log.Fatal("Chain ID is required")
	}

	ctx := context.Background()
	client, err := ethclient.Dial(*rpcURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	defer client.Close()

	minBalanceEth, ok := new(big.Float).SetString(*minBalance)
	if !ok {
		log.Fatalf("Invalid min-balance: %s", *minBalance)
	}
	minBalanceWei, _ := new(big.Float).Mul(minBalanceEth, big.NewFloat(1e18)).Int(nil)

	gasReserveEth, ok := new(big.Float).SetString(*gasReserve)
	if !ok {
		log.Fatalf("Invalid gas-reserve: %s", *gasReserve)
	}
	gasReserveWei, _ := new(big.Float).Mul(gasReserveEth, big.NewFloat(1e18)).Int(nil)

	var refundAddress common.Address
	if *refundAddr != "" {
		if !common.IsHexAddress(*refundAddr) {
			log.Fatalf("Invalid refund address: %s", *refundAddr)
		}
		refundAddress = common.HexToAddress(*refundAddr)
	}

	swept := sweepFunds(ctx, client, *mnemonic, *maxAccounts, minBalanceWei, gasReserveWei, refundAddress, *dryRun, *chainID)

	if *dryRun {
		fmt.Printf("DRY RUN: Would have swept %d accounts\n", swept)
	} else {
		fmt.Printf("Successfully swept %d accounts\n", swept)
	}
}

func sweepFunds(ctx context.Context, client *ethclient.Client, mnemonic string, maxAccounts int, minBalance, gasReserve *big.Int, refundAddr common.Address, dryRun bool, chainID int64) int {
	swept := 0

	// If no refund address specified, use the first derived address as the collector
	if refundAddr == (common.Address{}) {
		key, err := deriveKey(mnemonic, 0)
		if err != nil {
			log.Fatalf("Failed to derive first key: %v", err)
		}
		refundAddr = crypto.PubkeyToAddress(key.PublicKey)
		if *verbose {
			log.Printf("Using first derived address as refund target: %s", refundAddr.Hex())
		}
	}

	for i := 0; i < maxAccounts; i++ {
		key, err := deriveKey(mnemonic, uint32(i))
		if err != nil {
			log.Printf("Failed to derive key for account %d: %v", i, err)
			continue
		}

		addr := crypto.PubkeyToAddress(key.PublicKey)

		// Skip if this is the refund address to avoid self-transfer
		if addr == refundAddr {
			if *verbose {
				log.Printf("Skipping refund address: %s", addr.Hex())
			}
			continue
		}

		balance, err := client.BalanceAt(ctx, addr, nil)
		if err != nil {
			log.Printf("Failed to get balance for %s: %v", addr.Hex(), err)
			continue
		}

		if balance.Cmp(minBalance) < 0 {
			if *verbose {
				log.Printf("Account %s has insufficient balance: %s ETH", addr.Hex(), weiToEth(balance))
			}
			continue
		}

		// Calculate sweep amount (balance - gas reserve)
		sweepAmount := new(big.Int).Sub(balance, gasReserve)
		if sweepAmount.Cmp(big.NewInt(0)) <= 0 {
			if *verbose {
				log.Printf("Account %s: sweep amount too low after gas reserve", addr.Hex())
			}
			continue
		}

		if dryRun {
			fmt.Printf("DRY RUN: Would sweep %s ETH from %s to %s\n",
				weiToEth(sweepAmount), addr.Hex(), refundAddr.Hex())
			swept++
			continue
		}

		// Perform the actual sweep
		if err := performSweep(ctx, client, key, addr, refundAddr, sweepAmount, chainID); err != nil {
			log.Printf("Failed to sweep from %s: %v", addr.Hex(), err)
			continue
		}

		fmt.Printf("Swept %s ETH from %s to %s\n",
			weiToEth(sweepAmount), addr.Hex(), refundAddr.Hex())
		swept++
	}

	return swept
}

func performSweep(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, from, to common.Address, amount *big.Int, chainID int64) error {
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := types.NewTransaction(nonce, to, amount, 21000, gasPrice, nil)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), key)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	if *verbose {
		log.Printf("Transaction sent: %s", signedTx.Hash().Hex())
	}

	return nil
}

func deriveKey(mnemonic string, index uint32) (*ecdsa.PrivateKey, error) {
	// This is a simplified key derivation
	// In a real implementation, you would use proper BIP39/BIP44 derivation
	words := strings.Fields(mnemonic)
	if len(words) != 12 && len(words) != 24 {
		return nil, fmt.Errorf("invalid mnemonic length")
	}

	// Create a deterministic but unique seed per index
	seed := fmt.Sprintf("%s:%d", mnemonic, index)
	hash := crypto.Keccak256([]byte(seed))

	return crypto.ToECDSA(hash)
}

func weiToEth(wei *big.Int) string {
	eth := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(1e18))
	return eth.Text('f', 6)
}
