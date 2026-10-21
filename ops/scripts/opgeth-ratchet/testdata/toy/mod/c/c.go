package c

import (
	"example.com/geth/types"

	"example.com/toy/a"
)

func Signer() int { return a.Signer() + types.Holesky }
