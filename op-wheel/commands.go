package wheel

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-wheel/cheat"
	"github.com/ethereum-optimism/optimism/op-wheel/engine"
)

const envVarPrefix = "OP_WHEEL"

func prefixEnvVars(name string) []string {
	return []string{envVarPrefix + "_" + name}
}

var (
	GlobalGethLogLvlFlag = &cli.GenericFlag{
		Name:    "geth-log-level",
		Usage:   "Set the global geth logging level",
		EnvVars: prefixEnvVars("GETH_LOG_LEVEL"),
		Value:   oplog.NewLvlFlagValue(log.LvlError),
	}
	DataDirFlag = &cli.StringFlag{
		Name:      "data-dir",
		Usage:     "Geth data dir location.",
		Required:  true,
		TakesFile: true,
		EnvVars:   prefixEnvVars("DATA_DIR"),
	}
	EngineEndpoint = &cli.StringFlag{
		Name:     "engine",
		Usage:    "Engine API RPC endpoint, can be HTTP/WS/IPC",
		Required: true,
		EnvVars:  prefixEnvVars("ENGINE"),
	}
	EngineJWTPath = &cli.StringFlag{
		Name:      "engine.jwt-secret",
		Usage:     "Path to JWT secret file used to authenticate Engine API communication with.",
		Required:  true,
		TakesFile: true,
		EnvVars:   prefixEnvVars("ENGINE_JWT_SECRET"),
	}
	FeeRecipientFlag = &cli.GenericFlag{
		Name:    "fee-recipient",
		Usage:   "fee-recipient of the block building",
		EnvVars: prefixEnvVars("FEE_RECIPIENT"),
		Value:   &TextFlag[*common.Address]{Value: &common.Address{1: 0x13, 2: 0x37}},
	}
	RandaoFlag = &cli.GenericFlag{
		Name:    "randao",
		Usage:   "randao value of the block building",
		EnvVars: prefixEnvVars("RANDAO"),
		Value:   &TextFlag[*common.Hash]{Value: &common.Hash{1: 0x13, 2: 0x37}},
	}
	BlockTimeFlag = &cli.Uint64Flag{
		Name:    "block-time",
		Usage:   "block time, interval of timestamps between blocks to build, in seconds",
		EnvVars: prefixEnvVars("BLOCK_TIME"),
		Value:   12,
	}
	BuildingTime = &cli.DurationFlag{
		Name:    "building-time",
		Usage:   "duration of of block building, this should be set to something lower than the block time.",
		EnvVars: prefixEnvVars("BUILDING_TIME"),
		Value:   time.Second * 6,
	}
	AllowGaps = &cli.BoolFlag{
		Name:    "allow-gaps",
		Usage:   "allow gaps in block building, like missed slots on the beacon chain.",
		EnvVars: prefixEnvVars("ALLOW_GAPS"),
	}
)

func ParseBuildingArgs(ctx *cli.Context) *engine.BlockBuildingSettings {
	return &engine.BlockBuildingSettings{
		BlockTime:    ctx.Uint64(BlockTimeFlag.Name),
		AllowGaps:    ctx.Bool(AllowGaps.Name),
		Random:       hashFlagValue(RandaoFlag.Name, ctx),
		FeeRecipient: addrFlagValue(FeeRecipientFlag.Name, ctx),
		BuildTime:    ctx.Duration(BuildingTime.Name),
	}
}

func CheatAction(readOnly bool, fn func(ctx *cli.Context, ch *cheat.Cheater) error) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		dataDir := ctx.String(DataDirFlag.Name)
		ch, err := cheat.OpenGethDB(dataDir, readOnly)
		if err != nil {
			return fmt.Errorf("failed to open geth db: %w", err)
		}
		return fn(ctx, ch)
	}
}

func CheatRawDBAction(readOnly bool, fn func(ctx *cli.Context, db ethdb.Database) error) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		dataDir := ctx.String(DataDirFlag.Name)
		db, err := cheat.OpenGethRawDB(dataDir, readOnly)
		if err != nil {
			return fmt.Errorf("failed to open raw geth db: %w", err)
		}
		return fn(ctx, db)
	}
}

