package batcher

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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

func (tm *BitcoinTransactionManager) SendTransactionTest(data []byte) (*btcjson.TxRawResult, error) {
	privateKeyBytes, err := hex.DecodeString("2acec091f15620eaaa778a1186ed479e1df4e217f308ecc3383a6136b0a07e0a")
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	_, publicKey := btcec.PrivKeyFromBytes(privateKeyBytes)

	chunkedDataToPost := CreateChunks(520, data)

	pkScript, err := CreateBitcoinScript(chunkedDataToPost)
	if err != nil {
		return nil, fmt.Errorf("error creating Taproot script: %v", err)
	}

	// create a new basetapleaf
	baseTapLeafForScript := txscript.NewBaseTapLeaf(pkScript)

	// assemble tapscript tree
	tapScriptTree := txscript.AssembleTaprootScriptTree(baseTapLeafForScript)

	ctrlBlock := tapScriptTree.LeafMerkleProofs[0].ToControlBlock(
		publicKey,
	)

	tapScriptRootHash := tapScriptTree.RootNode.TapHash()

	// outputKey is the tweaked public key of the taproot output
	outputKey := txscript.ComputeTaprootOutputKey(
		publicKey, tapScriptRootHash[:],
	)

	outputKeyAddress, err := btcutil.NewAddressTaproot(outputKey.SerializeCompressed()[1:], &chaincfg.RegressionNetParams)
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	outputKeyAddressEncoded := outputKeyAddress.EncodeAddress()

	fmt.Println("Output Key Address: ", outputKeyAddressEncoded)

	//Create a new transaction

	txid, err := tm.client.SendToAddress(outputKeyAddress, 10000)
	if err != nil {
		log.Fatalf("Error sending coins: %v", err)
	}

	confirmation, err := waitForTransactionConfirmation(*tm.client, txid)
	if err != nil {
		log.Fatalf("Error waiting for transaction confirmation: %v", err)
	}

	txInString := confirmation.Hex
	txId := confirmation.Txid

	fmt.Println("Confirmation: ", txInString)
	fmt.Println("TxId: ", txId)

	p2trScript, err := PayToTaprootScript(outputKey)
	if err != nil {
		log.Fatalf("Failed to create P2TR script: %v", err)
	}

	// fmt.Println("P2TR script: \n", p2trScript)

	// fmt.Println("SCRIPT: \n", tapScriptTree.LeafMerkleProofs[0].TapLeaf.Script)

	err = txscript.VerifyTaprootLeafCommitment(
		&ctrlBlock, schnorr.SerializePubKey(outputKey),
		tapScriptTree.LeafMerkleProofs[0].TapLeaf.Script,
	)

	if err != nil {
		log.Fatalf("Failed to verify taproot leaf commitment: %v", err)
	} else {
		fmt.Println("Taproot leaf commitment verified")
	}

	ctrlBytes, err := ctrlBlock.ToBytes()
	if err != nil {
		log.Fatalf("Failed to convert control block to bytes: %v", err)
	}

	fmt.Println("ctrlBlock: ", hex.EncodeToString(ctrlBytes))

	testTx := wire.NewMsgTx(2)
	testTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: wire.OutPoint{
			Index: 0,
			Hash:  *txid,
		},
	})
	txOut := &wire.TxOut{
		Value: 10000, PkScript: p2trScript,
	}

	testTx.AddTxOut(txOut)

	prevFetcher := txscript.NewCannedPrevOutputFetcher(
		txOut.PkScript, txOut.Value,
	)
	sigHashes := txscript.NewTxSigHashes(testTx, prevFetcher)

	// sig, err := txscript.RawTxInTapscriptSignature(
	// 	testTx, sigHashes, 0, txOut.Value,
	// 	txOut.PkScript, baseTapLeafForScript, txscript.SigHashNone,
	// 	privateKey,
	// )
	if err != nil {
		log.Fatalf("Failed to create raw tx in tapscript signature: %v", err)
	}

	// Now that we have the sig, we'll make a valid witness
	// including the control block.
	ctrlBlockBytes, err := ctrlBlock.ToBytes()
	if err != nil {
		log.Fatalf("Failed to convert control block to bytes: %v", err)
	}
	txCopy := testTx.Copy()
	txCopy.TxIn[0].Witness = wire.TxWitness{
		pkScript, ctrlBlockBytes,
	}

	// Finally, ensure that the signature produced is valid.
	vm, err := txscript.NewEngine(
		txOut.PkScript, txCopy, 0, txscript.StandardVerifyFlags,
		nil, sigHashes, txOut.Value, prevFetcher,
	)
	if err != nil {
		log.Fatalf("Failed to create script engine: %v", err)
	}

	err = vm.Execute()
	if err != nil {
		log.Fatalf("Failed to execute script: %v", err)
	} else {
		log.Println("Script executed successfully")
	}

	fee := int64(5000) // Set the fee to 150,000 satoshis
	txCopy.TxOut[0].Value -= fee

	txHash, err := tm.client.SendRawTransaction(txCopy, true)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	} else {
		log.Println("Transaction sent: ", txHash.String())
	}

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
