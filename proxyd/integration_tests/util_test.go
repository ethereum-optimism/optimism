package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/gorilla/websocket"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

type ProxydHTTPClient struct {
	url     string
	headers http.Header
}

func NewProxydClient(url string) *ProxydHTTPClient {
	return NewProxydClientWithHeaders(url, make(http.Header))
}

func NewProxydClientWithHeaders(url string, headers http.Header) *ProxydHTTPClient {
	clonedHeaders := headers.Clone()
	clonedHeaders.Set("Content-Type", "application/json")
	return &ProxydHTTPClient{
		url:     url,
		headers: clonedHeaders,
	}
}

func (p *ProxydHTTPClient) SendRPC(method string, params []interface{}) ([]byte, int, error) {
	rpcReq := NewRPCReq("999", method, params)
	body, err := json.Marshal(rpcReq)
	if err != nil {
		panic(err)
	}
	return p.SendRequest(body)
}

func (p *ProxydHTTPClient) SendBatchRPC(reqs ...*proxyd.RPCReq) ([]byte, int, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		panic(err)
	}
	return p.SendRequest(body)
}

func (p *ProxydHTTPClient) SendRequest(body []byte) ([]byte, int, error) {
	req, err := http.NewRequest("POST", p.url, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header = p.headers

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer res.Body.Close()
	code := res.StatusCode
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	return resBody, code, nil
}

func RequireEqualJSON(t *testing.T, expected []byte, actual []byte) {
	expJSON := canonicalizeJSON(t, expected)
	actJSON := canonicalizeJSON(t, actual)
	require.Equal(t, string(expJSON), string(actJSON))
}

func canonicalizeJSON(t *testing.T, in []byte) []byte {
	var any interface{}
	if in[0] == '[' {
		any = make([]interface{}, 0)
	} else {
		any = make(map[string]interface{})
	}

	err := json.Unmarshal(in, &any)
	require.NoError(t, err)
	out, err := json.Marshal(any)
	require.NoError(t, err)
	return out
}

func ReadConfig(name string) *proxyd.Config {
	config := new(proxyd.Config)
	_, err := toml.DecodeFile(fmt.Sprintf("testdata/%s.toml", name), config)
	if err != nil {
		panic(err)
	}
	return config
}

func NewRPCReq(id string, method string, params []interface{}) *proxyd.RPCReq {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}

	return &proxyd.RPCReq{
		JSONRPC: proxyd.JSONRPCVersion,
		Method:  method,
		Params:  jsonParams,
		ID:      []byte(id),
	}
}

type ProxydWSClient struct {
	conn    *websocket.Conn
	msgCB   ProxydWSClientOnMessage
	closeCB ProxydWSClientOnClose
}

type WSMessage struct {
	Type int
	Body []byte
}

type ProxydWSClientOnMessage func(msgType int, data []byte)
type ProxydWSClientOnClose func(err error)

func NewProxydWSClient(
	url string,
	msgCB ProxydWSClientOnMessage,
	closeCB ProxydWSClientOnClose,
) (*ProxydWSClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
	if err != nil {
		return nil, err
	}

	c := &ProxydWSClient{
		conn:    conn,
		msgCB:   msgCB,
		closeCB: closeCB,
	}
	go c.readPump()
	return c, nil
}

func (h *ProxydWSClient) readPump() {
	for {
		mType, msg, err := h.conn.ReadMessage()
		if err != nil {
			if h.closeCB != nil {
				h.closeCB(err)
			}
			return
		}
		if h.msgCB != nil {
			h.msgCB(mType, msg)
		}
	}
}

func (h *ProxydWSClient) HardClose() {
	h.conn.Close()
}

func (h *ProxydWSClient) SoftClose() error {
	return h.WriteMessage(websocket.CloseMessage, nil)
}

func (h *ProxydWSClient) WriteMessage(msgType int, msg []byte) error {
	return h.conn.WriteMessage(msgType, msg)
}

func (h *ProxydWSClient) WriteControlMessage(msgType int, msg []byte) error {
	return h.conn.WriteControl(msgType, msg, time.Now().Add(time.Minute))
}

func InitLogger() {
	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlDebug,
			log.StreamHandler(
				os.Stdout,
				log.TerminalFormat(false),
			)),
	)
}
