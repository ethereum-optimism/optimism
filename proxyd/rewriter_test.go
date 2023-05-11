package proxyd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

type args struct {
	rctx RewriteContext
	req  *RPCReq
	res  *RPCRes
}

type rewriteTest struct {
	name        string
	args        args
	expected    RewriteResult
	expectedErr error
	check       func(*testing.T, args)
}

func TestRewriteRequest(t *testing.T) {
	tests := []rewriteTest{
		/* range scoped */
		{
			name: "eth_getLogs fromBlock latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": "latest"}})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(100).String(), p[0]["fromBlock"])
			},
		},
		{
			name: "eth_getLogs fromBlock within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": hexutil.Uint64(55).String()}})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(55).String(), p[0]["fromBlock"])
			},
		},
		{
			name: "eth_getLogs fromBlock out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": hexutil.Uint64(111).String()}})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		{
			name: "eth_getLogs toBlock latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"toBlock": "latest"}})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(100).String(), p[0]["toBlock"])
			},
		},
		{
			name: "eth_getLogs toBlock within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"toBlock": hexutil.Uint64(55).String()}})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(55).String(), p[0]["toBlock"])
			},
		},
		{
			name: "eth_getLogs toBlock out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"toBlock": hexutil.Uint64(111).String()}})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		{
			name: "eth_getLogs fromBlock, toBlock latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": "latest", "toBlock": "latest"}})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(100).String(), p[0]["fromBlock"])
				require.Equal(t, hexutil.Uint64(100).String(), p[0]["toBlock"])
			},
		},
		{
			name: "eth_getLogs fromBlock, toBlock within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": hexutil.Uint64(55).String(), "toBlock": hexutil.Uint64(77).String()}})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []map[string]interface{}
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, hexutil.Uint64(55).String(), p[0]["fromBlock"])
				require.Equal(t, hexutil.Uint64(77).String(), p[0]["toBlock"])
			},
		},
		{
			name: "eth_getLogs fromBlock, toBlock out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getLogs", Params: mustMarshalJSON([]map[string]interface{}{{"fromBlock": hexutil.Uint64(111).String(), "toBlock": hexutil.Uint64(222).String()}})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		/* default block parameter */
		{
			name: "eth_getCode omit block, should add",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getCode", Params: mustMarshalJSON([]string{"0x123"})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 2, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, hexutil.Uint64(100).String(), p[1])
			},
		},
		{
			name: "eth_getCode latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getCode", Params: mustMarshalJSON([]string{"0x123", "latest"})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 2, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, hexutil.Uint64(100).String(), p[1])
			},
		},
		{
			name: "eth_getCode within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getCode", Params: mustMarshalJSON([]string{"0x123", hexutil.Uint64(55).String()})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 2, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, hexutil.Uint64(55).String(), p[1])
			},
		},
		{
			name: "eth_getCode out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getCode", Params: mustMarshalJSON([]string{"0x123", hexutil.Uint64(111).String()})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		/* default block parameter, at position 2 */
		{
			name: "eth_getStorageAt omit block, should add",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getStorageAt", Params: mustMarshalJSON([]string{"0x123", "5"})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 3, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, "5", p[1])
				require.Equal(t, hexutil.Uint64(100).String(), p[2])
			},
		},
		{
			name: "eth_getStorageAt latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getStorageAt", Params: mustMarshalJSON([]string{"0x123", "5", "latest"})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 3, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, "5", p[1])
				require.Equal(t, hexutil.Uint64(100).String(), p[2])
			},
		},
		{
			name: "eth_getStorageAt within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getStorageAt", Params: mustMarshalJSON([]string{"0x123", "5", hexutil.Uint64(55).String()})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 3, len(p))
				require.Equal(t, "0x123", p[0])
				require.Equal(t, "5", p[1])
				require.Equal(t, hexutil.Uint64(55).String(), p[2])
			},
		},
		{
			name: "eth_getStorageAt out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getStorageAt", Params: mustMarshalJSON([]string{"0x123", "5", hexutil.Uint64(111).String()})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		/* default block parameter, at position 0 */
		{
			name: "eth_getBlockByNumber omit block, should add",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getBlockByNumber", Params: mustMarshalJSON([]string{})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 1, len(p))
				require.Equal(t, hexutil.Uint64(100).String(), p[0])
			},
		},
		{
			name: "eth_getBlockByNumber latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getBlockByNumber", Params: mustMarshalJSON([]string{"latest"})},
				res:  nil,
			},
			expected: RewriteOverrideRequest,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 1, len(p))
				require.Equal(t, hexutil.Uint64(100).String(), p[0])
			},
		},
		{
			name: "eth_getBlockByNumber within range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getBlockByNumber", Params: mustMarshalJSON([]string{hexutil.Uint64(55).String()})},
				res:  nil,
			},
			expected: RewriteNone,
			check: func(t *testing.T, args args) {
				var p []string
				err := json.Unmarshal(args.req.Params, &p)
				require.Nil(t, err)
				require.Equal(t, 1, len(p))
				require.Equal(t, hexutil.Uint64(55).String(), p[0])
			},
		},
		{
			name: "eth_getBlockByNumber out of range",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_getBlockByNumber", Params: mustMarshalJSON([]string{hexutil.Uint64(111).String()})},
				res:  nil,
			},
			expected:    RewriteOverrideError,
			expectedErr: ErrRewriteBlockOutOfRange,
		},
		{
			name: "eth_getStorageAt using rpc.BlockNumberOrHash",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req: &RPCReq{Method: "eth_getStorageAt", Params: mustMarshalJSON([]string{
					"0xae851f927ee40de99aabb7461c00f9622ab91d60",
					"0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08",
					"0x1c4840bcb3de3ac403c0075b46c2c47d4396c5b624b6e1b2874ec04e8879b483"})},
				res: nil,
			},
			expected: RewriteNone,
		},
	}

	// generalize tests for other methods with same interface and behavior
	tests = generalize(tests, "eth_getLogs", "eth_newFilter")
	tests = generalize(tests, "eth_getCode", "eth_getBalance")
	tests = generalize(tests, "eth_getCode", "eth_getTransactionCount")
	tests = generalize(tests, "eth_getCode", "eth_call")
	tests = generalize(tests, "eth_getBlockByNumber", "eth_getBlockTransactionCountByNumber")
	tests = generalize(tests, "eth_getBlockByNumber", "eth_getUncleCountByBlockNumber")
	tests = generalize(tests, "eth_getBlockByNumber", "eth_getTransactionByBlockNumberAndIndex")
	tests = generalize(tests, "eth_getBlockByNumber", "eth_getUncleByBlockNumberAndIndex")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RewriteRequest(tt.args.rctx, tt.args.req, tt.args.res)
			if result != RewriteOverrideError {
				require.Nil(t, err)
				require.Equal(t, tt.expected, result)
			} else {
				require.Equal(t, tt.expectedErr, err)
			}
			if tt.check != nil {
				tt.check(t, tt.args)
			}
		})
	}
}

