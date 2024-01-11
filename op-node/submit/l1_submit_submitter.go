package submit

import (
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

// L1SubmitTxData creates the transaction data for the L1Submit function
func L1SubmitTxData(index, length, gasPrice uint64, address common.Address, sign, commitment hexutil.Bytes) ([]byte, error) {
	parsed, err := bindings.L1DomiconCommitment.GetAbi()
	if err != nil {
		return nil, err
	}
	return l1SubmitTxData(parsed, index, length, gasPrice, address, sign, commitment)
}

func l1SubmitTxData(abi *abi.ABI, index, length, gasPrice uint64, address common.Address, sign, commitment []byte) ([]byte, error) {
	return abi.Pack(
		"SubmitCommitment", new(big.Int).SetUint64(index), new(big.Int).SetUint64(length), new(big.Int).SetUint64(gasPrice), address, sign, commitment)
}
