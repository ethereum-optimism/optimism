package main

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// productionChains are the chain IDs the devnet helpers refuse to touch: the production
// interop L2s and Ethereum mainnet L1. Production pauses go through the guardian's
// DeputyPauseModule flow, never this tool.
var productionChains = map[uint64]string{
	1:        "Ethereum mainnet",
	10:       "OP Mainnet",
	57073:    "Ink",
	130:      "Unichain",
	11155420: "OP Sepolia",
	763373:   "Ink Sepolia",
	1301:     "Unichain Sepolia",
}

// refuseProduction returns an error naming every production chain among l1ChainID and
// l2ChainIDs.
func refuseProduction(l1ChainID uint64, l2ChainIDs []uint64) error {
	var hits []string
	for _, id := range append([]uint64{l1ChainID}, l2ChainIDs...) {
		if name, ok := productionChains[id]; ok {
			hits = append(hits, fmt.Sprintf("%s (%d)", name, id))
		}
	}
	if len(hits) == 0 {
		return nil
	}
	return fmt.Errorf("refusing to run a devnet helper against production chain(s) %s: "+
		"production pauses go through the guardian (DeputyPauseModule); use `sync` to mirror an existing L1 pause",
		strings.Join(hits, ", "))
}

// chainPair is one --l2 endpoint and its (optional) --l1-messenger, as given on the CLI.
type chainPair struct {
	L2        string
	Messenger common.Address
}

// pairChains matches --l2 endpoints with --l1-messenger addresses by position. Messengers are
// optional unless requireMessengers is set.
func pairChains(l2s, messengers []string, requireMessengers bool) ([]chainPair, error) {
	if len(l2s) == 0 {
		return nil, errors.New("at least one --l2 is required")
	}
	if len(messengers) == 0 && requireMessengers {
		return nil, errors.New("--l1-messenger is required, once per --l2 in the same order")
	}
	if len(messengers) != 0 && len(messengers) != len(l2s) {
		return nil, fmt.Errorf("got %d --l2 and %d --l1-messenger values; give one messenger per chain in the same order",
			len(l2s), len(messengers))
	}
	pairs := make([]chainPair, len(l2s))
	for i, l2 := range l2s {
		pairs[i].L2 = l2
		if len(messengers) == 0 {
			continue
		}
		if !common.IsHexAddress(messengers[i]) {
			return nil, fmt.Errorf("--l1-messenger %q is not an address", messengers[i])
		}
		pairs[i].Messenger = common.HexToAddress(messengers[i])
		if pairs[i].Messenger == (common.Address{}) {
			return nil, fmt.Errorf("--l1-messenger for --l2 %s is the zero address; give the chain's L1CrossDomainMessenger proxy", l2)
		}
	}
	return pairs, nil
}

// l1Backend is the L1 RPC surface the tool uses; *ethclient.Client implements it.
type l1Backend interface {
	ethereum.ChainIDReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer1559
	ethereum.PendingStateReader
	ethereum.ChainReader
	ethereum.TransactionReader
	ethereum.TransactionSender
}

// l2Reader is the L2 RPC surface the tool reads through; *ethclient.Client implements it.
type l2Reader interface {
	ethereum.ContractCaller
	ethereum.BlockNumberReader
}

// l2Client is a dialed L2 endpoint. Geth (state-override eth_call) is nil in tests that only
// read ETHLiquidity.
type l2Client struct {
	URL     string
	ChainID uint64
	Eth     l2Reader
	Geth    *gethclient.Client
	close   func()
}

func (c *l2Client) Close() {
	if c.close != nil {
		c.close()
	}
}

func dialL2(ctx context.Context, url string) (*l2Client, error) {
	rpcCl, err := gethrpc.DialContext(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("dial L2 %s: %w", url, err)
	}
	eth := ethclient.NewClient(rpcCl)
	cl := &l2Client{URL: url, Eth: eth, Geth: gethclient.New(rpcCl), close: eth.Close}
	id, err := eth.ChainID(ctx)
	if err != nil {
		cl.Close()
		return nil, fmt.Errorf("L2 %s chain ID: %w", url, err)
	}
	if !id.IsUint64() {
		cl.Close()
		return nil, fmt.Errorf("L2 %s chain ID %s does not fit in uint64", url, id)
	}
	cl.ChainID = bigs.Uint64Strict(id)
	return cl, nil
}

// chain is one L2 with its resolved L1 contracts (nil without --l1-messenger).
type chain struct {
	L2 *l2Client
	L1 *l1Contracts
}

// connection holds the dialed L1 (nil without --l1) and every chain.
type connection struct {
	L1        l1Backend
	L1ChainID uint64
	Chains    []chain
	closeL1   func()
}

func (c *connection) Close() {
	if c.closeL1 != nil {
		c.closeL1()
	}
	for _, ch := range c.Chains {
		ch.L2.Close()
	}
}

func (c *connection) l2ChainIDs() []uint64 {
	ids := make([]uint64, len(c.Chains))
	for i, ch := range c.Chains {
		ids[i] = ch.L2.ChainID
	}
	return ids
}

