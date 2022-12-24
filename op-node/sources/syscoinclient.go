package sources

import (
	"context"
	"bytes"
	"encoding/json"
	"encoding/hex"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"runtime/debug"
	"time"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
func NewSyscoinClient(sysdesc string, sysdescinternal string) (SyscoinClient, error) {
	transport := &http.Transport{
		Dial:                (&net.Dialer{KeepAlive: 600 * time.Second}).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100, // necessary to not to deplete ports
	}
	s := &SyscoinRPC{
		client:       http.Client{Timeout: time.Duration(600) * time.Second, Transport: transport},
		rpcURL:       "http://l1:18370",
		user:         "u",
		password:     "p",
		RPCMarshaler: JSONMarshalerV2{},
	}
	client := SyscoinClient{s}
	if len(sysdesc) > 0 && len(sysdescinternal) > 0 {
		walletName := "wallet"
		err := client.CreateOrLoadWallet(walletName)
		if err != nil {
			return client, err
		}
		err = client.ImportDescriptor(sysdesc)
		if err != nil {
			return client, err
		}
		err = client.ImportDescriptor(sysdescinternal)
		if err != nil {
			return client, err
		}
		client.client.rpcURL += "/wallet/" + walletName
	}
	return client, nil
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
			VH string `json:"versionhash"`
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
	req.Params.Data = hex.EncodeToString(data)
	err := s.Call(&req, &res)
	if err != nil {
		return common.Hash{}, err
	}
	if res.Error != nil {
		return common.Hash{}, res.Error
	}
	return common.HexToHash(res.Result.VH), err
}
func (s *SyscoinClient) CreateOrLoadWallet(walletName string) (error) {
	type ResCreateWallet struct {
		Error  *RPCError `json:"error"`
		Result struct {
			Warning string `json:"warning"`
		} `json:"result"`
	}

	res := ResCreateWallet{}
	type CmdCreateWallet struct {
		Method string `json:"method"`
		Params struct {
			WalletName string `json:"wallet_name"`
		} `json:"params"`
	}
	req := CmdCreateWallet{Method: "createwallet"}
	req.Params.WalletName = walletName
	err := s.Call(&req, &res)
	if err != nil {
		return err
	}
	// might actually be created already so just load it
	if res.Error != nil {
		type ResLoadWallet struct {
			Error  *RPCError `json:"error"`
			Result struct {
				Warning string `json:"warning"`
			} `json:"result"`
		}

		res := ResLoadWallet{}
		type CmdLoadWallet struct {
			Method string `json:"method"`
			Params struct {
				WalletName string `json:"filename"`
			} `json:"params"`
		}
		req := CmdLoadWallet{Method: "loadwallet"}
		req.Params.WalletName = walletName
		err = s.Call(&req, &res)
		if err != nil {
			return err
		}
		if res.Error != nil {
			return res.Error
		}
	}
	if len(res.Result.Warning) > 0 {
		return errors.New(res.Result.Warning)
	}
	return nil
}
func (s *SyscoinClient) ImportDescriptor(descriptor string) (error) {
	type ResImportDescriptor struct {
		Error  *RPCError `json:"error"`
	}

	res := ResImportDescriptor{}
	type CmdImportDescriptor struct {
		Method string `json:"method"`
		Params struct {
			Desc interface{} `json:"requests"`
		} `json:"params"`
	}
	req := CmdImportDescriptor{Method: "importdescriptors"}
	descBytes := []byte(descriptor)
	err := json.Unmarshal(descBytes, &req.Params.Desc)
	if err != nil {
		return err
	}
	err = s.Call(&req, &res)
	if err != nil {
		return err
	}
	if res.Error != nil {
		return res.Error
	}
	return nil
}
// SYSCOIN used to get blob confirmation by checking block number then tx receipt below to get block height of blob confirmation
func (s *SyscoinClient) BlockNumber(ctx context.Context) (uint64, error) {
	type ResGetBlockNumber struct {
		Error  *RPCError `json:"error"`
		BlockNumber uint64 `json:"result"`
	}
	res := ResGetBlockNumber{}
	type CmdGetBlockNumber struct {
		Method string `json:"method"`
		Params struct {
		} `json:"params"`
	}
	req := CmdGetBlockNumber{Method: "getblockcount"}
	err := s.Call(&req, &res)
	if err != nil {
		return 0, err
	}
	if res.Error != nil {
		return 0, res.Error
	}
	return res.BlockNumber, err
}
// SYSCOIN used to get blob receipt
func (s *SyscoinClient) TransactionReceipt(ctx context.Context, vh common.Hash) (*types.Receipt, error) {
	type ResGetBlobReceipt struct {
		Error  *RPCError `json:"error"`
		Result struct {
			MPT int64 `json:"mpt"`
		} `json:"result"`
	}
	res := ResGetBlobReceipt{}
	type CmdGetBlobReceipt struct {
		Method string `json:"method"`
		Params struct {
			TXID string `json:"versionhash_or_txid"`
		} `json:"params"`
	}
	req := CmdGetBlobReceipt{Method: "getnevmblobdata"}
	req.Params.TXID = vh.String()[2:]
	err := s.Call(&req, &res)
	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, res.Error
	}
	receipt := types.Receipt{}
	if res.Result.MPT > 0 {
		// store VH in TxHash used by driver to put into BatchInbox
		receipt = types.Receipt{
			TxHash:      vh,
			// store MPT in BlockNumber to be used in caller
			BlockNumber: big.NewInt(res.Result.MPT),
			Status:      types.ReceiptStatusSuccessful,
		}
	}
	return &receipt, err
}

func (s *SyscoinClient) GetBlobFromRPC(vh common.Hash) ([]byte, error) {
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
		return nil, err
	}
	if res.Error != nil {
		return nil, res.Error
	}
	data, err := hex.DecodeString(res.Result.Data)
	if err != nil {
		return nil, err
	}
	return data, err
}

func (s *SyscoinClient) GetBlobFromCloud(vh common.Hash) ([]byte, error) {
	url := "http://poda.tanenbaum.io/vh/" + vh.String()[2:]
	var res *http.Response
	var err error
	// try 4 times incase of timeout or reset/hanging socket with 5+i second expiry each attempt
	for i := 0; i < 4; i++ {
		client := http.Client{
			Timeout: (5 + time.Duration(i)) * time.Second,
		}
		res, err = client.Get(url)
		if err != nil {
			continue
		} else {
			err = nil
			break
		}
	}
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() // we need to close the connection
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
		return nil, err
	}
	txBytes, err := hex.DecodeString(string(body))
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}