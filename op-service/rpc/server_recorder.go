package rpc

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
)

// setServerRecorder makes srv report its traffic to rec, through op-geth's server recording hook.
// It is the only place that hook is used.
func setServerRecorder(srv *gethrpc.Server, rec jsonrpc.Recorder) {
	if rec == nil {
		return
	}
	srv.SetRecorder(serverRecorder{rec: rec})
}

type serverRecorder struct {
	rec jsonrpc.Recorder
}

func (s serverRecorder) RecordIncoming(ctx context.Context, msg gethrpc.RecordedMsg) gethrpc.RecordDone {
	return adaptRecordDone(msg, s.rec.RecordIncoming(ctx, toMessage(msg)))
}

func (s serverRecorder) RecordOutgoing(ctx context.Context, msg gethrpc.RecordedMsg) gethrpc.RecordDone {
	return adaptRecordDone(msg, s.rec.RecordOutgoing(ctx, toMessage(msg)))
}

func toMessage(msg gethrpc.RecordedMsg) jsonrpc.Message {
	if msg.MsgIsNotification() {
		// Nothing reads notification params, and op-geth marshals them for every outgoing one.
		return jsonrpc.Message{Method: msg.MsgMethod(), Notification: true}
	}
	return jsonrpc.Message{Method: msg.MsgMethod(), Params: msg.MsgParams()}
}

// adaptRecordDone never returns a RecordDone for a notification: op-geth calls it with a typed nil
// output for one, which a nil check cannot catch.
func adaptRecordDone(msg gethrpc.RecordedMsg, done jsonrpc.RecordDone) gethrpc.RecordDone {
	if done == nil || msg.MsgIsNotification() {
		return nil
	}
	return func(ctx context.Context, _, output gethrpc.RecordedMsg) {
		resp := jsonrpc.Response{Result: output.MsgResult()}
		if err := output.MsgError(); err != nil {
			resp.Error = &jsonrpc.Error{Code: err.Code, Message: err.Message, Data: err.Data}
		}
		done(ctx, resp)
	}
}
