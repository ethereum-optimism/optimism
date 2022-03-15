package proxyd

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
)

func handleRPC(ctx context.Context, w http.ResponseWriter, r *http.Request, maxBodySize int64, doRequest func(context.Context, *RPCReq) (*RPCRes, bool)) {
	log.Info(
		"received RPC request",
		"req_id", GetReqID(ctx),
		"auth", GetAuthCtx(ctx),
		"user_agent", r.Header.Get("user-agent"),
	)

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		log.Error("error reading request body", "err", err)
		writeRPCError(ctx, w, nil, ErrInternal)
		return
	}
	RecordRequestPayloadSize(ctx, len(body))

	if IsBatch(body) {
		reqs, err := ParseBatchRPCReq(body)
		if err != nil {
			log.Error("error parsing batch RPC request", "err", err)
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
			writeRPCError(ctx, w, nil, ErrParseErr)
			return
		}

		if len(reqs) > MaxBatchRPCCalls {
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, ErrTooManyBatchRequests)
			writeRPCError(ctx, w, nil, ErrTooManyBatchRequests)
			return
		}

		if len(reqs) == 0 {
			writeRPCError(ctx, w, nil, ErrInvalidRequest("must specify at least one batch call"))
			return
		}

		batchRes := make([]*RPCRes, len(reqs))
		var batchContainsCached bool
		for i := 0; i < len(reqs); i++ {
			req, err := ParseRPCReq(reqs[i])
			if err != nil {
				log.Info("error parsing RPC call", "source", "rpc", "err", err)
				batchRes[i] = NewRPCErrorRes(nil, err)
				continue
			}

			var cached bool
			batchRes[i], cached = doRequest(ctx, req)
			if cached {
				batchContainsCached = true
			}
		}

		setCacheHeader(w, batchContainsCached)
		writeBatchRPCRes(ctx, w, batchRes)
		return
	}

	req, err := ParseRPCReq(body)
	if err != nil {
		log.Info("error parsing RPC call", "source", "rpc", "err", err)
		writeRPCError(ctx, w, nil, err)
		return
	}

	backendRes, cached := doRequest(ctx, req)
	setCacheHeader(w, cached)
	writeRPCRes(ctx, w, backendRes)
}

func handleWS(ctx context.Context, w http.ResponseWriter, r *http.Request, getProxier func() (*WSProxier, error)) {
	log.Info("received WS connection", "req_id", GetReqID(ctx))

	proxier, err := getProxier()
	if err != nil {
		return
	}

	activeClientWsConnsGauge.WithLabelValues(GetAuthCtx(ctx)).Inc()
	go func() {
		// Below call blocks so run it in a goroutine.
		if err := proxier.Proxy(ctx); err != nil {
			log.Error("error proxying websocket", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
		}
		activeClientWsConnsGauge.WithLabelValues(GetAuthCtx(ctx)).Dec()
	}()

	log.Info("accepted WS connection", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx))
}

func populateContext(w http.ResponseWriter, r *http.Request, authenticatedPaths map[string]string) context.Context {
	vars := mux.Vars(r)
	authorization := vars["authorization"]

	if authenticatedPaths == nil {
		// handle the edge case where auth is disabled
		// but someone sends in an auth key anyway
		if authorization != "" {
			log.Info("blocked authenticated request against unauthenticated proxy")
			httpResponseCodesTotal.WithLabelValues("404").Inc()
			w.WriteHeader(404)
			return nil
		}
		return context.WithValue(
			r.Context(),
			ContextKeyReqID, // nolint:staticcheck
			randStr(10),
		)
	}

	if authorization == "" || authenticatedPaths[authorization] == "" {
		log.Info("blocked unauthorized request", "authorization", authorization)
		httpResponseCodesTotal.WithLabelValues("401").Inc()
		w.WriteHeader(401)
		return nil
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		ipPort := strings.Split(r.RemoteAddr, ":")
		if len(ipPort) == 2 {
			xff = ipPort[0]
		}
	}

	ctx := context.WithValue(r.Context(), ContextKeyAuth, authenticatedPaths[authorization]) // nolint:staticcheck
	ctx = context.WithValue(ctx, ContextKeyXForwardedFor, xff)                               // nolint:staticcheck
	return context.WithValue(
		ctx,
		ContextKeyReqID, // nolint:staticcheck
		randStr(10),
	)
}
