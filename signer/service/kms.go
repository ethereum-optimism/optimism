package service

import (
	"context"
	"hash/crc32"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type SignatureProvider interface {
	Sign(ctx context.Context, keyName string, digest []byte) ([]byte, error)
}

type KMSClient struct {
	logger log.Logger
	client *kms.KeyManagementClient
}

type KMSClientSignRequestCorruptedError struct{}

func (e *KMSClientSignRequestCorruptedError) Error() string {
	return "kms sign request corrupted in transit"
}

type KMSClientSignResponseCorruptedError struct{}

func (e *KMSClientSignResponseCorruptedError) Error() string {
	return "kms sign response corrupted in transit"
}

func NewKMSClient(logger log.Logger) SignatureProvider {
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		logger.Error("failed to initialize kms client", "error", err)
		panic(err)
	}
	return &KMSClient{logger, client}
}

func (c *KMSClient) Sign(ctx context.Context, keyName string, digest []byte) ([]byte, error) {

	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)
	}
	digestCRC32C := crc32c(digest)

	request := kmspb.AsymmetricSignRequest{
		Name:       keyName,
		Data:       digest,
		DataCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}
	result, err := c.client.AsymmetricSign(ctx, &request)
	if err != nil {
		return nil, errors.Wrap(err, "kms sign request failed")
	}
	if result.Name != request.Name {
		return nil, errors.WithStack(new(KMSClientSignRequestCorruptedError))
	}
	if result.VerifiedDataCrc32C == false {
		return nil, errors.WithStack(new(KMSClientSignRequestCorruptedError))
	}
	if int64(crc32c(result.Signature)) != result.SignatureCrc32C.Value {
		return nil, errors.WithStack(new(KMSClientSignResponseCorruptedError))
	}

	return result.Signature, nil
}
