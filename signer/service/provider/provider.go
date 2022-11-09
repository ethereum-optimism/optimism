//go:generate mockgen -destination=mocks/mock_provider.go -package=mocks github.com/ethereum-optimism/optimism/signer/service/provider SignatureProvider
package provider

import "context"

type SignatureProvider interface {
	Sign(ctx context.Context, keyName string, digest []byte) ([]byte, error)
	GetPublicKey(ctx context.Context, keyName string) ([]byte, error)
}
