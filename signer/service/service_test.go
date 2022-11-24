package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/signer/service/provider/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func createTx() *types.Transaction {
	var aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
	accesses := types.AccessList{types.AccessTuple{
		Address:     aa,
		StorageKeys: []common.Hash{{0}},
	}}
	txdata := &types.DynamicFeeTx{
		ChainID:    params.AllEthashProtocolChanges.ChainID,
		Nonce:      0,
		To:         &aa,
		Gas:        30000,
		GasFeeCap:  big.NewInt(1),
		GasTipCap:  big.NewInt(1),
		AccessList: accesses,
		Data:       []byte{},
		Value:      big.NewInt(1),
	}
	tx := types.NewTx(txdata)
	return tx
}

func TestSignTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provider := mocks.NewMockSignatureProvider(ctrl)
	service := NewSignerServiceWithProvider(log.Root(), provider)

	tx := createTx()
	signer := types.LatestSignerForChainID(tx.ChainId())
	digest := signer.Hash(tx).Bytes()
	txraw, err := tx.MarshalBinary()
	if err != nil {
		panic(err)
	}

	var tests = []struct {
		testName    string
		keyName     string
		txraw       []byte
		digest      []byte
		wantErrCode int
	}{
		{"happy path", "key", txraw, digest, 0},
		{"invalid txraw", "key", append(txraw, 1), digest, -32010},
		{"invalid digest", "key", txraw, append(digest, 1), -32011},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if tt.wantErrCode == 0 {
				provider.EXPECT().
					Sign(gomock.Any(), gomock.Any(), tt.digest).
					Return([]byte("signature"), nil)
			}
			resp, err := service.SignTransaction(
				context.Background(),
				tt.txraw,
				tt.digest,
			)
			if tt.wantErrCode == 0 {
				assert.Nil(t, err)
				assert.NotEmpty(t, resp.Signature)
			} else {
				assert.NotNil(t, err)
				assert.Nil(t, resp)
				var rpcErr rpc.Error
				if errors.As(err, &rpcErr) {
					assert.Equal(t, tt.wantErrCode, rpcErr.ErrorCode())
				} else {
					assert.Fail(t, "returned error is not an rpc.Error")
				}
			}
		})
	}
}
