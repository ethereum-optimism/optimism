package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

type MockL2Client struct {
	MockEthClient
}

func (c *MockL2Client) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	out := c.Mock.MethodCalled("L2BlockRefByLabel", label)
	return out[0].(eth.L2BlockRef), *out[1].(*error)
}

func (m *MockL2Client) ExpectL2BlockRefByLabel(label eth.BlockLabel, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByLabel", label).Once().Return(ref, &err)
}

func (c *MockL2Client) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	out := c.Mock.MethodCalled("L2BlockRefByNumber", num)
	return out[0].(eth.L2BlockRef), *out[1].(*error)
}

func (m *MockL2Client) ExpectL2BlockRefByNumber(num uint64, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByNumber", num).Once().Return(ref, &err)
}

func (c *MockL2Client) L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	out := c.Mock.Called(hash)
	return out.Get(0).(eth.L2BlockRef), out.Error(1)
}

func (m *MockL2Client) ExpectL2BlockRefByHash(hash common.Hash, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByHash", hash).Once().Return(ref, err)
}

func (m *MockL2Client) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	out := m.Mock.Called(hash)
	return out.Get(0).(eth.SystemConfig), out.Error(1)
}

func (m *MockL2Client) ExpectSystemConfigByL2Hash(hash common.Hash, cfg eth.SystemConfig, err error) {
	m.Mock.On("SystemConfigByL2Hash", hash).Once().Return(cfg, err)
}

func (m *MockL2Client) OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error) {
	out := m.Mock.Called(blockHash)
	return out.Get(0).(*eth.OutputV0), out.Error(1)
}

func (m *MockL2Client) ExpectOutputV0AtBlock(blockHash common.Hash, output *eth.OutputV0, err error) {
	m.Mock.On("OutputV0AtBlock", blockHash).Once().Return(output, err)
}
