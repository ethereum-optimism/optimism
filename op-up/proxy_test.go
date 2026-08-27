package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

const elProxyTestOrigin = "http://127.0.0.1:4000"

type elProxyTestAPI struct{}

type elProxyTestClient struct {
	*gethrpc.Client
}

func (c *elProxyTestClient) Subscribe(ctx context.Context, namespace string, channel any,
	args ...any,
) (ethereum.Subscription, error) {
	return c.Client.Subscribe(ctx, namespace, channel, args...)
}

func (elProxyTestAPI) Echo(value string) string {
	return value
}

func newTestELProxyHandler(t *testing.T) (http.Handler, *bytes.Buffer) {
	t.Helper()
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("test", elProxyTestAPI{}))
	client := gethrpc.DialInProc(server)
	t.Cleanup(server.Stop)
	t.Cleanup(client.Close)
	logs := new(bytes.Buffer)
	return newELProxyHandler(logs, &elProxyTestClient{Client: client}, elProxyTestOrigin), logs
}

func TestELProxyCORS(t *testing.T) {
	handler, _ := newTestELProxyHandler(t)

	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", elProxyTestOrigin)
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, elProxyTestOrigin, response.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "POST, OPTIONS", response.Header().Get("Access-Control-Allow-Methods"))
	require.Contains(t, response.Header().Values("Vary"), "Origin")

	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["hello"]}`,
	))
	request.Header.Set("Origin", "https://example.invalid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Empty(t, response.Header().Get("Access-Control-Allow-Origin"))
}

func TestELProxySingleRequest(t *testing.T) {
	handler, logs := newTestELProxyHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(
		`{"jsonrpc":"2.0","id":7,"method":"test_echo","params":["hello"]}`,
	))
	request.Header.Set("Origin", elProxyTestOrigin)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, elProxyTestOrigin, response.Header().Get("Access-Control-Allow-Origin"))
	var rpcResponse elProxyResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&rpcResponse))
	require.Equal(t, "2.0", rpcResponse.JSONRPC)
	require.JSONEq(t, `7`, string(rpcResponse.ID))
	require.JSONEq(t, `"hello"`, string(rpcResponse.Result))
	require.Nil(t, rpcResponse.Error)
	require.Equal(t, "test_echo\n", logs.String())
}

func TestELProxyBatchRequest(t *testing.T) {
	handler, logs := newTestELProxyHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`[
		{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["public"]},
		{"jsonrpc":"2.0","id":"second","method":"test_echo","params":["private"]}
	]`))
	request.Header.Set("Origin", elProxyTestOrigin)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	var rpcResponses []elProxyResponse
	require.NoError(t, json.NewDecoder(response.Body).Decode(&rpcResponses))
	require.Len(t, rpcResponses, 2)
	require.JSONEq(t, `1`, string(rpcResponses[0].ID))
	require.JSONEq(t, `"public"`, string(rpcResponses[0].Result))
	require.Nil(t, rpcResponses[0].Error)
	require.JSONEq(t, `"second"`, string(rpcResponses[1].ID))
	require.JSONEq(t, `"private"`, string(rpcResponses[1].Result))
	require.Nil(t, rpcResponses[1].Error)
	require.Equal(t, "test_echo\ntest_echo\n", logs.String())
}

func TestELProxyRejectsMalformedRequests(t *testing.T) {
	handler, _ := newTestELProxyHandler(t)
	for _, body := range []string{
		``,
		`{}`,
		`[]`,
		`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":"hello"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		require.Equal(t, http.StatusBadRequest, response.Code, "body: %s", body)
	}
}
