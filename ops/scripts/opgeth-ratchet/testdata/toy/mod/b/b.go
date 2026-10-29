package b

import (
	"example.com/geth/types"

	"example.com/toy/a"
)

func Deposit(tx types.Tx) bool { return tx.IsDeposit() && a.Signer() == 0 }
