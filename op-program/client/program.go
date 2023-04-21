package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	cldr "github.com/ethereum-optimism/optimism/op-program/client/driver"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/preimage"
)

var (
	ErrClaimNotValid = errors.New("invalid claim")
)

// ClientProgram executes the Program, while attached to an IO based pre-image oracle, to be served by a host.
func ClientProgram(
	logger log.Logger,
	cfg *rollup.Config,
	l2Cfg *params.ChainConfig,
	l1Head common.Hash,
	l2Head common.Hash,
	l2Claim common.Hash,
	l2ClaimBlockNumber uint64,
	preimageOracle io.ReadWriter,
	preimageHinter io.ReadWriter,
) error {
	pClient := preimage.NewOracleClient(preimageOracle)
	hClient := preimage.NewHintWriter(preimageHinter)
	l1PreimageOracle := l1.NewPreimageOracle(pClient, hClient)
	l2PreimageOracle := l2.NewPreimageOracle(pClient, hClient)
	return Program(logger, cfg, l2Cfg, l1Head, l2Head, l2Claim, l2ClaimBlockNumber, l1PreimageOracle, l2PreimageOracle)
}

// Program executes the L2 state transition, given a minimal interface to retrieve data.
func Program(logger log.Logger, cfg *rollup.Config, l2Cfg *params.ChainConfig, l1Head common.Hash, l2Head common.Hash, l2Claim common.Hash, l2ClaimBlockNum uint64, l1Oracle l1.Oracle, l2Oracle l2.Oracle) error {
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
	if !d.ValidateClaim(eth.Bytes32(l2Claim)) {
		return ErrClaimNotValid
	}
	logger.Info("Derivation complete", "head", d.SafeHead())
	return nil
}
