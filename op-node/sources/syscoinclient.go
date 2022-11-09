package sources

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime/debug"
	"time"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)
// JSONMarshalerV2 is used for marshalling requests to newer Syscoin Type RPC interfaces
type JSONMarshalerV2 struct{}

// Marshal converts struct passed by parameter to JSON
func (JSONMarshalerV2) Marshal(v interface{}) ([]byte, error) {
	d, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return d, nil
}
// SyscoinRPC is an interface to JSON-RPC syscoind service.
type SyscoinRPC struct {
	client       http.Client
	rpcURL       string
	user         string
	password     string
	RPCMarshaler JSONMarshalerV2
}
type SyscoinClient struct {
	client *SyscoinRPC
}
func NewSyscoinClient() SyscoinClient {
	transport := &http.Transport{
		Dial:                (&net.Dialer{KeepAlive: 600 * time.Second}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100, // necessary to not to deplete ports
	}

	s := &SyscoinRPC{
		client:       http.Client{Timeout: time.Duration(25) * time.Second, Transport: transport},
		rpcURL:       "http://l1:18370/wallet/wallet",
		user:         "u",
		password:     "p",
		RPCMarshaler: JSONMarshalerV2{},
	}

	return SyscoinClient{s}
}
// RPCError defines rpc error returned by backend
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
func (e *RPCError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}
func safeDecodeResponse(body io.ReadCloser, res interface{}) (err error) {
	var data []byte
	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			if len(data) > 0 && len(data) < 2048 {
				err = errors.New(fmt.Sprintf("Error %v", string(data)))
			} else {
				err = errors.New("Internal error")
			}
		}
	}()
	data, err = ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	log.Info("CreateBlob", "data", data)
	return json.Unmarshal(data, &res)
}

// Call calls Backend RPC interface, using RPCMarshaler interface to marshall the request
func (s *SyscoinClient) Call(req interface{}, res interface{}) error {
	httpData, err := s.client.RPCMarshaler.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequest("POST", s.client.rpcURL, bytes.NewBuffer(httpData))
	if err != nil {
		return err
	}
	httpReq.SetBasicAuth(s.client.user, s.client.password)
	httpRes, err := s.client.client.Do(httpReq)
	// in some cases the httpRes can contain data even if it returns error
	// see http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		return err
	}
	// if server returns HTTP error code it might not return json with response
	// handle both cases
	if httpRes.StatusCode != 200 {
		err = safeDecodeResponse(httpRes.Body, &res)
		if err != nil {
			return errors.New(fmt.Sprintf("Error %v %v", httpRes.Status, err))
		}
		return nil
	}
	return safeDecodeResponse(httpRes.Body, &res)
}

func (s *SyscoinClient) CreateBlob(data []byte) (common.Hash, error) {
	type ResCreateBlob struct {
		Error  *RPCError `json:"error"`
		Result struct {
			VersionHash string `json:"versionhash"`
		} `json:"result"`
	}

	res := ResCreateBlob{}
	type CmdCreateBlob struct {
		Method string `json:"method"`
		Params struct {
			Data string `json:"data"`
		} `json:"params"`
	}
	req := CmdCreateBlob{Method: "syscoincreatenevmblob"}
	req.Params.Data = string(data)
	err := s.Call(&req, &res)
	if err != nil {
		return common.Hash{}, err
	}
	if res.Error != nil {
		return common.Hash{}, res.Error
	}
	return common.HexToHash(res.Result.VersionHash), err
}

func (s *SyscoinClient) IsBlobConfirmed(vh common.Hash) (bool, error) {
	type ResGetBlobMPT struct {
		Error  *RPCError `json:"error"`
		Result struct {
			MPT int64 `json:"mpt"`
		} `json:"result"`
	}
	res := ResGetBlobMPT{}
	type CmdGetBlobMPT struct {
		Method string `json:"method"`
		Params struct {
			VersionHash string `json:"versionhash_or_txid"`
		} `json:"params"`
	}
	req := CmdGetBlobMPT{Method: "getnevmblobdata"}
	req.Params.VersionHash = vh.String()[2:]
	err := s.Call(&req, &res)
	if err != nil {
		return false, err
	}
	if res.Error != nil {
		return false, res.Error
	}
	return res.Result.MPT > 0, err
}

func (s *SyscoinClient) GetBlobFromRPC(vh common.Hash) (string, error) {
	type ResGetBlobData struct {
		Error  *RPCError `json:"error"`
		Result struct {
			Data string `json:"data"`
		} `json:"result"`
	}
	res := ResGetBlobData{}
	type CmdGetBlobData struct {
		Method string `json:"method"`
		Params struct {
			VersionHash string `json:"versionhash_or_txid"`
			Verbose   bool   `json:"getdata"`
		} `json:"params"`
	}
	req := CmdGetBlobData{Method: "getnevmblobdata"}
	req.Params.VersionHash = vh.String()[2:]
	req.Params.Verbose = true
	err := s.Call(&req, &res)
	if err != nil {
		return "", err
	}
	if res.Error != nil {
		return "", res.Error
	}
	return res.Result.Data, err
}

func (s *SyscoinClient) GetBlobFromCloud(vh common.Hash) (string, error) {
	return "", nil
}