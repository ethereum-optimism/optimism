package service

import (
	"bytes"
	"context"
	"encoding/asn1"
	"math/big"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/pkg/errors"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/signer/service/provider"
)

type SignerService struct {
	logger   log.Logger
	provider provider.SignatureProvider
}

func NewSignerService(logger log.Logger) SignerService {
	return NewSignerServiceWithProvider(logger, provider.NewCloudKMSSignatureProvider(logger))
}

func NewSignerServiceWithProvider(
	logger log.Logger,
	provider provider.SignatureProvider,
) SignerService {
	return SignerService{logger, provider}
}

func (s *SignerService) RegisterAPIs(server *oprpc.Server) {
	server.AddAPI(rpc.API{
		Namespace: "signer",
		Service:   s,
	})
}

type SignTransactionResponse struct {
	Signature hexutil.Bytes `json:"signature"`
}

func (s *SignerService) SignTransaction(
	ctx context.Context, txraw hexutil.Bytes, digest hexutil.Bytes,
) (*SignTransactionResponse, error) {

	// TODO: will fix hardcoded key name when implementing auth
	keyName := "projects/op-dev-signer/locations/nam6/keyRings/signer/cryptoKeys/zhwrd-test-key-1/cryptoKeyVersions/1"
	clientName := "client"

	labels := prometheus.Labels{"client": clientName, "status": "error", "error": ""}
	defer func() {
		MetricSignTransactionTotal.With(labels).Inc()
	}()

	tx := &types.Transaction{}
	if err := tx.UnmarshalBinary(txraw); err != nil {
		labels["error"] = "transaction_unmarshal_error"
		return nil, new(TransactionUnmarshalError)
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	expectedDigest := signer.Hash(tx)
	s.logger.Debug(fmt.Sprintf("expected digest: %s", hexutil.Encode(expectedDigest.Bytes())))

	if bytes.Compare(expectedDigest.Bytes(), digest) != 0 {
		labels["error"] = "invalid_digest_error"
		return nil, new(InvalidDigestError)
	}

	rawSignature, err := s.provider.Sign(ctx, keyName, digest)
	if err != nil {
		labels["error"] = "sign_error"
		return nil, err
	}
	s.logger.Debug(fmt.Sprintf("raw signature: %s", hexutil.Encode(rawSignature)))

	signature, err := compressSignature(rawSignature)
	if err != nil {
		labels["error"] = "compress_signature_error"
		return nil, errors.Wrap(err, "failed to compress signature")
	}

	publicKey, err := s.provider.GetPublicKey(ctx, keyName)
	if err != nil {
		labels["error"] = "get_public_key_error"
		return nil, errors.Wrap(err, "failed to get public key")
	}

	recId, err := calculateRecoveryID(signature, digest, publicKey)
	if err != nil {
		labels["error"] = "calculate_recovery_id_error"
		return nil, errors.Wrap(err, "failed to calculate recovery id")
	}
	signature = append(signature, byte(recId))

	labels["status"] = "success"
	s.logger.Info(
		"Signed transaction",
		"digest", digest,
		"client.name", clientName,
		"keyname", keyName,
		"tx.raw", txraw,
		"tx.value", tx.Value(),
		"tx.to", tx.To().Hex(),
		"tx.nonce", tx.Nonce(),
		"tx.gas", tx.Gas(),
		"tx.gasprice", tx.GasPrice(),
		"tx.hash", tx.Hash().Hex(),
		"tx.chainid", tx.ChainId(),
		"signature", hexutil.Encode(signature),
	)

	return &SignTransactionResponse{Signature: hexutil.Bytes(signature)}, nil
}

// compressSignature compresses raw signature output from kms (>70 bytes) into 64 bytes
func compressSignature(kmsSignature []byte) ([]byte, error) {
	var parsedSig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(kmsSignature, &parsedSig); err != nil {
		return nil, errors.Wrap(err, "asn1.Unmarshal error")
	}

	curveOrderLen := 32
	signature := make([]byte, 2*curveOrderLen)

	// left pad R and S with zeroes
	rBytes := parsedSig.R.Bytes()
	sBytes := parsedSig.S.Bytes()
	copy(signature[curveOrderLen-len(rBytes):], rBytes)
	copy(signature[len(signature)-len(sBytes):], sBytes)

	return signature, nil
}

// calculateRecoveryID calculates the signature recovery id (65th byte, [0-3])
func calculateRecoveryID(signature, digest, pubKey []byte) (int, error) {
	recId := -1;
	var errorRes error

	for i := 0; i < 4; i++ {
		recSig := append(signature, byte(i))
		publicKey, err := secp256k1.RecoverPubkey(digest, recSig)
		if err != nil {
			errorRes = err
			continue
		}
		if bytes.Compare(publicKey, pubKey) == 0 {
			recId = i
			break
		}
	}

	if recId == -1 {
		return recId, errors.Wrap(errorRes, "failed to calculate recovery id, should never happen")
	}
	return recId, nil
}
