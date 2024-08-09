package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type VMType string

var cannonVMType VMType = "cannon"
var mtVMType VMType = "cannon-mt"

var VMTypeFlag = &cli.StringFlag{
	Name:     "type",
	Usage:    "VM type to create state for. Options are 'cannon' (default), 'cannon-mt'",
	Value:    "cannon",
	Required: false,
}

func vmTypeFromString(ctx *cli.Context) (VMType, error) {
	if vmTypeStr := ctx.String(VMTypeFlag.Name); vmTypeStr == string(cannonVMType) {
		return cannonVMType, nil
	} else if vmTypeStr == string(mtVMType) {
		return mtVMType, nil
	} else {
		return "", fmt.Errorf("unknown VM type %q", vmTypeStr)
	}
}
