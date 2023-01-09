package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dyson/certman"
	optls "github.com/ethereum-optimism/optimism/op-service/tls"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type SignerClient struct {
	client *rpc.Client
	status string
	logger log.Logger
}

type ethLogger struct{ logger log.Logger }

func (l ethLogger) Printf(format string, v ...interface{}) { l.logger.Info(fmt.Sprintf(format, v...)) }

func NewSignerClient(logger log.Logger, endpoint string, tlsConfig optls.CLIConfig) (*SignerClient, error) {

	caCert, err := ioutil.ReadFile(tlsConfig.TLSCaCert)
	if err != nil {
		return nil, fmt.Errorf("failed to read tls.ca: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cm, err := certman.New(tlsConfig.TLSCert, tlsConfig.TLSKey)
	cm.Logger(ethLogger{logger: logger})
	if err != nil {
		logger.Error("failed to read tls cert or key", "err", err)
		return nil, err
	}
	if err := cm.Watch(); err != nil {
		logger.Error("failed to start certman watcher", "err", err)
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
				GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return cm.GetCertificate(nil)
				},
			},
		},
	}

	rpcClient, err := rpc.DialOptions(context.Background(), endpoint, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	signer := &SignerClient{logger: logger, client: rpcClient}
	// Check if reachable
	version, err := signer.pingVersion()
	if err != nil {
		return nil, err
	}
	signer.status = fmt.Sprintf("ok [version=%v]", version)
	return signer, nil
}

func NewSignerClientFromConfig(logger log.Logger, config CLIConfig) (*SignerClient, error) {
	return NewSignerClient(logger, config.Endpoint, config.TLSConfig)
}

func (s *SignerClient) pingVersion() (string, error) {
	var v string
	if err := s.client.Call(&v, "health_status"); err != nil {
		return "", err
	}
	return v, nil
}

func (s *SignerClient) SignTransaction(
	ctx context.Context,
	tx *types.Transaction,
) (*types.Transaction, error) {

	args := NewTransactionArgsFromTransaction(tx)

	var result hexutil.Bytes

	if err := s.client.Call(&result, "eth_signTransaction", args); err != nil {
		return nil, fmt.Errorf("eth_signTransaction failed: %w", err)
	}

	signed := &types.Transaction{}
	if err := signed.UnmarshalBinary(result); err != nil {
		return nil, err
	}

	return signed, nil
}
