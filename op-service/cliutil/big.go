package cliutil

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/urfave/cli/v2"
)

var ErrFlagBlank = errors.New("cannot parse blank big int flag")

func BigIntFlag(cliCtx *cli.Context, flagName string) (*big.Int, error) {
	intStr := cliCtx.String(flagName)
	if intStr == "" {
		return nil, ErrFlagBlank
	}
	base := 10
	if strings.HasPrefix(intStr, "0x") {
		base = 16
		intStr = intStr[2:]
	}
	out, ok := new(big.Int).SetString(intStr, base)
	if !ok {
		return nil, fmt.Errorf("error parsing bigint flag '%s'", intStr)
	}
	return out, nil
}
