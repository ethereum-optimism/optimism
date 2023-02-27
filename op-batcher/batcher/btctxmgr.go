package batcher

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// TransactionManager wraps the simple txmgr package to make it easy to send & wait for transactions
type BitcoinTransactionManager struct {
	// Config
	recipientAddress btcutil.Address
	senderAddress    btcutil.Address
	// Outside world
	client *rpcclient.Client
	//signerFn  opcrypto.SignerFn
	//log       log.Logger
}

func NewBitcoinTransactionManager(recipientAddress btcutil.Address, senderAddress btcutil.Address, client *rpcclient.Client) *BitcoinTransactionManager {
	t := &BitcoinTransactionManager{
		recipientAddress: recipientAddress,
		senderAddress:    senderAddress,
		client:           client,
	}
	return t
}

func (tm *BitcoinTransactionManager) SendTransaction(amount int64, fee int64, data []byte) (*btcjson.TxRawResult, error) {

	// Create an empty transaction with the current Bitcoin transaction version.
	tx := wire.NewMsgTx(wire.TxVersion)

	// Adds coinbase input to the transaction with no previous output (we instanciate the thing lmao)
	prevOut := wire.NewOutPoint(&chainhash.Hash{}, 0)
	txIn := wire.NewTxIn(prevOut, nil, nil)
	tx.AddTxIn(txIn)

	// add output to the transaction designating the recipient address from the bitcoin tx manager object
	pkScript, err := txscript.PayToAddrScript(tm.recipientAddress)
	if err != nil {
		return nil, fmt.Errorf("error creating pkScript: %v", err)
	}
	txOut := wire.NewTxOut(amount, pkScript)
	tx.AddTxOut(txOut)

	chunkedDataToPost := CreateChunks(520, data)
	script, err := CreateBitcoinScript(chunkedDataToPost)
	if err != nil {
		return nil, fmt.Errorf("error creating Taproot script: %v", err)
	}

	// place teh script in the output
	output := wire.NewTxOut(0, script)
	output.PkScript, err = txscript.PayToAddrScript(tm.recipientAddress)
	if err != nil {
		return nil, fmt.Errorf("error creating Taproot pkScript: %v", err)
	}
	tx.AddTxOut(output)

	// Sign transaction
	// fetch previous output from transaction
	// prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(script, 0)

	// get signatures hash
	//_ = txscript.NewTxSigHashes(tx, prevOutputFetcher)
	privKey, err := tm.client.DumpPrivKey(tm.senderAddress)
	if err != nil {
		return nil, fmt.Errorf("error getting private key: %v", err)
	}
	wif, err := btcutil.DecodeWIF(privKey.String())
	if err != nil {
		return nil, fmt.Errorf("error decoding WIF: %v", err)
	}
	sig, err := txscript.RawTxInSignature(tx, 0, pkScript, txscript.SigHashAll, wif.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("error signing transaction: %v", err)
	}
	sigScript, err := txscript.NewScriptBuilder().AddData(sig).AddData(wif.SerializePubKey()).Script()
	if err != nil {
		return nil, fmt.Errorf("error creating sigScript: %v", err)
	}
	tx.TxIn[0].SignatureScript = sigScript

	// Send transaction
	txHash, err := tm.client.SendRawTransaction(tx, false)
	if err != nil {
		return nil, fmt.Errorf("error sending transaction: %v", err)
	}

	// Wait for transaction confirmation
	confirmations := make(chan int64)
	confirmedTxResult := make(chan *btcjson.TxRawResult)
	go func() {
		for {
			tx, err := tm.client.GetRawTransactionVerbose(txHash)
			if err != nil {
				log.Printf("error getting transaction: %v", err)
				continue
			}
			confirmations <- int64(tx.Confirmations)
			confirmedTxResult <- tx
			if tx.Confirmations > 0 {
				// I'm just going to ignore that bitcoin has reorgs thanks
				close(confirmedTxResult)
				close(confirmations)
				return
			}
			// retry in channel every minute
			time.Sleep(60 * time.Second)
		}
	}()

	// Wait for confirmation or timeout
	select {
	case confirmations := <-confirmations:
		if confirmations == 0 {
			return nil, errors.New("transaction not confirmed")
		}
		confirmedTxResult := <-confirmedTxResult

		return confirmedTxResult, nil
	case <-time.After(20 * time.Minute):
		return nil, errors.New("timeout waiting for confirmation")
	}
}
