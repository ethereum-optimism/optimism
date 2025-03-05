package inspect

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/urfave/cli/v2"
)

func GenesisCLI(cliCtx *cli.Context) error {
	cfg, err := readConfig(cliCtx)
	if err != nil {
		return err
	}

	globalState, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	l2Genesis, _, err := pipeline.RenderGenesisAndRollup(globalState, cfg.ChainID, nil)
	if err != nil {
		return fmt.Errorf("failed to generate genesis block: %w", err)
	}

	if err := jsonutil.WriteJSON(l2Genesis, ioutil.ToStdOutOrFileOrNoop(cfg.Outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	return nil
}
