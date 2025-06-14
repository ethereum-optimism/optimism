package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/event"
)

type requestContextKeyType struct{}

var requestContextKey = requestContextKeyType{}

func RequestFromContext(ctx context.Context) *Request {
	v := ctx.Value(requestContextKey)
	if v == nil {
		return nil
	}
	return v.(*Request)
}

func WithRequest(ctx context.Context, cond event.Matcher) (context.Context, *Request) {
	ctx, cancel := context.WithCancelCause(ctx)
	req := &Request{
		cond:   cond,
		ctx:    ctx,
		cancel: cancel,
	}
	return context.WithValue(ctx, requestContextKey, req), req
}

type Request struct {
	// cond is the completion condition: when we have seen this event the request is done.
	// The condition is only checked for events that have this request attached to them.
	// All responses have the contexts (or wrappers of) the context of the request.
	// This is set to nil when the condition is met.
	cond event.Matcher

	// ctx for the duration of the request
	ctx context.Context

	// cancel is called when the condition completes
	cancel context.CancelCauseFunc
}
