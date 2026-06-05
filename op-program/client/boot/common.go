package boot

import preimage "github.com/ethereum-optimism/optimism/op-preimage"

type oracleClient interface {
	Get(key preimage.Key) []byte
}
