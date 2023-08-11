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

func NewKmsManager(keyId, endpoint, region string) (*KmsManager, error) {
	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      aws.String(endpoint),
		Endpoint:    aws.String(region),
	})
	if err != nil {
		return nil, err
	}
	return &KmsManager{
		keyId:      keyId,
		kmsSession: kms.New(session),
	}, nil
}
