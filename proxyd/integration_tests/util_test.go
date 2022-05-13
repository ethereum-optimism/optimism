package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type ProxydClient struct {
	url string
}

func NewProxydClient(url string) *ProxydClient {
	return &ProxydClient{url: url}
}

func (p *ProxydClient) SendRPC(method string, params []interface{}) ([]byte, int, error) {
	rpcReq := NewRPCReq("999", method, params)
	body, err := json.Marshal(rpcReq)
	if err != nil {
		panic(err)
	}
	return p.SendRequest(body)
}

func (p *ProxydClient) SendBatchRPC(reqs ...*proxyd.RPCReq) ([]byte, int, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		panic(err)
	}
	return p.SendRequest(body)
}

func (p *ProxydClient) SendRequest(body []byte) ([]byte, int, error) {
	res, err := http.Post(p.url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, -1, err
	}
	defer res.Body.Close()
	code := res.StatusCode
	resBody, err := ioutil.ReadAll(res.Body)
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

func InitLogger() {
	log.Root().SetHandler(
		log.LvlFilterHandler(log.LvlDebug,
			log.StreamHandler(
				os.Stdout,
				log.TerminalFormat(false),
			)),
	)
}
