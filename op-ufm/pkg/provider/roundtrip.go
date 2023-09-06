package provider

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-ufm/pkg/metrics"
	iclients "github.com/ethereum-optimism/optimism/op-ufm/pkg/metrics/clients"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum-optimism/optimism/op-service/tls"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// RoundTrip send a new transaction to measure round trip latency
func (p *Provider) RoundTrip(ctx context.Context) {
	log.Debug("roundTripLatency",
		"provider", p.name)

	client, err := iclients.Dial(p.name, p.config.URL)
	if err != nil {
		log.Error("cant dial to provider",
			"provider", p.name,
			"url", p.config.URL,
			"err", err)
		return
	}

	var nonce uint64
	p.txPool.M.Lock()
	if p.txPool.Nonce == uint64(0) {
		nonce, err = client.PendingNonceAt(ctx, p.walletConfig.Address)
		if err != nil {
			log.Error("cant get nounce",
				"provider", p.name,
				"err", err)
			p.txPool.M.Unlock()
			return
		}
		p.txPool.Nonce = nonce
	} else {
		p.txPool.Nonce++
		nonce = p.txPool.Nonce
	}
	p.txPool.M.Unlock()

	txHash := common.Hash{}
	attempt := 0
	// used for timeout
	firstAttemptAt := time.Now()
	// used for actual round trip time (disregard retry time)
	roundTripStartedAt := time.Now()
	for {
		// sleep until we get a clear to send
		for {
			coolDown := time.Duration(p.config.SendTransactionCoolDown) - time.Since(p.txPool.LastSend)
			if coolDown > 0 {
				time.Sleep(coolDown)
			} else {
				break
			}
		}

		tx := p.createTx(nonce)
		txHash = tx.Hash()

		signedTx, err := p.sign(ctx, tx)
		if err != nil {
			log.Error("cant sign tx",
				"provider", p.name,
				"tx", tx,
				"err", err)
			return
		}

		txHash = signedTx.Hash()

		roundTripStartedAt = time.Now()
		err = client.SendTransaction(ctx, signedTx)
		if err != nil {
			if err.Error() == txpool.ErrAlreadyKnown.Error() ||
				err.Error() == txpool.ErrReplaceUnderpriced.Error() ||
				err.Error() == core.ErrNonceTooLow.Error() {
				if time.Since(firstAttemptAt) >= time.Duration(p.config.SendTransactionRetryTimeout) {
					log.Error("send transaction timed out (known already)",
						"provider", p.name,
						"hash", txHash.Hex(),
						"elapsed", time.Since(firstAttemptAt),
						"attempt", attempt,
						"nonce", nonce)
					metrics.RecordError(p.name, "ethclient.SendTransaction.nonce")
					return
				}
				log.Warn("tx already known, incrementing nonce and trying again",
					"provider", p.name,
					"nonce", nonce)
				time.Sleep(time.Duration(p.config.SendTransactionRetryInterval))

				p.txPool.M.Lock()
				p.txPool.Nonce++
				nonce = p.txPool.Nonce
				p.txPool.M.Unlock()
				attempt++
				if attempt%10 == 0 {
					log.Debug("retrying send transaction...",
						"provider", p.name,
						"attempt", attempt,
						"nonce", nonce,
						"elapsed", time.Since(firstAttemptAt))
				}
			} else {
				log.Error("cant send transaction",
					"provider", p.name,
					"err", err)
				metrics.RecordErrorDetails(p.name, "ethclient.SendTransaction", err)
				return
			}
		} else {
			break
		}
	}

	log.Info("transaction sent",
		"provider", p.name,
		"hash", txHash.Hex(),
		"nonce", nonce)

	// add to pool
	sentAt := time.Now()
	p.txPool.M.Lock()
	p.txPool.Transactions[txHash.Hex()] = &TransactionState{
		Hash:           txHash,
		ProviderSource: p.name,
		SentAt:         sentAt,
		SeenBy:         make(map[string]time.Time),
	}
	p.txPool.M.Unlock()

	var receipt *types.Receipt
	attempt = 0
	for receipt == nil {
		if time.Since(sentAt) >= time.Duration(p.config.ReceiptRetrievalTimeout) {
			log.Error("receipt retrieval timed out",
				"provider", p.name,
				"hash", txHash,
				"elapsed", time.Since(sentAt))
			return
		}
		time.Sleep(time.Duration(p.config.ReceiptRetrievalInterval))
		if attempt%10 == 0 {
			log.Debug("checking for receipt...",
				"provider", p.name,
				"attempt", attempt,
				"elapsed", time.Since(sentAt))
		}
		receipt, err = client.TransactionReceipt(ctx, txHash)
		if err != nil && !errors.Is(err, ethereum.NotFound) {
			log.Error("cant get receipt for transaction",
				"provider", p.name,
				"hash", txHash.Hex(),
				"err", err)
			return
		}
		attempt++
	}

	roundTripLatency := time.Since(roundTripStartedAt)

	metrics.RecordRoundTripLatency(p.name, roundTripLatency)
	metrics.RecordGasUsed(p.name, receipt.GasUsed)

	log.Info("got transaction receipt",
		"hash", txHash.Hex(),
		"roundTripLatency", roundTripLatency,
		"provider", p.name,
		"blockNumber", receipt.BlockNumber,
		"blockHash", receipt.BlockHash,
		"gasUsed", receipt.GasUsed)
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
		client, err := iclients.NewSignerClient(p.name, log.Root(), p.signerConfig.URL, tlsConfig)
		log.Debug("signerclient",
			"client", client,
			"err", err)
		if err != nil {
			return nil, err
		}

		if client == nil {
			return nil, errors.New("could not initialize signer client")
		}

		signedTx, err := client.SignTransaction(ctx, &p.walletConfig.ChainID, tx)
		if err != nil {
			return nil, err
		}

		return signedTx, nil
	} else {
		return nil, errors.New("invalid signer method")
	}
}
