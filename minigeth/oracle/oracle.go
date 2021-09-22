package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

type jsonreq struct {
	Jsonrpc string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  [3]interface{} `json:"params"`
	Id      uint64         `json:"id"`
}

type jsonresp struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      uint64        `json:"id"`
	Result  AccountResult `json:"result"`
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

func GetProvedAccountBytes(blockNumber *big.Int, stateRoot common.Hash, addr common.Address) []byte {
	fmt.Println("ORACLE GetProvedAccountBytes:", blockNumber, stateRoot, addr)

	r := jsonreq{Jsonrpc: "2.0", Method: "eth_getProof", Id: 1}
	r.Params[0] = addr
	r.Params[1] = []common.Hash{}
	r.Params[2] = fmt.Sprintf("0x%x", blockNumber.Int64()-1)

	jsonData, _ := json.Marshal(r)
	resp, _ := http.Post("http://192.168.1.213:8545", "application/json", bytes.NewBuffer(jsonData))
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
	return ret
	//return []byte("12")
}
