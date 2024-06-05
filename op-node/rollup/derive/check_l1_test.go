package derive

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func TestVerifyNewL1Origin(t *testing.T) {
	t.Run("same height inconsistency", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xb}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.ErrorIs(t, err, ErrReset, "different origin at same height, must be a reorg")
	})
	t.Run("same height success", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.NoError(t, err, "same origin")
	})
	t.Run("parent-hash inconsistency", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123 + 1, Hash: common.Hash{0xb}, ParentHash: common.Hash{42}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.ErrorIs(t, err, ErrReset, "parent hash of new origin does not match")
	})
	t.Run("parent-hash success", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123 + 1, Hash: common.Hash{0xb}, ParentHash: common.Hash{0xa}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.NoError(t, err, "expecting block b just after a")
	})
	t.Run("failed canonical check", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		mockErr := errors.New("test error")
		l1F.ExpectL1BlockRefByNumber(123, eth.L1BlockRef{}, mockErr)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123 + 2, Hash: common.Hash{0xb}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.ErrorIs(t, err, ErrTemporary, "temporary fetching error")
		require.ErrorIs(t, err, mockErr, "wraps the underlying error")
	})
	t.Run("older not canonical", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(123, eth.L1BlockRef{Number: 123, Hash: common.Hash{42}}, nil)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123 + 2, Hash: common.Hash{0xb}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.ErrorIs(t, err, ErrReset, "block A is no longer canonical, need to reset")
	})
	t.Run("success older block", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectL1BlockRefByNumber(123, eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}, nil)
		a := eth.L1BlockRef{Number: 123, Hash: common.Hash{0xa}}
		b := eth.L1BlockRef{Number: 123 + 2, Hash: common.Hash{0xb}}
		err := VerifyNewL1Origin(context.Background(), a, l1F, b)
		require.NoError(t, err, "block A is still canonical, can proceed")
	})
}
