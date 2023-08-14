package txmgr

import (
	"math/big"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	ethawskmssigner "github.com/welthee/go-ethereum-aws-kms-tx-signer"
)

//go:generate mockery --name KmsManager --output ./mocks
type KmsManager interface {
	GetAddr() (common.Address, error)
	Sign(chainID *big.Int, tx *types.Transaction) (*types.Transaction, error)
}

type KmsConfig struct {
	keyId      string
	kmsSession *kms.KMS
}

func NewKmsConfig(cfg CLIConfig) (*KmsConfig, error) {
	if cfg.KmsKeyID == "" || cfg.KmsEndpoint == "" || cfg.KmsRegion == "" {
		return nil, nil
	}

	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      aws.String(cfg.KmsRegion),
		Endpoint:    aws.String(cfg.KmsEndpoint),
	})
	if err != nil {
		return nil, err
	}
	return &KmsConfig{
		keyId:      cfg.KmsKeyID,
		kmsSession: kms.New(session),
	}, nil
}

func (k *KmsConfig) GetAddr() (common.Address, error) {
	pubkey, err := ethawskmssigner.GetPubKey(k.kmsSession, k.keyId)
	if err != nil {
		return common.Address{}, err
	}
	addr := crypto.PubkeyToAddress(*pubkey)
	return addr, nil
}

func (k *KmsConfig) Sign(chainID *big.Int, tx *types.Transaction) (*types.Transaction, error) {
	transactOpts, err := ethawskmssigner.NewAwsKmsTransactorWithChainID(k.kmsSession, k.keyId, chainID)
	if err != nil {
		return nil, err
	}
	signedTx, err := transactOpts.Signer(transactOpts.From, tx)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}
