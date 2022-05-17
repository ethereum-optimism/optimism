package l1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type elemCall struct {
	id  int
	err bool
}

type batchCall struct {
	elems []elemCall
	err   error
}

type batchTestCase struct {
	name  string
	items int

	batchCalls []batchCall
	err        error

	maxRetry    int
	maxPerBatch int
	maxParallel int

	mock.Mock
}

func (tc *batchTestCase) Inputs() []rpc.BatchElem {
	out := make([]rpc.BatchElem, tc.items)
	for i := 0; i < tc.items; i++ {
		out[i] = rpc.BatchElem{
			Method: "testing_foobar",
			Args:   []interface{}{i},
			Result: nil,
			Error:  nil,
		}
	}
	return out
}

func (tc *batchTestCase) GetBatch(ctx context.Context, b []rpc.BatchElem) error {
	return tc.Mock.MethodCalled("get", b).Get(0).([]error)[0]
}

func (tc *batchTestCase) Run(t *testing.T) {
	requests := tc.Inputs()

	// mock all the results of the batch calls
	for bci, b := range tc.batchCalls {
		batchCall := b
		var batch []rpc.BatchElem
		for _, elem := range batchCall.elems {
			batch = append(batch, requests[elem.id])
		}
		tc.On("get", batch).Run(func(args mock.Arguments) {
			batch := args[0].([]rpc.BatchElem)
			for i := range batch {
				if batchCall.elems[i].err {
					batch[i].Error = fmt.Errorf("mock err batch-call %d, elem call %d", bci, i)
					batch[i].Result = nil
				} else {
					batch[i].Error = nil
					batch[i].Result = fmt.Sprintf("mock result batch-call %d, elem call %d", bci, i)
				}
			}
		}).Return([]error{batchCall.err}) // wrap to preserve nil as type of error
	}

	err := fetchBatched(context.Background(), testlog.Logger(t, log.LvlError), requests, tc.GetBatch, tc.maxRetry, tc.maxPerBatch, tc.maxParallel)
	assert.Equal(t, err, tc.err)

	tc.AssertExpectations(t)
}

func TestFetchBatched(t *testing.T) {
	testCases := []*batchTestCase{
		{
			name:        "empty",
			items:       0,
			batchCalls:  []batchCall{},
			err:         nil,
			maxRetry:    3,
			maxPerBatch: 10,
			maxParallel: 10,
		},
		{
			name:  "simple",
			items: 4,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: false},
						{id: 2, err: false},
						{id: 3, err: false},
					},
					err: nil,
				},
			},
			err:         nil,
			maxRetry:    3,
			maxPerBatch: 10,
			maxParallel: 10,
		},
		{
			name:  "split",
			items: 5,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: false},
						{id: 2, err: false},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 3, err: false},
						{id: 4, err: false},
					},
					err: nil,
				},
			},
			err:         nil,
			maxRetry:    2,
			maxPerBatch: 3,
			maxParallel: 10,
		},
		{
			name:  "batch split and parallel constrain",
			items: 3,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 1, err: false},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 2, err: false},
					},
					err: nil,
				},
			},
			err:         nil,
			maxRetry:    2,
			maxPerBatch: 1,
			maxParallel: 2,
		},
		{
			name:  "efficient retry",
			items: 5,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: true},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 2, err: false},
						{id: 3, err: false},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 4, err: false},
						{id: 1, err: false},
					},
					err: nil,
				},
			},
			err:         nil,
			maxRetry:    2,
			maxPerBatch: 2,
			maxParallel: 2,
		},
		{
			name:  "repeated sequential retries",
			items: 3,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: true},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 2, err: false},
						{id: 1, err: true},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 1, err: false},
					},
					err: nil,
				},
			},
			err:         nil,
			maxRetry:    2,
			maxPerBatch: 2,
			maxParallel: 1,
		},
		{
			name:  "too many retries",
			items: 3,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: true},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 2, err: false},
						{id: 1, err: true},
					},
					err: nil,
				},
				{
					elems: []elemCall{
						{id: 1, err: true},
					},
					err: nil,
				},
			},
			err:         TooManyRetries,
			maxRetry:    2,
			maxPerBatch: 2,
			maxParallel: 1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}

type parentErrBatchTestCase struct {
	mock.Mock
}

func (c *parentErrBatchTestCase) GetBatch(ctx context.Context, b []rpc.BatchElem) error {
	return c.Mock.MethodCalled("get", b).Get(0).([]error)[0]
}

func (c *parentErrBatchTestCase) Run(t *testing.T) {
	var requests []rpc.BatchElem
	for i := 0; i < 2; i++ {
		requests = append(requests, rpc.BatchElem{
			Method: "testing",
			Args:   []interface{}{i},
		})
	}

	// shouldn't retry if it's an error on the actual request
	expErr := errors.New("fail")
	c.On("get", requests).Run(func(args mock.Arguments) {
	}).Return([]error{expErr})
	err := fetchBatched(context.Background(), testlog.Logger(t, log.LvlError), requests, c.GetBatch, 2, 2, 1)
	assert.ErrorIs(t, err, expErr)
	c.AssertExpectations(t)
}

func TestFetchBatchedContextTimeout(t *testing.T) {
	var c parentErrBatchTestCase
	c.Run(t)
}
