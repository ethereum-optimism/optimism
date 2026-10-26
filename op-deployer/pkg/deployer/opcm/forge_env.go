package opcm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/google/uuid"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

// ForgeEnv runs the deploy scripts with forge against a live L1.
type ForgeEnv struct {
	Client     *forge.Client
	Context    context.Context
	L1RPCUrl   string
	PrivateKey string
}

type newScriptCaller[I, O any] func(*forge.Client) forge.ScriptCaller[I, O]

// broadcastScript runs a script that sends transactions to the L1.
func broadcastScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, true)
}

// readScript runs a script that only reads.
func readScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I) (O, error) {
	return runScript(env, newCaller, input, false)
}

func runScript[I, O any](env *ForgeEnv, newCaller newScriptCaller[I, O], input I, broadcast bool) (O, error) {
	var out O
	if env.Client == nil {
		return out, errors.New("forge client is nil")
	}
	ctx := env.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if env.L1RPCUrl == "" {
		return out, errors.New("L1 RPC URL is required for Forge deployments")
	}
	flags := []string{"--rpc-url", env.L1RPCUrl}
	switch {
	case !broadcast:
	case env.PrivateKey != "":
		keystore, err := newForgeKeystore(env.PrivateKey)
		if err != nil {
			return out, err
		}
		defer keystore.close()
		tip, feeCap, err := broadcastFees(ctx, env.L1RPCUrl)
		if err != nil {
			return out, err
		}
		flags = append(flags, "--broadcast", "--keystore", keystore.path, "--password-file", keystore.passwordPath,
			"--priority-gas-price", tip.String(), "--with-gas-price", feeCap.String())
	default:
		return out, errors.New("private key is required for Forge deployments")
	}
	out, err := newCaller(env.Client)(ctx, input, flags...)
	return out, err
}

// minPriorityFee is the tip floor, as txmgr's MinTipCap: forge's tip percentile is a few wei on an
// L1 with empty blocks, and geth does not include such a transaction.
var minPriorityFee = big.NewInt(params.GWei)

// minBaseFee is the base fee floor of the fee cap, as txmgr's MinBaseFee: forge does not bump a
// stuck transaction, so the cap keeps room for a rise.
var minBaseFee = big.NewInt(params.GWei)

// broadcastFees returns the priority fee, at least minPriorityFee, and the fee cap, twice the base
// fee (at least minBaseFee) plus the priority fee, as txmgr's calcGasFeeCap. forge's own fee cap
// can be below minPriorityFee.
func broadcastFees(ctx context.Context, rpcURL string) (*big.Int, *big.Int, error) {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to the L1: %w", err)
	}
	defer client.Close()
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the suggested priority fee: %w", err)
	}
	if tip.Cmp(minPriorityFee) < 0 {
		tip = new(big.Int).Set(minPriorityFee)
	}
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the latest L1 block: %w", err)
	}
	if head.BaseFee == nil {
		return nil, nil, errors.New("the latest L1 block has no base fee")
	}
	baseFee := head.BaseFee
	if baseFee.Cmp(minBaseFee) < 0 {
		baseFee = minBaseFee
	}
	feeCap := new(big.Int).Add(tip, new(big.Int).Mul(baseFee, big.NewInt(2)))
	return tip, feeCap, nil
}

// forgeKeystore holds the deployer key for one run, in a keystore with a random password in a
// directory only the user can read, so the key is never on forge's command line, which other users
// of the host can read.
type forgeKeystore struct {
	dir          string
	path         string
	passwordPath string
}

func newForgeKeystore(privateKey string) (*forgeKeystore, error) {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("failed to generate the keystore password: %w", err)
	}
	password := hex.EncodeToString(passwordBytes)
	keyJSON, err := keystore.EncryptKey(&keystore.Key{
		Id:         uuid.New(),
		Address:    crypto.PubkeyToAddress(key.PublicKey),
		PrivateKey: key,
	}, password, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt the deployer key: %w", err)
	}

	// MkdirTemp makes the directory readable only by the user.
	dir, err := os.MkdirTemp("", "op-deployer-key-")
	if err != nil {
		return nil, fmt.Errorf("failed to create the keystore directory: %w", err)
	}
	ks := &forgeKeystore{
		dir:          dir,
		path:         filepath.Join(dir, "key.json"),
		passwordPath: filepath.Join(dir, "password"),
	}
	if err := os.WriteFile(ks.path, keyJSON, 0o600); err != nil {
		ks.close()
		return nil, fmt.Errorf("failed to write the keystore: %w", err)
	}
	if err := os.WriteFile(ks.passwordPath, []byte(password), 0o600); err != nil {
		ks.close()
		return nil, fmt.Errorf("failed to write the keystore password: %w", err)
	}
	return ks, nil
}

func (ks *forgeKeystore) close() {
	_ = os.RemoveAll(ks.dir)
}