func generalize(tests []rewriteTest, baseMethod string, generalizedMethod string) []rewriteTest {
	newCases := make([]rewriteTest, 0)
	for _, t := range tests {
		if t.args.req.Method == baseMethod {
			newName := strings.Replace(t.name, baseMethod, generalizedMethod, -1)
			var req *RPCReq
			var res *RPCRes

			if t.args.req != nil {
				req = &RPCReq{
					JSONRPC: t.args.req.JSONRPC,
					Method:  generalizedMethod,
					Params:  t.args.req.Params,
					ID:      t.args.req.ID,
				}
			}

			if t.args.res != nil {
				res = &RPCRes{
					JSONRPC: t.args.res.JSONRPC,
					Result:  t.args.res.Result,
					Error:   t.args.res.Error,
					ID:      t.args.res.ID,
				}
			}
			newCases = append(newCases, rewriteTest{
				name: newName,
				args: args{
					rctx: t.args.rctx,
					req:  req,
					res:  res,
				},
				expected:    t.expected,
				expectedErr: t.expectedErr,
				check:       t.check,
			})
		}
	}
	return append(tests, newCases...)
}

func TestRewriteResponse(t *testing.T) {
	type args struct {
		rctx RewriteContext
		req  *RPCReq
		res  *RPCRes
	}
	tests := []struct {
		name     string
		args     args
		expected RewriteResult
		check    func(*testing.T, args)
	}{
		{
			name: "eth_blockNumber latest",
			args: args{
				rctx: RewriteContext{latest: hexutil.Uint64(100)},
				req:  &RPCReq{Method: "eth_blockNumber"},
				res:  &RPCRes{Result: hexutil.Uint64(200)},
			},
			expected: RewriteOverrideResponse,
			check: func(t *testing.T, args args) {
				require.Equal(t, args.res.Result, hexutil.Uint64(100))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RewriteResponse(tt.args.rctx, tt.args.req, tt.args.res)
			require.Nil(t, err)
			require.Equal(t, tt.expected, result)
			if tt.check != nil {
				tt.check(t, tt.args)
			}
		})
	}
}
