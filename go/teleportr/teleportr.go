package teleportr

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	bsscore "github.com/ethereum-optimism/optimism/go/bss-core"
	"github.com/ethereum-optimism/optimism/go/bss-core/dial"
	"github.com/ethereum-optimism/optimism/go/bss-core/metrics"
	"github.com/ethereum-optimism/optimism/go/bss-core/txmgr"
	"github.com/ethereum-optimism/optimism/go/teleportr/db"
	"github.com/ethereum-optimism/optimism/go/teleportr/drivers/disburser"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

func Main(gitVersion string) func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := NewConfig(cliCtx)
		if err != nil {
			return err
		}

		log.Info("Initializing teleportr")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var logHandler log.Handler
		if cfg.LogTerminal {
			logHandler = log.StreamHandler(os.Stdout, log.TerminalFormat(true))
		} else {
			logHandler = log.StreamHandler(os.Stdout, log.JSONFormat())
		}

		logLevel, err := log.LvlFromString(cfg.LogLevel)
		if err != nil {
			return err
		}

		log.Root().SetHandler(log.LvlFilterHandler(logLevel, logHandler))

		disburserPrivKey, disburserAddr, err := bsscore.ParseWalletPrivKeyAndContractAddr(
			"Teleportr", cfg.Mnemonic, cfg.DisburserHDPath,
			cfg.DisburserPrivKey, cfg.DisburserAddress,
		)
		if err != nil {
			return err
		}

		depositAddr, err := bsscore.ParseAddress(cfg.DepositAddress)
		if err != nil {
			return err
		}

		l1Client, err := dial.L1EthClientWithTimeout(ctx, cfg.L1EthRpc, cfg.DisableHTTP2)
		if err != nil {
			return err
		}
		defer l1Client.Close()

		l2Client, err := dial.L1EthClientWithTimeout(ctx, cfg.L2EthRpc, cfg.DisableHTTP2)
		if err != nil {
			return err
		}
		defer l2Client.Close()

		database, err := db.Open(db.Config{
			Host:      cfg.PostgresHost,
			Port:      uint16(cfg.PostgresPort),
			User:      cfg.PostgresUser,
			Password:  cfg.PostgresPassword,
			DBName:    cfg.PostgresDBName,
			EnableSSL: cfg.PostgresEnableSSL,
		})
		if err != nil {
			return err
		}
		defer database.Close()

		if cfg.MetricsServerEnable {
			go metrics.RunServer(cfg.MetricsHostname, cfg.MetricsPort)
		}

		chainID, err := l2Client.ChainID(ctx)
		if err != nil {
			return err
		}

		txManagerConfig := txmgr.Config{
			ResubmissionTimeout:       cfg.ResubmissionTimeout,
			ReceiptQueryInterval:      time.Second,
			NumConfirmations:          1, // L2 insta confs
			SafeAbortNonceTooLowCount: cfg.SafeAbortNonceTooLowCount,
		}

		teleportrDriver, err := disburser.NewDriver(disburser.Config{
			Name:                 "Teleportr",
			L1Client:             l1Client,
			L2Client:             l2Client,
			Database:             database,
			MaxTxSize:            cfg.MaxL2TxSize,
			NumConfirmations:     cfg.NumDepositConfirmations,
			DeployBlockNumber:    cfg.DepositDeployBlockNumber,
			FilterQueryMaxBlocks: cfg.FilterQueryMaxBlocks,
			DepositAddr:          depositAddr,
			DisburserAddr:        disburserAddr,
			ChainID:              chainID,
			PrivKey:              disburserPrivKey,
		})
		if err != nil {
			return err
		}

		teleportrService := bsscore.NewService(bsscore.ServiceConfig{
			Context:         ctx,
			Driver:          teleportrDriver,
			PollInterval:    cfg.PollInterval,
			ClearPendingTx:  false,
			L1Client:        l2Client,
			TxManagerConfig: txManagerConfig,
		})

		services := []*bsscore.Service{teleportrService}
		teleportr, err := bsscore.NewBatchSubmitter(ctx, cancel, services)
		if err != nil {
			return err
		}

		log.Info("Starting teleportr")

		err = teleportr.Start()
		if err != nil {
			return err
		}
		defer teleportr.Stop()

		log.Info("Teleportr started")

		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)
		<-interruptChannel

		return nil
	}
}

func Migrate() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := NewConfig(cliCtx)
		if err != nil {
			return err
		}

		log.Info("Initializing teleportr")

		database, err := db.Open(db.Config{
			Host:      cfg.PostgresHost,
			Port:      uint16(cfg.PostgresPort),
			User:      cfg.PostgresUser,
			Password:  cfg.PostgresPassword,
			DBName:    cfg.PostgresDBName,
			EnableSSL: cfg.PostgresEnableSSL,
		})
		if err != nil {
			return err
		}

		log.Info("Migrating database")
		if err := database.Migrate(); err != nil {
			return err
		}
		log.Info("Done")
		return nil
	}
}
