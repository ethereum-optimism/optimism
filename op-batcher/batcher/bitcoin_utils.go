package batcher

import (
	"log"
	"math"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
)

func CreateChunks(chunkSize uint64, calldata []byte) [][]byte {
	chunkArray := make([][]byte, 0)
	for i := uint64(0); i < uint64(len(calldata)); i += chunkSize {
		chunk := make([]byte, 0)
		index := math.Min(float64(len(calldata)), float64(i+chunkSize))
		calldataToAppend := calldata[i:int(index)]
		chunk = append(chunk, calldataToAppend...)
		chunkArray = append(chunkArray, chunk)
	}
	return chunkArray
}

func CreateBitcoinScript(calldataChunks [][]byte) ([]byte, error) {
	tapscript := txscript.NewScriptBuilder()
	//tapscript.AddOp(txscript.OP_TRUE)
	tapscript.AddOp(txscript.OP_FALSE)
	tapscript.AddOp(txscript.OP_IF)
	// for loop appending elements of calldata to tapscript with AddData
	for _, calldata := range calldataChunks {
		tapscript.AddData(calldata)
	}
	tapscript.AddOp(txscript.OP_ENDIF)

	return tapscript.Script()
}

func PayToTaprootScript(taprootKey *btcec.PublicKey) ([]byte, error) {
	return txscript.NewScriptBuilder().
		AddOp(txscript.OP_1).
		AddData(schnorr.SerializePubKey(taprootKey)).
		Script()
}

func ConnectToClient() (*rpcclient.Client, error) {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host:         "regtest.dctrl.wtf",
		User:         "test",
		Pass:         "test",
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   false,
		Params:       "regtest",
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func waitForTransactionConfirmation(client rpcclient.Client, txHash *chainhash.Hash) (*btcjson.TxRawResult, error) {
	for {
		time.Sleep(20 * time.Second)
		tx, err := client.GetRawTransactionVerbose(txHash)
		if err != nil {
			log.Printf("error getting transaction: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if tx.Confirmations > 0 {
			return tx, nil
		}
		// retry every 10 seconds
		time.Sleep(10 * time.Second)
	}
}

func getCurrBTCBlockHeight(client rpcclient.Client) (int32, error) {

	info, err := client.GetBlockChainInfo()
	if err != nil {
		return 0, err
	}

	return info.Blocks, err
}

func getBTCBlockHeaderForHash(client rpcclient.Client, blockHash string) (*btcjson.GetBlockHeaderVerboseResult, error) {
	log.Println("In getBTCBlockHeaderForHash, NewHashFromStr")
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return nil, err
	}

	log.Println("In getBTCBlockHeaderForHash, GetBlockHeaderVerbose")
	info, err := client.GetBlockHeaderVerbose(hash)
	if err != nil {
		return nil, err
	}

	return info, err
}

func getBTCBlockHeaderForHeight(client rpcclient.Client, blockHeight int32) (*btcjson.GetBlockHeaderVerboseResult, error) {

	hash, err := client.GetBlockHash(int64(blockHeight))
	if err != nil {
		return nil, err
	}
	info, err := client.GetBlockHeaderVerbose(hash)
	if err != nil {
		return nil, err
	}

	return info, err
}
