package bridge

import (
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Bridge interface {
	Address() common.Address
	Contract() *bind.BoundContract
	GetDepositsByBlockRange(uint64, uint64) (map[common.Hash][]db.Deposit, error)
	GetWithdrawalsByBlockRange(uint64, uint64) (map[common.Hash][]db.Withdrawal, error)
}
