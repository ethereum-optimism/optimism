package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

const EnvPrefix = "OP_CHAIN_OPS_REGISTRY_DATA"

var (
	L2GenesisFlag = &cli.PathFlag{
		Name:    "l2-genesis",
		Value:   "genesis.json",
		Usage:   "Path to genesis json (go-ethereum format)",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "L2_GENESIS"),
	}
	L2GenesisHeaderFlag = &cli.PathFlag{
		Name:    "l2-genesis-header",
		Value:   "genesis-header.json",
		Usage:   "Alternative to l2-genesis flag, if genesis-state is omitted. Path to block header at genesis",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "L2_GENESIS_HEADER"),
	}
	BytecodesDirFlag = &cli.PathFlag{
		Name:    "bytecodes-dir",
		Value:   "superchain-registry/superchain/bytecodes",
		Usage:   "Collection of gzipped L2 bytecodes",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "BYTECODES_DIR"),
	}
	OutputFlag = &cli.PathFlag{
		Name:    "output",
		Value:   "out.json.gz",
		Usage:   "output gzipped JSON file path to write to",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "OUTPUT"),
	}
)

func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(isatty.IsTerminal(os.Stderr.Fd()))))

	app := &cli.App{
		Name:  "registry-data",
		Usage: "Prepare superchain-registry genesis data files based on full genesis dump",
		Flags: []cli.Flag{
			L2GenesisFlag,
			L2GenesisHeaderFlag,
			BytecodesDirFlag,
			OutputFlag,
		},
		Action: entrypoint,
	}
	app.Commands = []*cli.Command{
		{
			Name:  "bytecode",
			Usage: "Generate a single gzipped data file from a bytecode hex string",
			Flags: []cli.Flag{
				BytecodesDirFlag,
			},
			Action: bytecode,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error while generating registry data", "err", err)
	}
}

type GenesisAccount struct {
	CodeHash common.Hash                                          `json:"codeHash,omitempty"`
	Storage  jsonutil.LazySortedJsonMap[common.Hash, common.Hash] `json:"storage,omitempty"`
	Balance  *hexutil.Big                                         `json:"balance,omitempty"`
	Nonce    uint64                                               `json:"nonce,omitempty"`
}

type Genesis struct {
	Nonce      uint64         `json:"nonce"`
	Timestamp  uint64         `json:"timestamp"`
	ExtraData  []byte         `json:"extraData"`
	GasLimit   uint64         `json:"gasLimit"`
	Difficulty *hexutil.Big   `json:"difficulty"`
	Mixhash    common.Hash    `json:"mixHash"`
	Coinbase   common.Address `json:"coinbase"`
	Number     uint64         `json:"number"`
	GasUsed    uint64         `json:"gasUsed"`
	ParentHash common.Hash    `json:"parentHash"`
	BaseFee    *hexutil.Big   `json:"baseFeePerGas"`

	Alloc jsonutil.LazySortedJsonMap[common.Address, GenesisAccount] `json:"alloc"`
	// For genesis definitions without full state (OP-Mainnet, OP-Goerli)
	StateHash *common.Hash `json:"stateHash,omitempty"`
}

func loadJSON[X any](inputPath string) (*X, error) {
	if inputPath == "" {
		return nil, errors.New("no path specified")
	}
	f, err := os.OpenFile(inputPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", inputPath, err)
	}
	defer f.Close()
	var obj X
	if err := json.NewDecoder(f).Decode(&obj); err != nil {
		return nil, fmt.Errorf("failed to decode file %q: %w", inputPath, err)
	}
	return &obj, nil
}

func writeGzipJSON(outputPath string, value any) error {
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer f.Close()
	w, err := gzip.NewWriterLevel(f, flate.BestCompression)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer w.Close()
	enc := json.NewEncoder(w)
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("failed to encode to JSON: %w", err)
	}
	return nil
}

