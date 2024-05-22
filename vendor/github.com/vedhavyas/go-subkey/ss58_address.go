package subkey

import (
	"github.com/decred/base58"
	"golang.org/x/crypto/blake2b"
)

const (
	ss58Prefix = "SS58PRE"
)

// SS58Address derives ss58 address from the accountID and network
// uses SS58Checksum checksum type
// SS58Checksum uses the concat(network, accountID) as blake2b hash pre-image
// More here: https://github.com/paritytech/substrate/wiki/External-Address-Format-(SS58)#checksum-types
func SS58Address(accountID []byte, network uint8) (string, error) {
	return toBase58(append([]byte{network}, accountID...), accountID, network)
}

// SS58AddressWithAccountIDChecksum derives ss58 address from the accountID, network
// uses AccountID checksum type
// AccountIDChecksum uses the accountID as the blake2b hash pre-image
// More here: https://github.com/paritytech/substrate/wiki/External-Address-Format-(SS58)#checksum-types
func SS58AddressWithAccountIDChecksum(accountID []byte, network uint8) (string, error) {
	return toBase58(accountID, accountID, network)
}

func toBase58(buf, accountID []byte, network uint8) (string, error) {
	cs, err := ss58Checksum(buf)
	if err != nil {
		return "", err
	}

	fb := append([]byte{network}, accountID...)
	fb = append(fb, cs[0:2]...)
	return base58.Encode(fb), nil
}

// https://github.com/paritytech/substrate/wiki/External-Address-Format-(SS58)#checksum-types
func ss58Checksum(data []byte) ([]byte, error) {
	hasher, err := blake2b.New(64, nil)
	if err != nil {
		return nil, err
	}

	_, err = hasher.Write([]byte(ss58Prefix))
	if err != nil {
		return nil, err
	}

	_, err = hasher.Write(data)
	if err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}
