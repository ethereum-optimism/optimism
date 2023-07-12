package provider

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/tls"
	signer "github.com/ethereum-optimism/optimism/op-signer/client"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Heartbeat poll for expected transactions
func (p *Provider) Heartbeat(ctx context.Context) {
	log.Debug("heartbeat", "provider", p.name)

	ethClient, err := p.dial(ctx)
	if err != nil {
		log.Error("cant dial to provider", "provider", p.name, "url", p.config.URL, "err", err)
	}

	nonce, err := p.nonce(ctx, ethClient)
	if err != nil {
		log.Error("cant get nounce", "provider", p.name, "err", err)
	}

	tx := p.createTx(nonce)

	signedTx, err := p.sign(ctx, tx)
	if err != nil {
		log.Error("cant sign tx", "tx", tx, "err", err)
	}

	err = ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Error("cant send transaction", "provider", p.name, "err", err)
	}
	txHash := signedTx.Hash()
	log.Info("transaction sent", "hash", txHash.Hex())

	sentAt := time.Now()
	var receipt *types.Receipt
	attempt := 0
	for receipt == nil {
		if time.Since(sentAt) >= time.Duration(p.config.ReceiptRetrievalTimeout) {
			log.Error("receipt retrieval timedout", "provider", p.name, "hash", "ellapsed", time.Since(sentAt))
			break
		}
		time.Sleep(time.Duration(p.config.ReceiptRetrievalInterval))
		if attempt%10 == 0 {
			log.Debug("checking for receipt...", "attempt", attempt, "ellapsed", time.Since(sentAt))
		}
		receipt, err = ethClient.TransactionReceipt(ctx, txHash)
		if err != nil && !errors.Is(err, ethereum.NotFound) {
			log.Error("cant get receipt for transaction", "provider", p.name, "hash", txHash.Hex(), "err", err)
			break
		}
		attempt++
	}
	roundtrip := time.Since(sentAt)

	log.Info("got transaction receipt", "hash", txHash.Hex(),
		"roundtrip", roundtrip,
		"blockNumber", receipt.BlockNumber,
		"blockHash", receipt.BlockHash,
		"gasUsed", receipt.GasUsed)
}

func (p *Provider) dial(ctx context.Context) (*ethclient.Client, error) {
	return ethclient.Dial(p.config.URL)
}

func (p *Provider) createTx(nonce uint64) *types.Transaction {
	toAddress := common.HexToAddress(p.walletConfig.Address)
	var data []byte
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   &p.walletConfig.ChainID,
		Nonce:     nonce,
		GasFeeCap: &p.walletConfig.GasFeeCap,
		GasTipCap: &p.walletConfig.GasTipCap,
		Gas:       p.walletConfig.GasLimit,
		To:        &toAddress,
		Value:     &p.walletConfig.TxValue,
		Data:      data,
	})
	log.Debug("tx", "tx", tx)
	return tx
}

func (p *Provider) sign(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
	if p.walletConfig.SignerMethod == "static" {
		log.Debug("using static signer")
		privateKey, err := crypto.HexToECDSA(p.walletConfig.PrivateKey)
		if err != nil {
			return nil, err
		}
		return types.SignTx(tx, types.LatestSignerForChainID(&p.walletConfig.ChainID), privateKey)
	} else if p.walletConfig.SignerMethod == "signer" {
		tlsConfig := tls.CLIConfig{
			TLSCaCert: p.signerConfig.TLSCaCert,
			TLSCert:   p.signerConfig.TLSCert,
			TLSKey:    p.signerConfig.TLSKey,
		}
		client, err := signer.NewSignerClient(log.Root(), p.signerConfig.URL, tlsConfig)
		log.Debug("signerclient", "client", client, "err", err)
		if err != nil {
			return nil, err
		}

		if client == nil {
			return nil, errors.New("could not initialize signer client")
		}

		signedTx, err := client.SignTransaction(ctx, &p.walletConfig.ChainID, tx)
		log.Debug("signedtx", "tx", signedTx, "err", err)
		if err != nil {
			return nil, err
		}

		return signedTx, nil
	} else {
		return nil, errors.New("invalid signer method")
	}
}

func (p *Provider) nonce(ctx context.Context, client *ethclient.Client) (uint64, error) {
	fromAddress := common.HexToAddress(p.walletConfig.Address)
	return client.PendingNonceAt(ctx, fromAddress)
}

// Roundtrip send a new transaction to measure round trip latency
func (p *Provider) Roundtrip(ctx context.Context) {
	log.Debug("roundtrip", "provider", p.name)
}