func entrypoint(ctx *cli.Context) error {
	genesisPath := ctx.Path(L2GenesisFlag.Name)
	if genesisPath == "" {
		// When the genesis-state is too large, or not meant to be available, only the header data is made available.
		// This allows the user to verify the header-chain starting from genesis, and state-sync the latest state,
		// skipping the historical state.
		// Archive nodes that depend on this historical state should instantiate the chain from a full genesis dump
		// with allocation data, or datadir.
		genesisHeaderPath := ctx.Path(L2GenesisHeaderFlag.Name)
		genesisHeader, err := loadJSON[types.Header](genesisHeaderPath)
		if err != nil {
			return fmt.Errorf("genesis-header %q failed to load: %w", genesisHeaderPath, err)
		}
		if genesisHeader.TxHash != types.EmptyTxsHash {
			return errors.New("genesis-header based genesis must have no transactions")
		}
		if genesisHeader.ReceiptHash != types.EmptyReceiptsHash {
			return errors.New("genesis-header based genesis must have no receipts")
		}
		if genesisHeader.UncleHash != types.EmptyUncleHash {
			return errors.New("genesis-header based genesis must have no uncle hashes")
		}
		if genesisHeader.WithdrawalsHash != nil && *genesisHeader.WithdrawalsHash != types.EmptyWithdrawalsHash {
			return errors.New("genesis-header based genesis must have no withdrawals")
		}
		out := Genesis{
			Nonce:      genesisHeader.Nonce.Uint64(),
			Timestamp:  genesisHeader.Time,
			ExtraData:  genesisHeader.Extra,
			GasLimit:   genesisHeader.GasLimit,
			Difficulty: (*hexutil.Big)(genesisHeader.Difficulty),
			Mixhash:    genesisHeader.MixDigest,
			Coinbase:   genesisHeader.Coinbase,
			Number:     genesisHeader.Number.Uint64(),
			GasUsed:    genesisHeader.GasUsed,
			ParentHash: genesisHeader.ParentHash,
			BaseFee:    (*hexutil.Big)(genesisHeader.BaseFee),
			Alloc:      make(jsonutil.LazySortedJsonMap[common.Address, GenesisAccount]),
			StateHash:  &genesisHeader.Root,
		}
		if err := writeGzipJSON(ctx.Path(OutputFlag.Name), out); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	genesis, err := loadJSON[core.Genesis](genesisPath)
	if err != nil {
		return fmt.Errorf("failed to load L2 genesis: %w", err)
	}

	// export all contract bytecodes, write them to bytecodes collection
	bytecodesDir := ctx.Path(BytecodesDirFlag.Name)
	if err := os.MkdirAll(bytecodesDir, 0755); err != nil {
		return fmt.Errorf("failed to make bytecodes dir: %w", err)
	}
	for addr, account := range genesis.Alloc {
		if len(account.Code) > 0 {
			err = writeBytecode(bytecodesDir, account.Code, addr)
			if err != nil {
				return err
			}
		}
	}

	// convert into allocation data
	out := Genesis{
		Nonce:      genesis.Nonce,
		Timestamp:  genesis.Timestamp,
		ExtraData:  genesis.ExtraData,
		GasLimit:   genesis.GasLimit,
		Difficulty: (*hexutil.Big)(genesis.Difficulty),
		Mixhash:    genesis.Mixhash,
		Coinbase:   genesis.Coinbase,
		Number:     genesis.Number,
		GasUsed:    genesis.GasUsed,
		ParentHash: genesis.ParentHash,
		BaseFee:    (*hexutil.Big)(genesis.BaseFee),
		Alloc:      make(jsonutil.LazySortedJsonMap[common.Address, GenesisAccount]),
	}

	// write genesis, but only reference code by code-hash, and don't encode the L2 predeploys to save space.
	for addr, account := range genesis.Alloc {
		var codeHash common.Hash
		if len(account.Code) > 0 {
			codeHash = crypto.Keccak256Hash(account.Code)
		}
		outAcc := GenesisAccount{
			CodeHash: codeHash,
			Nonce:    account.Nonce,
		}
		if account.Balance != nil && account.Balance.Cmp(common.Big0) != 0 {
			outAcc.Balance = (*hexutil.Big)(account.Balance)
		}
		if len(account.Storage) > 0 {
			outAcc.Storage = make(jsonutil.LazySortedJsonMap[common.Hash, common.Hash])
			for k, v := range account.Storage {
				outAcc.Storage[k] = v
			}
		}
		out.Alloc[addr] = outAcc
	}

	// write genesis alloc
	if err := writeGzipJSON(ctx.Path(OutputFlag.Name), out); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

func bytecode(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("expected hex-encoded bytecode as single argument; received %d arguments", ctx.NArg())
	}
	bc, err := hexutil.Decode(ctx.Args().First())
	if err != nil {
		return fmt.Errorf("failed to decode hex: %w", err)
	}
	bytecodesDir := ctx.Path(BytecodesDirFlag.Name)
	if err := os.MkdirAll(bytecodesDir, 0755); err != nil {
		return fmt.Errorf("failed to make bytecodes dir: %w", err)
	}
	return writeBytecode(bytecodesDir, bc, common.Address{})
}

func writeBytecode(bytecodesDir string, code []byte, addr common.Address) error {
	codeHash := crypto.Keccak256Hash(code)
	name := filepath.Join(bytecodesDir, fmt.Sprintf("%s.bin.gz", codeHash))
	_, err := os.Stat(name)
	if err == nil {
		// file already exists
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for pre-existing bytecode %s for address %s: %w", codeHash, addr, err)
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, 9)
	if err != nil {
		return fmt.Errorf("failed to construct gzip writer for bytecode %s: %w", codeHash, err)
	}
	if _, err := w.Write(code); err != nil {
		return fmt.Errorf("failed to write bytecode %s to gzip writer: %w", codeHash, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	// new bytecode
	if err := os.WriteFile(name, buf.Bytes(), 0755); err != nil {
		return fmt.Errorf("failed to write bytecode %s of account %s: %w", codeHash, addr, err)
	}
	return nil
}
