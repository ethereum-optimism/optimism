// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/ethereum/go-ethereum/common"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"

	ethereum "github.com/ethereum/go-ethereum"

	mock "github.com/stretchr/testify/mock"

	types "github.com/ethereum/go-ethereum/core/types"
)

// MinimalEthClient is an autogenerated mock type for the MinimalEthClient type
type MinimalEthClient struct {
	mock.Mock
}

// FetchReceipts provides a mock function with given fields: ctx, blockHash
func (_m *MinimalEthClient) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error) {
	ret := _m.Called(ctx, blockHash)

	var r0 eth.BlockInfo
	var r1 types.Receipts
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) (eth.BlockInfo, types.Receipts, error)); ok {
		return rf(ctx, blockHash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) eth.BlockInfo); ok {
		r0 = rf(ctx, blockHash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eth.BlockInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) types.Receipts); ok {
		r1 = rf(ctx, blockHash)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(types.Receipts)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, common.Hash) error); ok {
		r2 = rf(ctx, blockHash)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// InfoByHash provides a mock function with given fields: ctx, hash
func (_m *MinimalEthClient) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	ret := _m.Called(ctx, hash)

	var r0 eth.BlockInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) (eth.BlockInfo, error)); ok {
		return rf(ctx, hash)
	}
	if rf, ok := ret.Get(0).(func(context.Context, common.Hash) eth.BlockInfo); ok {
		r0 = rf(ctx, hash)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eth.BlockInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, common.Hash) error); ok {
		r1 = rf(ctx, hash)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InfoByNumber provides a mock function with given fields: ctx, number
func (_m *MinimalEthClient) InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	ret := _m.Called(ctx, number)

	var r0 eth.BlockInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, uint64) (eth.BlockInfo, error)); ok {
		return rf(ctx, number)
	}
	if rf, ok := ret.Get(0).(func(context.Context, uint64) eth.BlockInfo); ok {
		r0 = rf(ctx, number)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(eth.BlockInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, uint64) error); ok {
		r1 = rf(ctx, number)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SubscribeNewHead provides a mock function with given fields: ctx, ch
func (_m *MinimalEthClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	ret := _m.Called(ctx, ch)

	var r0 ethereum.Subscription
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, chan<- *types.Header) (ethereum.Subscription, error)); ok {
		return rf(ctx, ch)
	}
	if rf, ok := ret.Get(0).(func(context.Context, chan<- *types.Header) ethereum.Subscription); ok {
		r0 = rf(ctx, ch)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ethereum.Subscription)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, chan<- *types.Header) error); ok {
		r1 = rf(ctx, ch)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewMinimalEthClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewMinimalEthClient creates a new instance of MinimalEthClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMinimalEthClient(t mockConstructorTestingTNewMinimalEthClient) *MinimalEthClient {
	mock := &MinimalEthClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
