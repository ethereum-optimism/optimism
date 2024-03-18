package batching

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

	batchCalls  []batchCall
	singleCalls []elemCall

	mock.Mock
}

func makeTestRequest(i int) (*string, rpc.BatchElem) {
	out := new(string)
	return out, rpc.BatchElem{
		Method: "testing_foobar",
		Args:   []any{i},
		Result: out,
		Error:  nil,
	}
}

func (tc *batchTestCase) GetBatch(ctx context.Context, b []rpc.BatchElem) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return tc.Mock.MethodCalled("getBatch", b).Get(0).([]error)[0]
}

func (tc *batchTestCase) GetSingle(ctx context.Context, result any, method string, args ...any) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return tc.Mock.MethodCalled("getSingle", (*(result.(*interface{}))).(*string), method, args[0]).Get(0).([]error)[0]
}

var mockErr = errors.New("mockErr")

func (tc *batchTestCase) Run(t *testing.T) {
	keys := make([]int, tc.items)
	for i := 0; i < tc.items; i++ {
		keys[i] = i
	}

	makeBatchMock := func(bc batchCall) func(args mock.Arguments) {
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
	for _, bc := range tc.batchCalls {
		var batch []rpc.BatchElem
		for _, elem := range bc.elems {
			batch = append(batch, rpc.BatchElem{
				Method: "testing_foobar",
				Args:   []any{elem.id},
				Result: new(string),
				Error:  nil,
			})
		}
		if len(bc.elems) > 0 {
			tc.On("getBatch", batch).Once().Run(makeBatchMock(bc)).Return([]error{bc.rpcErr}) // wrap to preserve nil as type of error
		}
	}
	makeSingleMock := func(ec elemCall) func(args mock.Arguments) {
		return func(args mock.Arguments) {
			result := args[0].(*string)
			id := args[2].(int)
			require.Equal(t, ec.id, id, "element should match expected element")
			if ec.err {
				*result = ""
			} else {
				*result = fmt.Sprintf("mock result id %d", id)
			}
		}
	}
	// mock the results of unbatched calls
	for _, ec := range tc.singleCalls {
		var ret error
		if ec.err {
			ret = mockErr
		}
		tc.On("getSingle", new(string), "testing_foobar", ec.id).Once().Run(makeSingleMock(ec)).Return([]error{ret})
	}
	iter := NewIterativeBatchCall[int, *string](keys, makeTestRequest, tc.GetBatch, tc.GetSingle, tc.batchSize)
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
	for i, ec := range tc.singleCalls {
		ctx := context.Background()
		err := iter.Fetch(ctx)
		if err == io.EOF {
			require.Equal(t, i, len(tc.singleCalls)-1, "EOF only on last call")
		} else {
			require.False(t, iter.Complete())
			if ec.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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
			name:      "single element",
			items:     1,
			batchSize: 4,
			singleCalls: []elemCall{
				{id: 0, err: false},
			},
		},
		{
			name:      "unbatched",
			items:     4,
			batchSize: 1,
			singleCalls: []elemCall{
				{id: 0, err: false},
				{id: 1, err: false},
				{id: 2, err: false},
				{id: 3, err: false},
			},
		},
		{
			name:      "unbatched with retry",
			items:     4,
			batchSize: 1,
			singleCalls: []elemCall{
				{id: 0, err: false},
				{id: 1, err: true},
				{id: 2, err: false},
				{id: 3, err: false},
				{id: 1, err: false},
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
			items:     2,
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
						{id: 1, err: false},
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
