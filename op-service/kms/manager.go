package kms

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

type KmsManager struct {
	keyId      string
	kmsSession *kms.KMS
}

func NewKmsManager(cfg CLIConfig) (*KmsManager, error) {
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
	return &KmsManager{
		keyId:      cfg.KmsKeyID,
		kmsSession: kms.New(session),
	}, nil
}
