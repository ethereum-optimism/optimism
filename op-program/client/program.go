package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	oppio "github.com/ethereum-optimism/optimism/op-program/io"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

// Main executes the client program in a detached context and exits the current process.
// The client runtime environment must be preset before calling this function.
func Main(logger log.Logger) {
	log.Info("Starting fault proof program client")
	preimageOracle := CreatePreimageChannel()
	preimageHinter := CreateHinterChannel()
	bootOracle := os.NewFile(BootRFd, "bootR")
	err := RunProgram(logger, bootOracle, preimageOracle, preimageHinter)
	if err != nil {
		log.Error("Program failed", "err", err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

// RunProgram executes the Program, while attached to an IO based pre-image oracle, to be served by a host.
func RunProgram(logger log.Logger, bootOracle io.Reader, preimageOracle io.ReadWriter, preimageHinter io.ReadWriter) error {
	bootReader := NewBootstrapOracleReader(bootOracle)
	bootInfo, err := bootReader.BootInfo()
	if err != nil {
		return fmt.Errorf("failed to read boot info: %w", err)
	}
	logger.Debug("Loaded boot info", "bootInfo", bootInfo)

	pClient := preimage.NewOracleClient(preimageOracle)
	hClient := preimage.NewHintWriter(preimageHinter)
	l1PreimageOracle := l1.NewPreimageOracle(pClient, hClient)
	l2PreimageOracle := l2.NewPreimageOracle(pClient, hClient)
	return runDerivation(
		logger,
		bootInfo.Rollup,
		bootInfo.L2ChainConfig,
		bootInfo.L1Head,
		bootInfo.L2Head,
		bootInfo.L2Claim,
		bootInfo.L2ClaimBlockNumber,
		l1PreimageOracle,
		l2PreimageOracle,
	)
}

// runDerivation executes the L2 state transition, given a minimal interface to retrieve data.
func runDerivation(logger log.Logger, cfg *rollup.Config, l2Cfg *params.ChainConfig, l1Head common.Hash, l2Head common.Hash, l2Claim common.Hash, l2ClaimBlockNum uint64, l1Oracle l1.Oracle, l2Oracle l2.Oracle) error {
	l1Source := l1.NewOracleL1Client(logger, l1Oracle, l1Head)
	engineBackend, err := l2.NewOracleBackedL2Chain(logger, l2Oracle, l2Cfg, l2Head)
	if err != nil {
		return fmt.Errorf("failed to create oracle-backed L2 chain: %w", err)
	}
	l2Source := l2.NewOracleEngine(cfg, logger, engineBackend)

	logger.Info("Starting derivation")
	d := cldr.NewDriver(logger, cfg, l1Source, l2Source, l2ClaimBlockNum)
	for {
		if err = d.Step(context.Background()); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
	}
	return d.ValidateClaim(eth.Bytes32(l2Claim))
}

func CreateHinterChannel() oppio.FileChannel {
	r := os.NewFile(HClientRFd, "preimage-hint-read")
	w := os.NewFile(HClientWFd, "preimage-hint-write")
	return oppio.NewReadWritePair(r, w)
}

// CreatePreimageChannel returns a FileChannel for the preimage oracle in a detached context
func CreatePreimageChannel() oppio.FileChannel {
	r := os.NewFile(PClientRFd, "preimage-oracle-read")
	w := os.NewFile(PClientWFd, "preimage-oracle-write")
	return oppio.NewReadWritePair(r, w)
}
