package testutils

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

type MockL2Client struct {
	MockEthClient
}

func (c *MockL2Client) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return c.Mock.MethodCalled("L2BlockRefByLabel", label).Get(0).(eth.L2BlockRef), nil
}

func (m *MockL2Client) ExpectL2BlockRefByLabel(label eth.BlockLabel, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByLabel", label).Once().Return(ref, &err)
}

func (c *MockL2Client) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return c.Mock.MethodCalled("L2BlockRefByNumber", num).Get(0).(eth.L2BlockRef), nil
}

func (m *MockL2Client) ExpectL2BlockRefByNumber(num uint64, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByNumber", num).Once().Return(ref, &err)
}

func (c *MockL2Client) L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error) {
	return c.Mock.MethodCalled("L2BlockRefByHash", hash).Get(0).(eth.L2BlockRef), nil
}

func (m *MockL2Client) ExpectL2BlockRefByHash(hash common.Hash, ref eth.L2BlockRef, err error) {
	m.Mock.On("L2BlockRefByHash", hash).Once().Return(ref, &err)
}

func (m *MockL2Client) SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error) {
	return m.Mock.MethodCalled("SystemConfigByL2Hash", hash).Get(0).(eth.SystemConfig), nil
}

func (m *MockL2Client) ExpectSystemConfigByL2Hash(hash common.Hash, cfg eth.SystemConfig, err error) {
	m.Mock.On("SystemConfigByL2Hash", hash).Once().Return(cfg, &err)
}
