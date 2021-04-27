package eth

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestGasLimit(t *testing.T) {
	backend := &EthAPIBackend{
		extRPCEnabled: false,
		eth:           nil,
		gpo:           nil,
		verifier:      false,
		gasLimit:      0,
		UsingOVM:      true,
	}

	nonce := uint64(0)
	to := common.HexToAddress("0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c")
	value := big.NewInt(0)
	gasPrice := big.NewInt(0)
	data := []byte{}

	// Set the gas limit to 1 so that the transaction will not be
	// able to be added.
	gasLimit := uint64(1)
	tx := types.NewTransaction(nonce, to, value, gasLimit, gasPrice, data)

	err := backend.SendTx(context.Background(), tx)
	if err == nil {
		t.Fatal("Transaction with too large of gas limit accepted")
	}
	if err.Error() != fmt.Sprintf("Transaction gasLimit (%d) is greater than max gasLimit (%d)", gasLimit, backend.GasLimit()) {
		t.Fatalf("Unexpected error type: %s", err)
	}
}