func EngineAction(fn func(ctx *cli.Context, client client.RPC) error) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		jwtData, err := os.ReadFile(ctx.String(EngineJWTPath.Name))
		if err != nil {
			return fmt.Errorf("failed to read jwt: %w", err)
		}
		secret := common.HexToHash(strings.TrimSpace(string(jwtData)))
		endpoint := ctx.String(EngineEndpoint.Name)
		client, err := engine.DialClient(context.Background(), endpoint, secret)
		if err != nil {
			return fmt.Errorf("failed to dial Engine API endpoint %q: %w", endpoint, err)
		}
		return fn(ctx, client)
	}
}

type Text interface {
	encoding.TextUnmarshaler
	fmt.Stringer
	comparable
}

type TextFlag[T Text] struct {
	Value T
}

func (a *TextFlag[T]) Set(value string) error {
	var defaultValue T
	if a.Value == defaultValue {
		return fmt.Errorf("cannot unmarshal into nil value")
	}
	return a.Value.UnmarshalText([]byte(value))
}

func (a *TextFlag[T]) String() string {
	var defaultValue T
	if a.Value == defaultValue {
		return "<nil>"
	}
	return a.Value.String()
}

func (a *TextFlag[T]) Get() T {
	return a.Value
}

func (a *TextFlag[T]) Clone() any {
	var out TextFlag[T]
	if err := out.Set(a.String()); err != nil {
		panic(fmt.Errorf("cannot clone invalid text value: %w", err))
	}
	return &out
}

var _ cli.Generic = (*TextFlag[*common.Address])(nil)

func textFlag[T Text](name string, usage string, value T) *cli.GenericFlag {
	return &cli.GenericFlag{
		Name:     name,
		Usage:    usage,
		EnvVars:  prefixEnvVars(strings.ToUpper(name)),
		Required: true,
		Value:    &TextFlag[T]{Value: value},
	}
}

func addrFlag(name string, usage string) *cli.GenericFlag {
	return textFlag[*common.Address](name, usage, new(common.Address))
}

func bytesFlag(name string, usage string) *cli.GenericFlag {
	return textFlag[*hexutil.Bytes](name, usage, new(hexutil.Bytes))
}

func hashFlag(name string, usage string) *cli.GenericFlag {
	return textFlag[*common.Hash](name, usage, new(common.Hash))
}

func bigFlag(name string, usage string) *cli.GenericFlag {
	return textFlag[*big.Int](name, usage, new(big.Int))
}

func addrFlagValue(name string, ctx *cli.Context) common.Address {
	return *ctx.Generic(name).(*TextFlag[*common.Address]).Value
}

func bytesFlagValue(name string, ctx *cli.Context) hexutil.Bytes {
	return *ctx.Generic(name).(*TextFlag[*hexutil.Bytes]).Value
}

func hashFlagValue(name string, ctx *cli.Context) common.Hash {
	return *ctx.Generic(name).(*TextFlag[*common.Hash]).Value
}

func bigFlagValue(name string, ctx *cli.Context) *big.Int {
	return ctx.Generic(name).(*TextFlag[*big.Int]).Value
}

