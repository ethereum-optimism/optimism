package oracle

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type jsonreq struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      uint64        `json:"id"`
}

type jsonresp struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      uint64        `json:"id"`
	Result  AccountResult `json:"result"`
}

type jsonresps struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      uint64 `json:"id"`
	Result  string `json:"result"`
}

// Result structs for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
}

//var nodeUrl = "http://192.168.1.213:8545"
var nodeUrl = "https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161"

func GetProvedAccountBytes(blockNumber *big.Int, stateRoot common.Hash, addr common.Address) []byte {
	fmt.Println("ORACLE GetProvedAccountBytes:", blockNumber, stateRoot, addr)
	cachePath := fmt.Sprintf("data/accounts_%d_%s", blockNumber, addr)

	// read cache if we can
	{
		dat, err := ioutil.ReadFile(cachePath)
		if err == nil {
			return dat
		}
	}

	r := jsonreq{Jsonrpc: "2.0", Method: "eth_getProof", Id: 1}
	r.Params = make([]interface{}, 3)
	r.Params[0] = addr
	r.Params[1] = []common.Hash{}
	r.Params[2] = fmt.Sprintf("0x%x", blockNumber.Int64()-1)
	jsonData, _ := json.Marshal(r)
	resp, _ := http.Post(nodeUrl, "application/json", bytes.NewBuffer(jsonData))
	defer resp.Body.Close()
	jr := jsonresp{}
	json.NewDecoder(resp.Body).Decode(&jr)

	// TODO: check proof
	account := Account{
		Nonce:    uint64(jr.Result.Nonce),
		Balance:  jr.Result.Balance.ToInt(),
		Root:     jr.Result.StorageHash,
		CodeHash: jr.Result.CodeHash.Bytes(),
	}

	/*fmt.Println(string(jsonData))
	fmt.Println(resp)
	fmt.Println(jr)*/

	ret, _ := rlp.EncodeToBytes(account)
	os.WriteFile(cachePath, ret, 0644)
	return ret
}

func GetProvedCodeBytes(blockNumber *big.Int, addr common.Address, codehash common.Hash) []byte {
	fmt.Println("ORACLE GetProvedCodeBytes:", blockNumber, addr, codehash)
	cachePath := fmt.Sprintf("data/code_%s", codehash)

	// read cache if we can
	{
		dat, err := ioutil.ReadFile(cachePath)
		if err == nil {
			return dat
		}
	}

	r := jsonreq{Jsonrpc: "2.0", Method: "eth_getCode", Id: 1}
	r.Params = make([]interface{}, 2)
	r.Params[0] = addr
	r.Params[1] = fmt.Sprintf("0x%x", blockNumber.Int64()-1)
	jsonData, _ := json.Marshal(r)
	//fmt.Println(string(jsonData))
	resp, _ := http.Post(nodeUrl, "application/json", bytes.NewBuffer(jsonData))
	defer resp.Body.Close()

	/*tmp, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(tmp))*/

	jr := jsonresps{}
	json.NewDecoder(resp.Body).Decode(&jr)

	//fmt.Println(jr.Result)

	// curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getCode","params":["0xa94f5374fce5edbc8e2a8697c15331677e6ebf0b", "0x2"],"id":1}'

	ret, _ := hex.DecodeString(jr.Result[2:])
	//fmt.Println(ret)

	if crypto.Keccak256Hash(ret) != codehash {
		panic("wrong code hash")
	}

	os.WriteFile(cachePath, ret, 0644)
	return ret
}
