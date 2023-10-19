package httputil

import (
	"context"
	"net/http"
)

// HTTPInterceptorFunc is a function expression of HTTPInterceptor for convenience
type HTTPInterceptorFunc func(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error)

func (fn HTTPInterceptorFunc) Intercept(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
	return fn(req, inner)
}

// HTTPInterceptor intercepts HTTP requests, allows the interceptor to run the inner round-tripper,
// and then return the response.
type HTTPInterceptor interface {
	Intercept(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error)
}

type interceptorChain struct {
	insp  HTTPInterceptor
	inner http.RoundTripper
}

func (ic interceptorChain) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	return ic.insp.Intercept(req, ic.inner)
}

type interceptorContextValue struct {
	insp  HTTPInterceptor
	inner HTTPInterceptor
}

func (ic interceptorContextValue) Intercept(req *http.Request, inner http.RoundTripper) (resp *http.Response, err error) {
	if ic.inner == nil {
		return ic.insp.Intercept(req, inner)
	}
	v := interceptorChain{
		insp:  ic.inner,
		inner: inner,
	}
	return ic.insp.Intercept(req, &v)
}

type interceptorKeyType struct{}

var interceptorKey = interceptorKeyType{}

// NewInterceptorContext adds an HTTPInterceptor to the context.
// The http interceptor of the parent context, of any, will run as the inner http.RoundTripper to the interceptor.
// I.e. HTTP interceptors can be chained together.
func NewInterceptorContext(ctx context.Context, interceptor HTTPInterceptor) context.Context {
	val := interceptorContextValue{
		insp:  interceptor,
		inner: InterceptorFromContext(ctx),
	}
	return context.WithValue(ctx, interceptorKey, val)
}

func InterceptorFromContext(ctx context.Context) HTTPInterceptor {
	insp := ctx.Value(interceptorKey)
	if insp == nil {
		return nil
	}
	return insp.(HTTPInterceptor)
}

// InterceptorRoundTripper wraps a http.RoundTripper and intercepts the HTTP requests,
// and passes them through the HTTPInterceptor attached to the request context, if any.
type InterceptorRoundTripper struct {
	Inner http.RoundTripper
}

func (ir InterceptorRoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	v := req.Context().Value(interceptorKey)
	if v == nil {
		return ir.Inner.RoundTrip(req)
	} else {
		interceptor := v.(interceptorContextValue)
		return interceptor.Intercept(req, ir.Inner)
	}
}
