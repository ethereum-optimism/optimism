package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
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

	// got these values by passing above test txraw into `./bin/signer client sign`
	// and inspecting debug logs on local test server that print these values
	rawSignature, _ := hexutil.Decode("0x3045022100d4bc81a0c9bb31dd0bb3b613782b08f16e955ba49f91da94ecec9fd2af27d29e022024fb3ed228b5f0f69e9e285ebb63ad9fc026534d169052a20dbb20d2f2a55f32")
	publicKey, _ := hexutil.Decode("0x04429753a0893d9708c1765d1572a5e5ec0c2a841f4dc117fe20744da8850d6bcbc01ffe65e4c1973bfa954f90ffe2c3937e9776f4140090354e813027780d95ae")

	var tests = []struct {
		testName    string
		txraw       []byte
		digest      []byte
		wantErrCode int
	}{
		{"happy path", txraw, digest, 0},
		{"invalid txraw", append(txraw, 1), digest, -32010},
		{"invalid digest", txraw, append(digest, 1), -32011},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if tt.wantErrCode == 0 {
				provider.EXPECT().
					Sign(gomock.Any(), gomock.Any(), tt.digest).
					Return(rawSignature, nil)
				provider.EXPECT().
					GetPublicKey(gomock.Any(), gomock.Any()).
					Return(publicKey, nil)
			}
			resp, err := service.SignTransaction(
				context.Background(),
				tt.txraw,
				tt.digest,
			)
			if tt.wantErrCode == 0 {
				assert.Nil(t, err)
				if assert.NotNil(t, resp) {
					assert.NotEmpty(t, resp.Signature)
				}
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
