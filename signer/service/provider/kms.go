package provider

import (
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"hash/crc32"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type CloudKMSSignatureProvider struct {
	logger log.Logger
	client *kms.KeyManagementClient
}

type CloudKMSSignRequestCorruptedError struct{}

func (e *CloudKMSSignRequestCorruptedError) Error() string {
	return "cloud kms sign request corrupted in transit"
}

type CloudKMSSignResponseCorruptedError struct{}

func (e *CloudKMSSignResponseCorruptedError) Error() string {
	return "cloud kms sign response corrupted in transit"
}

func NewCloudKMSSignatureProvider(logger log.Logger) SignatureProvider {
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		logger.Error("failed to initialize kms client", "error", err)
		panic(err)
	}
	return &CloudKMSSignatureProvider{logger, client}
}

func (c *CloudKMSSignatureProvider) Sign(
	ctx context.Context,
	keyName string,
	digest []byte,
) ([]byte, error) {

	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)
	}
	digestCRC32C := crc32c(digest)

	request := &kmspb.AsymmetricSignRequest{
		Name: keyName,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest,
			},
		},
		DigestCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}

	result, err := c.client.AsymmetricSign(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "kms sign request failed")
	}
	if result.Name != request.Name {
		return nil, errors.WithStack(new(CloudKMSSignRequestCorruptedError))
	}
	if result.VerifiedDigestCrc32C == false {
		return nil, errors.WithStack(new(CloudKMSSignRequestCorruptedError))
	}
	if int64(crc32c(result.Signature)) != result.SignatureCrc32C.Value {
		return nil, errors.WithStack(new(CloudKMSSignResponseCorruptedError))
	}

	return result.Signature, nil
}

func (c *CloudKMSSignatureProvider) GetPublicKey(
	ctx context.Context,
	keyName string,
) ([]byte, error) {

	request := kmspb.GetPublicKeyRequest{
		Name: keyName,
	}

	result, err := c.client.GetPublicKey(ctx, &request)
	if err != nil {
		return nil, errors.Wrap(err, "kms get public key request failed")
	}

	key := []byte(result.Pem)

	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)
	}
	if int64(crc32c(key)) != result.PemCrc32C.Value {
		return nil, errors.Wrap(err, "getPublicKey: response corrupted in-transit")
	}

	return DecodePublicKeyPEM(key)
}

type publicKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

var (
	oidPublicKeyECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	oidNamedCurveP256      = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
	oidNamedCurveSECP256K1 = asn1.ObjectIdentifier{1, 3, 132, 0, 10}
)

func DecodePublicKeyPEM(key []byte) ([]byte, error) {
	block, rest := pem.Decode([]byte(key))
	if len(rest) > 0 {
		return nil, errors.Errorf("crypto: failed to parse PEM string, not all bytes in PEM key were decoded: %x", rest)
	}

	pkBytes, err := x509ParseECDSAPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrapf(err, "crypto: failed to parse PEM string")
	}

	return pkBytes, err
}

func x509ParseECDSAPublicKey(derBytes []byte) ([]byte, error) {
	var pki publicKeyInfo
	if rest, err := asn1.Unmarshal(derBytes, &pki); err != nil {
		return nil, err
	} else if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after ASN.1 of public-key")
	}

	if !pki.Algorithm.Algorithm.Equal(oidPublicKeyECDSA) {
		return nil, errors.New("x509: unknown public key algorithm")
	}

	asn1Data := pki.PublicKey.RightAlign()
	paramsData := pki.Algorithm.Parameters.FullBytes
	namedCurveOID := new(asn1.ObjectIdentifier)
	rest, err := asn1.Unmarshal(paramsData, namedCurveOID)
	if err != nil {
		return nil, errors.Wrap(err, "x509: failed to parse ECDSA parameters as named curve")
	}
	if len(rest) != 0 {
		return nil, errors.New("x509: trailing data after ECDSA parameters")
	}

	if !(namedCurveOID.Equal(oidNamedCurveP256) || namedCurveOID.Equal(oidNamedCurveSECP256K1)) {
		return nil, errors.New("x509: unsupported elliptic curve")
	}

	if asn1Data[0] != 4 { // uncompressed form
		return nil, errors.New("x509: only uncompressed keys are supported")
	}

	return asn1Data, nil
}
