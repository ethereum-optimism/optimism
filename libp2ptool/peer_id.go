package libp2ptool

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
	"io/ioutil"
	"strings"
)

func ReadPeerID(isPrivateKey bool, r io.Reader) (string, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	rawStr := strings.TrimSpace(string(data))
	rawStr = strings.TrimPrefix(rawStr, "0x")
	b, err := hex.DecodeString(rawStr)
	if err != nil {
		return "", errors.New("p2p priv key is not formatted in hex chars")
	}

	var pid peer.ID
	if isPrivateKey {
		p, err := crypto.UnmarshalSecp256k1PrivateKey(b)
		if err != nil {
			return "", fmt.Errorf("failed to parse priv key from %d bytes: %w", len(b), err)
		}

		pid, err = peer.IDFromPrivateKey(p)
		if err != nil {
			return "", fmt.Errorf("failed to parse peer ID from private key: %w", err)
		}
	} else {
		p, err := crypto.UnmarshalSecp256k1PublicKey(b)
		if err != nil {
			return "", fmt.Errorf("failed to parse pub key from %d bytes: %w", len(b), err)
		}

		pid, err = peer.IDFromPublicKey(p)
		if err != nil {
			return "", fmt.Errorf("failed to parse peer ID from public key: %w", err)
		}
	}

	return pid.String(), nil
}
