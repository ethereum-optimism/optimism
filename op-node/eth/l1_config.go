package eth

import "github.com/ethereum/go-ethereum/common"

type SystemConfig struct {
	BatcherAddr common.Address `json:"batcherAddr"`
	Overhead    Bytes32        `json:"overhead"`
	Scalar      Bytes32        `json:"scalar"`
}
