package dump

import (
	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi"
	"github.com/ethereum-optimism/optimism/l2geth/common"
)

type OvmDumpAccount struct {
	Address  common.Address         `json:"address"`
	Code     string                 `json:"code"`
	CodeHash string                 `json:"codeHash"`
	Storage  map[common.Hash]string `json:"storage"`
	ABI      abi.ABI                `json:"abi"`
	Nonce    uint64                 `json:"nonce"`
}

type OvmDump struct {
	Accounts map[string]OvmDumpAccount `json:"accounts"`
}
