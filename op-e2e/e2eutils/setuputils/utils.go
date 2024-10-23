package setuputils

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/crypto"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

func hexPriv(in *ecdsa.PrivateKey) string {
	b := crypto.EncodePrivKey(in)
	return hexutil.Encode(b)
}

func NewTxMgrConfig(l1Addr endpoint.RPC, privKey *ecdsa.PrivateKey) txmgr.CLIConfig {
	return txmgr.CLIConfig{
		L1RPCURL:                  l1Addr.RPC(),
		PrivateKey:                hexPriv(privKey),
		NumConfirmations:          1,
		SafeAbortNonceTooLowCount: 3,
		FeeLimitMultiplier:        5,
		ResubmissionTimeout:       3 * time.Second,
		ReceiptQueryInterval:      50 * time.Millisecond,
		NetworkTimeout:            2 * time.Second,
		TxNotInMempoolTimeout:     2 * time.Minute,
	}
}
