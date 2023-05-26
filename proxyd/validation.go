package proxyd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

const (
	validationMethod = "validator_isAllowed"
)

type RPCValidator interface {
	IsAllowed(ctx context.Context, req *RPCReq) (bool, string)
}

type externalRPCValidator struct {
	client *http.Client
	rpcURL string

	failOpen bool
}

func newExternalRPCValidator(rpcURL string, failOpen bool, timeout time.Duration) externalRPCValidator {
	return externalRPCValidator{
		client:   &http.Client{Timeout: timeout},
		rpcURL:   rpcURL,
		failOpen: failOpen,
	}
}

func (v externalRPCValidator) IsAllowed(ctx context.Context, req *RPCReq) (bool, string) {
	rpcRes, err := v.doRequest(ctx, req)
	if err != nil {
		log.Warn(
			"error validating request via external validator",
			"validation_url", v.rpcURL,
			"req_id", GetReqID(ctx),
			"auth", GetAuthCtx(ctx),
			"err", err,
		)
		return v.failOpen, "unable to validate RPC request"
	}

	jsonRes, ok := rpcRes.Result.(bool)
	if !ok {
		log.Warn(
			"invalid response from external validator",
			"validation_url", v.rpcURL,
			"req_id", GetReqID(ctx),
			"auth", GetAuthCtx(ctx),
			"err", err,
		)
		return v.failOpen, "unable to validate RPC request"
	}
	return jsonRes, ""
}

func (v externalRPCValidator) doRequest(ctx context.Context, req *RPCReq) (*RPCRes, error) {
	validationReq := RPCReq{
		JSONRPC: JSONRPCVersion,
		Method:  validationMethod,
		ID:      []byte("1"),
		Params:  mustMarshalJSON(req),
	}

	body := mustMarshalJSON(validationReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", v.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("content-type", "application/json")
	httpRes, err := v.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if httpRes.StatusCode != 200 {
		return nil, fmt.Errorf("validation response code %d", httpRes.StatusCode)
	}

	defer httpRes.Body.Close()
	resB, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, wrapErr(err, "error reading validation response body")
	}

	var rpcRes *RPCRes
	if err := json.Unmarshal(resB, &rpcRes); err != nil {
		return nil, wrapErr(err, "invalid backend response")
	}

	return rpcRes, nil
}

type basicRPCValidator struct{}

func (b basicRPCValidator) IsAllowed(ctx context.Context, req *RPCReq) (bool, string) {
	if req.JSONRPC != JSONRPCVersion {
		return false, "invalid JSON-RPC version"
	}

	if req.Method == "" {
		return false, "no method specified"
	}

	if !IsValidID(req.ID) {
		return false, "invalid ID"
	}

	return true, ""
}