// connect dials every endpoint and, when withL1 is set (the user gave --l1-messenger), resolves
// each chain's L1 contracts, checking that each messenger's SystemConfig reports the chain ID of
// the L2 it is paired with.
func connect(ctx context.Context, l1URL string, pairs []chainPair, withL1 bool) (_ *connection, err error) {
	conn := &connection{}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	if !withL1 && l1URL != "" {
		return nil, errors.New("--l1 needs one --l1-messenger per --l2, in the same order; drop --l1 to read only L2")
	}
	if withL1 {
		if l1URL == "" {
			return nil, errors.New("--l1 is required with --l1-messenger")
		}
		l1, err := ethclient.DialContext(ctx, l1URL)
		if err != nil {
			return nil, fmt.Errorf("dial L1: %w", err)
		}
		conn.L1, conn.closeL1 = l1, l1.Close
		id, err := conn.L1.ChainID(ctx)
		if err != nil {
			return nil, fmt.Errorf("L1 chain ID: %w", err)
		}
		if !id.IsUint64() {
			return nil, fmt.Errorf("L1 chain ID %s does not fit in uint64", id)
		}
		conn.L1ChainID = bigs.Uint64Strict(id)
	}
	seen := make(map[uint64]string)
	for _, p := range pairs {
		l2, err := dialL2(ctx, p.L2)
		if err != nil {
			return nil, err
		}
		conn.Chains = append(conn.Chains, chain{L2: l2})
		if prev, ok := seen[l2.ChainID]; ok {
			return nil, fmt.Errorf("--l2 %s and %s are the same chain (%d)", prev, p.L2, l2.ChainID)
		}
		seen[l2.ChainID] = p.L2
		if !withL1 {
			continue
		}
		l1c, err := resolveL1(ctx, conn.L1, p.Messenger)
		if err != nil {
			return nil, fmt.Errorf("chain %d: %w", l2.ChainID, err)
		}
		if l1c.L2ChainID != l2.ChainID {
			return nil, fmt.Errorf("--l1-messenger %s belongs to chain %d but is paired with --l2 %s (chain %d)",
				p.Messenger, l1c.L2ChainID, p.L2, l2.ChainID)
		}
		conn.Chains[len(conn.Chains)-1].L1 = l1c
	}
	return conn, nil
}

func parseKey(flagName, keyHex string) (*ecdsa.PrivateKey, error) {
	if keyHex == "" {
		return nil, fmt.Errorf("--%s is required", flagName)
	}
	key, err := crypto.HexToECDSA(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse --%s: %w", flagName, err)
	}
	return key, nil
}

// pauseIdentifier is one SuperchainConfig pause scope: chains in an interop set share an
// ETHLockbox, so several chains can map to one identifier.
type pauseIdentifier struct {
	SuperchainConfig common.Address
	Lockbox          common.Address
	ChainIDs         []uint64
}

// pauseIdentifiers groups chains by (SuperchainConfig, ETHLockbox), in first-seen order.
func pauseIdentifiers(chains []chain) []pauseIdentifier {
	var out []pauseIdentifier
	index := make(map[[2]common.Address]int)
	for _, ch := range chains {
		k := [2]common.Address{ch.L1.SuperchainConfig, ch.L1.Lockbox}
		i, ok := index[k]
		if !ok {
			i = len(out)
			index[k] = i
			out = append(out, pauseIdentifier{SuperchainConfig: k[0], Lockbox: k[1]})
		}
		out[i].ChainIDs = append(out[i].ChainIDs, ch.L2.ChainID)
	}
	for i := range out {
		sort.Slice(out[i].ChainIDs, func(a, b int) bool { return out[i].ChainIDs[a] < out[i].ChainIDs[b] })
	}
	return out
}

// l2Paused reads ETHLiquidity.paused() on L2 at block (nil: latest).
func l2Paused(ctx context.Context, l2 *l2Client, block *big.Int) (bool, error) {
	var paused bool
	if err := callView(ctx, l2.Eth, predeploys.ETHLiquidityAddr, block, pausedFn, &paused); err != nil {
		return false, fmt.Errorf("ETHLiquidity.paused(): %w", err)
	}
	return paused, nil
}

// l2PausedMessageNonce reads ETHLiquidity.pausedMessageNonce() on L2 at block (nil: latest).
func l2PausedMessageNonce(ctx context.Context, l2 *l2Client, block *big.Int) (*big.Int, error) {
	nonce := new(big.Int)
	if err := callView(ctx, l2.Eth, predeploys.ETHLiquidityAddr, block, pausedMessageNonceFn, nonce); err != nil {
		return nil, fmt.Errorf("ETHLiquidity.pausedMessageNonce(): %w", err)
	}
	return nonce, nil
}

// fmtMessageNonce prints a versioned L1CrossDomainMessenger nonce as its plain nonce.
func fmtMessageNonce(versioned *big.Int) string {
	nonce, version := crossdomain.DecodeVersionedNonce(versioned)
	return fmt.Sprintf("L1 message nonce %s (version %s)", nonce, version)
}

// oneETH funds the status eth_call sender through a state override.
var oneETH = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
