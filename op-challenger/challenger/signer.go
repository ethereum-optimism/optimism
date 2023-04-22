package challenger

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OutputAtBlock is the struct representation of an L2OutputOracle output
// at a given L2 block number.
type OutputAtBlock struct {
	Output        common.Hash
	L2BlockNumber *big.Int
}

const domainType = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
const structType = "OutputAtBlock(bytes32 output,uint256 l2BlockNumber)"

func hashDomain(name, version string, chainID uint64, verifyingContract common.Address) []byte {
	domain := fmt.Sprintf("%s(%s)", domainType, "MyApp,1,1,<verifyingContract>")
	domainHash := crypto.Keccak256([]byte(domain))
	return domainHash
}

func hashTypedData(data OutputAtBlock) ([]byte, error) {
	structData := fmt.Sprintf("%s(%x,%x)", structType, data.Output, data.L2BlockNumber)
	structHash := crypto.Keccak256([]byte(structData))

	return structHash, nil
}

func signTypedData(privateKey *ecdsa.PrivateKey, data OutputAtBlock) ([]byte, error) {
	domainSeparator := hashDomain("MyApp", "1", 1, common.Address{})
	typedDataHash, err := hashTypedData(data)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	buf.Write([]byte("\x19\x01"))
	buf.Write(domainSeparator)
	buf.Write(typedDataHash)

	msg := crypto.Keccak256(buf.Bytes())
	signature, err := crypto.Sign(msg, privateKey)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// signOutput signs the typed data for a given output and L2 block number.
func (c *Challenger) signOutput(l2BlockNumber *big.Int, rootClaim common.Hash) ([]byte, error) {
	data := OutputAtBlock{
		Output:        rootClaim,
		L2BlockNumber: l2BlockNumber,
	}
	signature, err := signTypedData(c.privateKey, data)
	if err != nil {
		return nil, err
	}
	return signature, nil
}