var (
	CheatStorageGetCmd = &cli.Command{
		Name:    "get",
		Aliases: []string{"read"},
		Flags: []cli.Flag{
			DataDirFlag,
			addrFlag("address", "Address to read storage of"),
			hashFlag("key", "key in storage of address to read value"),
		},
		Action: CheatAction(true, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.StorageGet(addrFlagValue("address", ctx), hashFlagValue("key", ctx), ctx.App.Writer))
		}),
	}
	CheatStorageSetCmd = &cli.Command{
		Name:    "set",
		Aliases: []string{"write"},
		Flags: []cli.Flag{
			DataDirFlag,
			addrFlag("address", "Address to write storage of"),
			hashFlag("key", "key in storage of address to set value of"),
			hashFlag("value", "the value to write"),
		},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.StorageSet(addrFlagValue("address", ctx), hashFlagValue("key", ctx), hashFlagValue("value", ctx)))
		}),
	}
	CheatStorageReadAll = &cli.Command{
		Name:    "read-all",
		Aliases: []string{"get-all"},
		Usage:   "Read all storage of the given account",
		Flags:   []cli.Flag{DataDirFlag, addrFlag("address", "Address to read all storage of")},
		Action: CheatAction(true, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.StorageReadAll(addrFlagValue("address", ctx), ctx.App.Writer))
		}),
	}
	CheatStorageDiffCmd = &cli.Command{
		Name:  "diff",
		Usage: "Diff the storage of accounts A and B",
		Flags: []cli.Flag{DataDirFlag, hashFlag("a", "address of account A"), hashFlag("b", "address of account B")},
		Action: CheatAction(true, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.StorageDiff(ctx.App.Writer, addrFlagValue("a", ctx), addrFlagValue("b", ctx)))
		}),
	}
	CheatStoragePatchCmd = &cli.Command{
		Name:  "patch",
		Usage: "Apply storage patch from STDIN to the given account address",
		Flags: []cli.Flag{DataDirFlag, addrFlag("address", "Address to patch storage of")},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.StoragePatch(os.Stdin, addrFlagValue("address", ctx)))
		}),
	}
	CheatStorageCmd = &cli.Command{
		Name: "storage",
		Subcommands: []*cli.Command{
			CheatStorageGetCmd,
			CheatStorageSetCmd,
			CheatStorageReadAll,
			CheatStorageDiffCmd,
			CheatStoragePatchCmd,
		},
	}
	CheatSetBalanceCmd = &cli.Command{
		Name: "balance",
		Flags: []cli.Flag{
			DataDirFlag,
			addrFlag("address", "Address to change balance of"),
			bigFlag("balance", "New balance of the account"),
		},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.SetBalance(addrFlagValue("address", ctx), bigFlagValue("balance", ctx)))
		}),
	}
	CheatSetCodeCmd = &cli.Command{
		Name: "code",
		Flags: []cli.Flag{
			DataDirFlag,
			addrFlag("address", "Address to change code of"),
			bytesFlag("code", "New code of the account"),
		},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.SetCode(addrFlagValue("address", ctx), bytesFlagValue("code", ctx)))
		}),
	}
	CheatSetNonceCmd = &cli.Command{
		Name: "nonce",
		Flags: []cli.Flag{
			DataDirFlag,
			addrFlag("address", "Address to change nonce of"),
			bigFlag("nonce", "New nonce of the account"),
		},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			return ch.RunAndClose(cheat.SetNonce(addrFlagValue("address", ctx), bigFlagValue("balance", ctx).Uint64()))
		}),
	}
	CheatOvmOwnersCmd = &cli.Command{
		Name: "ovm-owners",
		Flags: []cli.Flag{
			DataDirFlag,
			&cli.StringFlag{
				Name:     "config",
				Usage:    "Path to JSON config of OVM address replacements to apply.",
				Required: true,
				EnvVars:  prefixEnvVars("OVM_OWNERS"),
				Value:    "ovm-owners.json",
			},
		},
		Action: CheatAction(false, func(ctx *cli.Context, ch *cheat.Cheater) error {
			confData, err := os.ReadFile(ctx.String("config"))
			if err != nil {
				return fmt.Errorf("failed to read OVM owners JSON config file: %w", err)
			}
			var conf cheat.OvmOwnersConfig
			if err := json.Unmarshal(confData, &conf); err != nil {
				return err
			}
			return ch.RunAndClose(cheat.OvmOwners(&conf))
		}),
	}
	CheatPrintHeadBlock = &cli.Command{
		Name:  "head-block",
		Usage: "dump head block as JSON",
		Flags: []cli.Flag{
			DataDirFlag,
		},
		Action: CheatRawDBAction(true, func(c *cli.Context, db ethdb.Database) error {
			enc := json.NewEncoder(c.App.Writer)
			enc.SetIndent("  ", "  ")
			block := rawdb.ReadHeadBlock(db)
			if block == nil {
				return enc.Encode(nil)
			}
			return enc.Encode(engine.RPCBlock{
				Header:       *block.Header(),
				Transactions: block.Transactions(),
			})
		}),
	}
	CheatPrintHeadHeader = &cli.Command{
		Name:  "head-header",
		Usage: "dump head header as JSON",
		Flags: []cli.Flag{
			DataDirFlag,
		},
		Action: CheatRawDBAction(true, func(c *cli.Context, db ethdb.Database) error {
			enc := json.NewEncoder(c.App.Writer)
			enc.SetIndent("  ", "  ")
			return enc.Encode(rawdb.ReadHeadHeader(db))
		}),
	}
	EngineBlockCmd = &cli.Command{
		Name:  "block",
		Usage: "build the next block using the Engine API",
		Flags: []cli.Flag{
			EngineEndpoint, EngineJWTPath,
			FeeRecipientFlag, RandaoFlag, BlockTimeFlag, BuildingTime, AllowGaps,
		},
		// TODO: maybe support transaction and tx pool engine flags, since we use op-geth?
		// TODO: reorg flag
		// TODO: finalize/safe flag

		Action: EngineAction(func(ctx *cli.Context, client client.RPC) error {
			settings := ParseBuildingArgs(ctx)
			status, err := engine.Status(context.Background(), client)
			if err != nil {
				return err
			}
			payload, err := engine.BuildBlock(context.Background(), client, status, settings)
			if err != nil {
				return err
			}
			_, err = io.WriteString(ctx.App.Writer, payload.BlockHash.String())
			return err
		}),
	}
	EngineAutoCmd = &cli.Command{
		Name:        "auto",
		Usage:       "Run a proof-of-nothing chain with fixed block time.",
		Description: "The block time can be changed. The execution engine must be synced to a post-Merge state first.",
		Flags: append(append([]cli.Flag{
			EngineEndpoint, EngineJWTPath,
			FeeRecipientFlag, RandaoFlag, BlockTimeFlag, BuildingTime, AllowGaps,
		}, oplog.CLIFlags(envVarPrefix)...), opmetrics.CLIFlags(envVarPrefix)...),
		Action: EngineAction(func(ctx *cli.Context, client client.RPC) error {
			logCfg := oplog.ReadCLIConfig(ctx)
			l := oplog.NewLogger(oplog.AppOut(ctx), logCfg)
			oplog.SetGlobalLogHandler(l.GetHandler())

			settings := ParseBuildingArgs(ctx)
			// TODO: finalize/safe flag

			metricsCfg := opmetrics.ReadCLIConfig(ctx)

			return opservice.CloseAction(func(ctx context.Context, shutdown <-chan struct{}) error {
				registry := opmetrics.NewRegistry()
				metrics := engine.NewMetrics("wheel", registry)
				if metricsCfg.Enabled {
					l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
					metricsSrv, err := opmetrics.StartServer(registry, metricsCfg.ListenAddr, metricsCfg.ListenPort)
					if err != nil {
						return fmt.Errorf("failed to start metrics server: %w", err)
					}
					defer func() {
						if err := metricsSrv.Stop(context.Background()); err != nil {
							l.Error("failed to stop metrics server: %w", err)
						}
					}()
				}
				return engine.Auto(ctx, metrics, client, l, shutdown, settings)
			})
		}),
	}
	EngineStatusCmd = &cli.Command{
		Name:  "status",
		Flags: []cli.Flag{EngineEndpoint, EngineJWTPath},
		Action: EngineAction(func(ctx *cli.Context, client client.RPC) error {
			stat, err := engine.Status(context.Background(), client)
			if err != nil {
				return err
			}
			enc := json.NewEncoder(ctx.App.Writer)
			enc.SetIndent("", "  ")
			return enc.Encode(stat)
		}),
	}
	EngineCopyCmd = &cli.Command{
		Name: "copy",
		Flags: []cli.Flag{
			EngineEndpoint, EngineJWTPath,
			&cli.StringFlag{
				Name:     "source",
				Usage:    "Unauthenticated regular eth JSON RPC to pull block data from, can be HTTP/WS/IPC.",
				Required: true,
				EnvVars:  prefixEnvVars("ENGINE"),
			},
		},
		Action: EngineAction(func(ctx *cli.Context, dest client.RPC) error {
			rpcClient, err := rpc.DialOptions(context.Background(), ctx.String("source"))
			if err != nil {
				return fmt.Errorf("failed to dial engine source endpoint: %w", err)
			}
			source := client.NewBaseRPCClient(rpcClient)
			return engine.Copy(context.Background(), source, dest)
		}),
	}

	EngineSetForkchoiceCmd = &cli.Command{
		Name:        "set-forkchoice",
		Description: "Set forkchoice, specify unsafe, safe and finalized blocks by number",
		Flags: []cli.Flag{
			EngineEndpoint, EngineJWTPath,
			&cli.Uint64Flag{
				Name:     "unsafe",
				Usage:    "Block number of block to set as latest block",
				Required: true,
				EnvVars:  prefixEnvVars("UNSAFE"),
			},
			&cli.Uint64Flag{
				Name:     "safe",
				Usage:    "Block number of block to set as safe block",
				Required: true,
				EnvVars:  prefixEnvVars("SAFE"),
			},
			&cli.Uint64Flag{
				Name:     "finalized",
				Usage:    "Block number of block to set as finalized block",
				Required: true,
				EnvVars:  prefixEnvVars("FINALIZED"),
			},
		},
		Action: EngineAction(func(ctx *cli.Context, client client.RPC) error {
			return engine.SetForkchoice(ctx.Context, client, ctx.Uint64("finalized"), ctx.Uint64("safe"), ctx.Uint64("unsafe"))
		}),
	}

	EngineJSONCmd = &cli.Command{
		Name:        "json",
		Description: "read json values from remaining args, or STDIN, and use them as RPC params to call the engine RPC method (first arg)",
		Flags: []cli.Flag{
			EngineEndpoint, EngineJWTPath,
			&cli.BoolFlag{
				Name:     "stdin",
				Usage:    "Read params from stdin instead",
				Required: false,
				EnvVars:  prefixEnvVars("STDIN"),
			},
		},
		ArgsUsage: "<rpc-method-name> [params...]",
		Action: EngineAction(func(ctx *cli.Context, client client.RPC) error {
			if ctx.NArg() == 0 {
				return fmt.Errorf("expected at least 1 argument: RPC method name")
			}
			var r io.Reader
			var args []string
			if ctx.Bool("stdin") {
				r = ctx.App.Reader
			} else {
				args = ctx.Args().Tail()
			}
			return engine.RawJSONInteraction(ctx.Context, client, ctx.Args().Get(0), args, r, ctx.App.Writer)
		}),
	}
)

var CheatCmd = &cli.Command{
	Name:  "cheat",
	Usage: "Cheating commands to modify a Geth database.",
	Description: "Each sub-command opens a Geth database, applies the cheat, and then saves and closes the database." +
		"The Geth node will live in its own false reality, other nodes cannot sync the cheated state if they process the blocks.",
	Subcommands: []*cli.Command{
		CheatStorageCmd,
		CheatSetBalanceCmd,
		CheatSetCodeCmd,
		CheatSetNonceCmd,
		CheatOvmOwnersCmd,
		CheatPrintHeadBlock,
		CheatPrintHeadHeader,
	},
}

var EngineCmd = &cli.Command{
	Name:        "engine",
	Usage:       "Engine API commands to build/reorg/finalize blocks.",
	Description: "Each sub-command dials the engine API endpoint (with provided JWT secret) and then runs the action",
	Subcommands: []*cli.Command{
		EngineBlockCmd,
		EngineAutoCmd,
		EngineStatusCmd,
		EngineCopyCmd,
		EngineSetForkchoiceCmd,
		EngineJSONCmd,
	},
}
