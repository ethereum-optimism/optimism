package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

// recordingRPC reports the traffic of an RPC to a jsonrpc.Recorder.
//
// Requests go out with their params pre-encoded, and results come back raw and are decoded
// here, so the recorder sees the exact JSON without a second encoding pass.
// Subscriptions are not recorded, neither their requests nor their notifications.
type recordingRPC struct {
	inner RPC
	rec   jsonrpc.Recorder
}

var _ RPC = (*recordingRPC)(nil)

func newRecordingRPC(inner RPC, rec jsonrpc.Recorder) *recordingRPC {
	return &recordingRPC{inner: inner, rec: rec}
}

func (r *recordingRPC) Close() {
	r.inner.Close()
}

func (r *recordingRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	// The inner call always gets a pointer, so reject a bad result before the request is sent.
	if result != nil && reflect.TypeOf(result).Kind() != reflect.Pointer {
		return fmt.Errorf("call result parameter must be pointer or nil interface: %v", result)
	}
	params, rawArgs, err := encodeParams(args)
	if err != nil {
		return err
	}
	done := r.rec.RecordOutgoing(ctx, jsonrpc.Message{Method: method, Params: params})
	var raw json.RawMessage
	if err := r.inner.CallContext(ctx, &raw, method, rawArgs...); err != nil {
		recordError(ctx, done, err)
		return err
	}
	if done != nil {
		done(ctx, jsonrpc.Response{Result: raw})
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(raw, result)
}

func (r *recordingRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	raws := make([]json.RawMessage, len(b))
	elems := make([]rpc.BatchElem, len(b))
	msgs := make([]jsonrpc.Message, len(b))
	for i, el := range b {
		params, rawArgs, err := encodeParams(el.Args)
		if err != nil {
			return err
		}
		elems[i] = rpc.BatchElem{Method: el.Method, Args: rawArgs, Result: &raws[i]}
		msgs[i] = jsonrpc.Message{Method: el.Method, Params: params}
	}
	dones := make([]jsonrpc.RecordDone, len(b))
	for i, msg := range msgs {
		dones[i] = r.rec.RecordOutgoing(ctx, msg)
	}
	if err := r.inner.BatchCallContext(ctx, elems); err != nil {
		return err
	}
	for i := range b {
		if err := elems[i].Error; err != nil {
			recordError(ctx, dones[i], err)
			b[i].Error = err
			continue
		}
		if dones[i] != nil {
			dones[i](ctx, jsonrpc.Response{Result: raws[i]})
		}
		b[i].Error = json.Unmarshal(raws[i], b[i].Result)
	}
	return nil
}

func (r *recordingRPC) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	return r.inner.Subscribe(ctx, namespace, channel, args...)
}

// encodeParams encodes each argument once, returning the encoded params array and the
// arguments as json.RawMessage values that the inner RPC re-emits verbatim.
func encodeParams(args []any) (json.RawMessage, []any, error) {
	if args == nil {
		return nil, nil, nil
	}
	rawArgs := make([]any, len(args))
	var params bytes.Buffer
	params.WriteByte('[')
	for i, arg := range args {
		enc, err := json.Marshal(arg)
		if err != nil {
			return nil, nil, err
		}
		if i > 0 {
			params.WriteByte(',')
		}
		params.Write(enc)
		rawArgs[i] = json.RawMessage(enc)
	}
	params.WriteByte(']')
	return params.Bytes(), rawArgs, nil
}

// recordError records err as the response if it came from one: a JSON-RPC error, or a response
// without a result. Any other error means no response arrived.
func recordError(ctx context.Context, done jsonrpc.RecordDone, err error) {
	if done == nil {
		return
	}
	if errors.Is(err, rpc.ErrNoResult) {
		done(ctx, jsonrpc.Response{})
		return
	}
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) {
		return
	}
	jsonErr := &jsonrpc.Error{Code: rpcErr.ErrorCode(), Message: rpcErr.Error()}
	var dataErr rpc.DataError
	if errors.As(err, &dataErr) {
		jsonErr.Data = dataErr.ErrorData()
	}
	done(ctx, jsonrpc.Response{Error: jsonErr})
}
