package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSuperNodeClient_SuperRootAtTimestamp(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)
		client := NewSuperNodeClient(rpc)

		timestamp := uint64(245)

		chainA := eth.ChainIDFromUInt64(1)
		chainB := eth.ChainIDFromUInt64(4)
		expected := eth.SuperRootAtTimestampResponse{
			CurrentL1: eth.BlockID{
				Number: 305,
				Hash:   common.Hash{0xdd, 0xee, 0xff},
			},
			ChainIDs: []eth.ChainID{chainA, chainB},
			OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
				chainA: {
					Output: &eth.OutputResponse{
						Version:    eth.Bytes32{0x01},
						OutputRoot: eth.Bytes32{0x11, 0x12},
						BlockRef: eth.L2BlockRef{
							Hash:       common.Hash{0x22},
							Number:     472,
							ParentHash: common.Hash{0xdd},
							Time:       9895839,
							L1Origin: eth.BlockID{
								Hash:   common.Hash{0xee},
								Number: 9802,
							},
							SequenceNumber: 4982,
						},
						WithdrawalStorageRoot: common.Hash{0xff},
						StateRoot:             common.Hash{0xaa},
					},
					RequiredL1: eth.BlockID{
						Hash:   common.Hash{0xbb},
						Number: 7842,
					},
				},
			},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: eth.BlockID{
					Hash:   common.Hash{0xcc},
					Number: 7411111,
				},
				Super: eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{
					ChainID: chainA,
					Output:  eth.Bytes32{0xa1},
				}, eth.ChainIDAndOutput{
					ChainID: chainB,
					Output:  eth.Bytes32{0xa2},
				}),
				SuperRoot: eth.Bytes32{0xdd},
			},
		}
		rpc.On("CallContext", ctx, new(eth.SuperRootAtTimestampResponse),
			"superroot_atTimestamp", []any{hexutil.Uint64(timestamp)}).Run(func(args mock.Arguments) {
			*args[1].(*eth.SuperRootAtTimestampResponse) = expected
		}).Return([]error{nil})
		result, err := client.SuperRootAtTimestamp(ctx, timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)
		client := NewSuperNodeClient(rpc)

		timestamp := uint64(245)

		chainA := eth.ChainIDFromUInt64(1)
		chainB := eth.ChainIDFromUInt64(4)
		expected := eth.SuperRootAtTimestampResponse{
			CurrentL1: eth.BlockID{
				Number: 305,
				Hash:   common.Hash{0xdd, 0xee, 0xff},
			},
			ChainIDs: []eth.ChainID{chainA, chainB},
			OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
				chainA: {
					Output: &eth.OutputResponse{
						Version:    eth.Bytes32{0x01},
						OutputRoot: eth.Bytes32{0x11, 0x12},
						BlockRef: eth.L2BlockRef{
							Hash:       common.Hash{0x22},
							Number:     472,
							ParentHash: common.Hash{0xdd},
							Time:       9895839,
							L1Origin: eth.BlockID{
								Hash:   common.Hash{0xee},
								Number: 9802,
							},
							SequenceNumber: 4982,
						},
						WithdrawalStorageRoot: common.Hash{0xff},
						StateRoot:             common.Hash{0xaa},
					},
					RequiredL1: eth.BlockID{
						Hash:   common.Hash{0xbb},
						Number: 7842,
					},
				},
			},
			Data: nil, // No super root found, so data is nil.
		}
		rpc.On("CallContext", ctx, new(eth.SuperRootAtTimestampResponse),
			"superroot_atTimestamp", []any{hexutil.Uint64(timestamp)}).Run(func(args mock.Arguments) {
			*args[1].(*eth.SuperRootAtTimestampResponse) = expected
		}).Return([]error{nil})
		result, err := client.SuperRootAtTimestamp(ctx, timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("Error", func(t *testing.T) {
		ctx := context.Background()
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)
		client := NewSuperNodeClient(rpc)

		timestamp := uint64(245)

		rpc.On("CallContext", ctx, new(eth.SuperRootAtTimestampResponse),
			"superroot_atTimestamp", []any{hexutil.Uint64(timestamp)}).Return([]error{errors.New("blah blah blah: not found")})
		_, err := client.SuperRootAtTimestamp(ctx, timestamp)
		require.NotErrorIs(t, err, ethereum.NotFound) // should not convert to not found even though it contains not found
		require.NotNil(t, err)
	})
}
