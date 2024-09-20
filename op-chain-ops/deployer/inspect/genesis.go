package inspect

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

func GenesisCLI(cliCtx *cli.Context) error {
	cfg, err := readConfig(cliCtx)
	if err != nil {
		return err
	}

	st, err := bootstrapState(cfg)
	if err != nil {
		return err
	}

	genesis, err := st.ChainState.UnmarshalGenesis()
	if err != nil {
		return fmt.Errorf("failed to unmarshal genesis: %w", err)
	}

	if err := jsonutil.WriteJSON(genesis, ioutil.ToStdOutOrFileOrNoop(cfg.Outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	return nil
}

func chainIDStrToHash(in string) (common.Hash, error) {
	var chainIDBig *big.Int
	if strings.HasPrefix(in, "0x") {
		in = strings.TrimPrefix(in, "0x")
		var ok bool
		chainIDBig, ok = new(big.Int).SetString(in, 16)
		if !ok {
			return common.Hash{}, fmt.Errorf("failed to parse chain ID %s", in)
		}
	} else {
		inUint, err := strconv.ParseUint(in, 10, 64)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to parse chain ID %s: %w", in, err)
		}

		chainIDBig = new(big.Int).SetUint64(inUint)
	}

	return common.BigToHash(chainIDBig), nil
}
