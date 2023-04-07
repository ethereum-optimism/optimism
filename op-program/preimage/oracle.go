package preimage

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Key interface {
	PreimageKey() common.Hash
}

type Oracle interface {
	Get(key Key) []byte
}

type Type byte

const (
	Keccak256Type Type = iota
)

func makeKey(typ Type, keyData []byte) common.Hash {
	return crypto.Keccak256Hash([]byte{byte(typ)}, keyData)
}

type Keccak256Key common.Hash

func (k Keccak256Key) PreimageKey() common.Hash {
	return makeKey(Keccak256Type, k[:])
}

// OracleClient implements the Oracle by writing the pre-image key to the given stream,
// and reading back a length-prefixed value.
type OracleClient struct {
	rw io.ReadWriter
}

func NewOracleClient(rw io.ReadWriter) *OracleClient {
	return &OracleClient{rw: rw}
}

var _ Oracle = (*OracleClient)(nil)

func (o *OracleClient) Get(key Key) []byte {
	h := key.PreimageKey()
	if _, err := o.rw.Write(h[:]); err != nil {
		panic(fmt.Errorf("failed to write key %s (%T) to pre-image oracle: %w", key, key, err))
	}

	var lengthPrefix [8]byte
	if _, err := io.ReadFull(o.rw, lengthPrefix[:]); err != nil {
		panic(fmt.Errorf("failed to read pre-image length of key %s (%T) from pre-image oracle: %w", key, key, err))
	}

	length := binary.LittleEndian.Uint64(lengthPrefix[:])
	if length == 0 { // don't read empty payloads
		return nil
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(o.rw, payload); err != nil {
		panic(fmt.Errorf("failed to read pre-image payload (length %d) of key %s (%T) from pre-image oracle: %w", length, key, key, err))
	}

	return payload
}

// OracleServer serves the pre-image requests of the OracleClient, implementing the same protocol as the onchain VM.
type OracleServer struct {
	rw io.ReadWriter
}

func NewOracleServer(rw io.ReadWriter) *OracleServer {
	return &OracleServer{rw: rw}
}

func (o *OracleServer) NextPreimageRequest(getPreimage func(key common.Hash) ([]byte, error)) error {
	var key common.Hash
	if _, err := io.ReadFull(o.rw, key[:]); err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("failed to read requested pre-image key: %w", err)
	}
	value, err := getPreimage(key)
	if err != nil {
		return fmt.Errorf("failed to serve pre-image %s request: %w", key, err)
	}

	var lengthPrefix [8]byte
	binary.LittleEndian.PutUint64(lengthPrefix[:], uint64(len(value)))
	if _, err := o.rw.Write(lengthPrefix[:]); err != nil {
		return fmt.Errorf("failed to write length-prefix %x: %w", lengthPrefix, err)
	}
	if len(value) == 0 {
		return nil
	}
	if _, err := o.rw.Write(value); err != nil {
		return fmt.Errorf("failed to write pre-image value (%d long): %w", len(value), err)
	}
	return nil
}
