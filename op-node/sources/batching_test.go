package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"
)

type elemCall struct {
	id  int
	err bool
}

type batchCall struct {
	elems  []elemCall
	rpcErr error
	err    string
	// Artificial delay to add before returning the call
	duration time.Duration
	makeCtx  func() context.Context
}

type batchTestCase struct {
	name  string
	items int

	batchSize int

	batchCalls []batchCall

	mock.Mock
}

func makeTestRequest(i int) (*string, rpc.BatchElem) {
	out := new(string)
	return out, rpc.BatchElem{
		Method: "testing_foobar",
		Args:   []interface{}{i},
		Result: out,
		Error:  nil,
	}
}

func makeTestResults() func(keys []int, values []*string) ([]*string, error) {
	return func(keys []int, values []*string) ([]*string, error) {
		return values, nil
	}
}

func (tc *batchTestCase) GetBatch(ctx context.Context, b []rpc.BatchElem) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return tc.Mock.MethodCalled("get", b).Get(0).([]error)[0]
}

var mockErr = errors.New("mockErr")

func (tc *batchTestCase) Run(t *testing.T) {
	keys := make([]int, tc.items)
	for i := 0; i < tc.items; i++ {
		keys[i] = i
	}

	makeMock := func(bci int, bc batchCall) func(args mock.Arguments) {
		return func(args mock.Arguments) {
			batch := args[0].([]rpc.BatchElem)
			for i, elem := range batch {
				id := elem.Args[0].(int)
				expectedID := bc.elems[i].id
				require.Equal(t, expectedID, id, "batch element should match expected batch element")
				if bc.elems[i].err {
					batch[i].Error = mockErr
					*batch[i].Result.(*string) = ""
				} else {
					batch[i].Error = nil
					*batch[i].Result.(*string) = fmt.Sprintf("mock result id %d", id)
				}
			}
			time.Sleep(bc.duration)
		}
	}
	// mock all the results of the batch calls
	for bci, bc := range tc.batchCalls {
		var batch []rpc.BatchElem
		for _, elem := range bc.elems {
			batch = append(batch, rpc.BatchElem{
				Method: "testing_foobar",
				Args:   []interface{}{elem.id},
				Result: new(string),
				Error:  nil,
			})
		}
		if len(bc.elems) > 0 {
			tc.On("get", batch).Once().Run(makeMock(bci, bc)).Return([]error{bc.rpcErr}) // wrap to preserve nil as type of error
		}
	}
	iter := NewIterativeBatchCall[int, *string, []*string](keys, makeTestRequest, makeTestResults(), tc.GetBatch, tc.batchSize)
	for i, bc := range tc.batchCalls {
		ctx := context.Background()
		if bc.makeCtx != nil {
			ctx = bc.makeCtx()
		}

		err := iter.Fetch(ctx)
		if err == io.EOF {
			require.Equal(t, i, len(tc.batchCalls)-1, "EOF only on last call")
		} else {
			require.False(t, iter.Complete())
			if bc.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, bc.err)
			}
		}
	}
	require.True(t, iter.Complete(), "batch iter should be complete after the expected calls")
	out, err := iter.Result()
	require.NoError(t, err)
	for i, v := range out {
		require.NotNil(t, v)
		require.Equal(t, fmt.Sprintf("mock result id %d", i), *v)
	}
	out2, err := iter.Result()
	require.NoError(t, err)
	require.Equal(t, out, out2, "cached result should match")
	require.Equal(t, io.EOF, iter.Fetch(context.Background()), "fetch after completion should EOF")

	tc.AssertExpectations(t)
}

func TestFetchBatched(t *testing.T) {
	testCases := []*batchTestCase{
		{
			name:       "empty",
			items:      0,
			batchCalls: []batchCall{},
		},
		{
			name:      "simple",
			items:     4,
			batchSize: 4,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: false},
						{id: 2, err: false},
						{id: 3, err: false},
					},
					err: "",
				},
			},
		},
		{
			name:      "split",
			items:     5,
			batchSize: 3,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: false},
						{id: 2, err: false},
					},
					err: "",
				},
				{
					elems: []elemCall{
						{id: 3, err: false},
						{id: 4, err: false},
					},
					err: "",
				},
			},
		},
		{
			name:      "efficient retry",
			items:     7,
			batchSize: 2,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: true},
					},
					err: "1 error occurred:",
				},
				{
					elems: []elemCall{
						{id: 2, err: false},
						{id: 3, err: false},
					},
					err: "",
				},
				{
					elems: []elemCall{ // in-process before retry even happens
						{id: 4, err: false},
						{id: 5, err: false},
					},
					err: "",
				},
				{
					elems: []elemCall{
						{id: 6, err: false},
						{id: 1, err: false}, // includes the element to retry
					},
					err: "",
				},
			},
		},
		{
			name:      "repeated sequential retries",
			items:     2,
			batchSize: 2,
			batchCalls: []batchCall{
				{
					elems: []elemCall{
						{id: 0, err: true},
						{id: 1, err: true},
					},
					err: "2 errors occurred:",
				},
				{
					elems: []elemCall{
						{id: 0, err: false},
						{id: 1, err: true},
					},
					err: "1 error occurred:",
				},
				{
					elems: []elemCall{
						{id: 1, err: false},
					},
					err: "",
				},
			},
		},
		{
			name:      "context timeout",
			items:     1,
			batchSize: 3,
			batchCalls: []batchCall{
				{
					elems: nil,
					err:   context.Canceled.Error(),
					makeCtx: func() context.Context {
						ctx, cancel := context.WithCancel(context.Background())
						cancel()
						return ctx
					},
				},
				{
					elems: []elemCall{
						{id: 0, err: false},
					},
					err: "",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, tc.Run)
	}
}
