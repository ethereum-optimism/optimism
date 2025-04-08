package eth

import "github.com/ethereum/go-ethereum/common/hexutil"

type ExecutionWitness struct {
	Keys  []hexutil.Bytes `json:"keys"`
	Codes []hexutil.Bytes `json:"codes"`
	State []hexutil.Bytes `json:"state"`
}
