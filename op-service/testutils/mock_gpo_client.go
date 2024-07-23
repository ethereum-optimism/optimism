package testutils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/mock"
)

type MockGasPriceOracle struct {
	mock.Mock
}

func (m *MockGasPriceOracle) BlobBaseFee(opts *bind.CallOpts) (*big.Int, error) {
	out := m.Mock.Called(opts)
	return out.Get(0).(*big.Int), out.Error(1)
}

func (m *MockGasPriceOracle) ExpectBlobBaseFee(blobBaseFee *big.Int, err error) {
	m.Mock.On("BlobBaseFee", mock.Anything).Once().Return(blobBaseFee, err)
}
