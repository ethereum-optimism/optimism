package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"
)

func Priv2PeerID(r io.Reader) (string, error) {
	b, err := readHexData(r)
	if err != nil {
		return "", err
	}

	p, err := crypto.UnmarshalSecp256k1PrivateKey(b)
	if err != nil {
		return "", fmt.Errorf("failed to parse priv key from %d bytes: %w", len(b), err)
	}

	pid, err := peer.IDFromPrivateKey(p)
	if err != nil {
		return "", fmt.Errorf("failed to parse peer ID from private key: %w", err)
	}
	return pid.String(), nil
}

func Pub2PeerID(r io.Reader) (string, error) {
	b, err := readHexData(r)
	if err != nil {
		return "", err
	}

	p, err := crypto.UnmarshalSecp256k1PublicKey(b)
	if err != nil {
		return "", fmt.Errorf("failed to parse pub key from %d bytes: %w", len(b), err)
	}

	pid, err := peer.IDFromPublicKey(p)
	if err != nil {
		return "", fmt.Errorf("failed to parse peer ID from public key: %w", err)
	}

	return pid.String(), nil
}

func readHexData(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	rawStr := strings.TrimSpace(string(data))
	rawStr = strings.TrimPrefix(rawStr, "0x")
	b, err := hex.DecodeString(rawStr)
	if err != nil {
		return nil, fmt.Errorf("p2p key is not formatted in hex chars: %w", err)
	}
	return b, nil
}

var Subcommands = cli.Commands{
	{
		Name:  "priv2id",
		Usage: "Reads a private key from STDIN, and returns a peer ID",
		Action: func(ctx *cli.Context) error {
			key, err := Priv2PeerID(os.Stdin)
			if err != nil {
				return err
			}
			fmt.Println(key)
			return nil
		},
	},
	{
		Name:  "pub2id",
		Usage: "Reads a public key from STDIN, and returns a peer ID",
		Action: func(ctx *cli.Context) error {
			key, err := Pub2PeerID(os.Stdin)
			if err != nil {
				return err
			}
			fmt.Println(key)
			return nil
		},
	},
	{
		Name:  "genkey",
		Usage: "Generates a private key",
		Action: func(ctx *cli.Context) error {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				return fmt.Errorf("failed to get entropy: %w", err)
			}
			fmt.Println(hex.EncodeToString(buf))
			return nil
		},
	},
}
