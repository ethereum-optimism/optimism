// Copyright (C) OmiseGO - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and confidential
// Written by Jeff Ploughman <jeff@immutability.io>, October 2019

package util

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/scrypt"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// PrettyPrint prints an indented JSON payload. This is used for development debugging.
func PrettyPrint(v interface{}) string {
	jsonString, _ := json.Marshal(v)
	var out bytes.Buffer
	json.Indent(&out, jsonString, "", "  ")
	return out.String()
}

// Dedup removes duplicates from a list
func Dedup(stringSlice []string) []string {
	var returnSlice []string
	for _, value := range stringSlice {
		if !Contains(returnSlice, value) {
			returnSlice = append(returnSlice, value)
		}
	}
	return returnSlice
}

// Contains returns true if an element is present in a list
func Contains(stringSlice []string, searchString string) bool {
	for _, value := range stringSlice {
		if value == searchString {
			return true
		}
	}
	return false
}

// Encode will encode a raw key or seed
func Encode(src []byte) ([]byte, error) {
	buf := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(buf, src)

	return buf[:], nil
}

// Decode will decode the hex
func Decode(src []byte) ([]byte, error) {
	raw := make([]byte, hex.EncodedLen(len(src)))
	n, err := hex.Decode(raw, src)
	if err != nil {
		return nil, err
	}
	raw = raw[:n]
	return raw[:], nil
}

// SealWrapAppend is a helper for appending lists of paths into a single
// list.
func SealWrapAppend(paths ...[]string) []string {
	result := make([]string, 0, 10)
	for _, ps := range paths {
		result = append(result, ps...)
	}

	return result
}

// PathExistenceCheck checks storage for a path
func PathExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	out, err := req.Storage.Get(ctx, req.Path)
	if err != nil {
		return false, fmt.Errorf("existence check failed: %v", err)
	}

	return out != nil, nil
}

// ValidNumber returns a valid positive integer
func ValidNumber(input string) *big.Int {
	if input == "" {
		return big.NewInt(0)
	}
	matched, err := regexp.MatchString("([0-9])", input)
	if !matched || err != nil {
		return nil
	}
	amount := math.MustParseBig256(input)
	return amount.Abs(amount)
}

// Pow computes a^b for int64
func Pow(a, b int64) int64 {
	var result int64 = 1

	for 0 != b {
		if 0 != (b & 1) {
			result *= a

		}
		b >>= 1
		a *= a
	}

	return result
}

// ZeroKey removes the key from memory
func ZeroKey(k *ecdsa.PrivateKey) {
	b := k.D.Bits()
	for i := range b {
		b[i] = 0
	}
}

// EstimateGas attempts to determine the cost for a contract deploy... super annoying
func EstimateGas(opts *bind.TransactOpts, abi abi.ABI, bytecode []byte, backend bind.ContractBackend, params ...interface{}) (uint64, error) {
	var input []byte
	packed, err := abi.Pack("", params...)
	if err != nil {
		return 0, fmt.Errorf("failed to pack parameters: %v", err)
	}
	input = append(bytecode, packed...)
	msg := ethereum.CallMsg{From: opts.From, To: nil, Value: opts.Value, Data: input}
	gasLimit, err := backend.EstimateGas(ensureContext(opts.Context), msg)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas needed: %v", err)
	}
	return gasLimit, nil
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.TODO()
	}
	return ctx
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

const (
	version = 3
)

type encryptedKeyJSONV3 struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	ID      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

const (
	keyHeaderKDF = "scrypt"

	scryptR     = 8
	scryptDKLen = 32
)

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

// KeyFileName returns the name of the keystore file based on the account address
func KeyFileName(keyAddr common.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

// EncryptKey encrypts an ecdsa.PrivateKey and returns a JSON keystore
func EncryptKey(key *ecdsa.PrivateKey, address *common.Address, id uuid.UUID, auth string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(auth)

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(key.D, 32)

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}
	encryptedKeyJSONV3 := encryptedKeyJSONV3{
		hex.EncodeToString(address[:]),
		cryptoStruct,
		id.String(),
		version,
	}
	return json.Marshal(encryptedKeyJSONV3)
}

// ImportJSONKeystore decrypts a JSON keystore given a passphrase
func ImportJSONKeystore(keystoreBytes []byte, passphrase string) (*ecdsa.PrivateKey, error) {
	var key *keystore.Key
	key, err := keystore.DecryptKey(keystoreBytes, passphrase)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("failed to decrypt key")
	}

	return key.PrivateKey, err
}

// WriteKeyFile will create the keystore directory with appropriate permissions
// in case it is not present yet.
func WriteKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

// TokenAmount does the requisite math on tokens
func TokenAmount(amount int64, decimals uint8) *big.Int {
	var bigDecimal big.Int
	bigDecimal.SetString(fmt.Sprintf("%d", decimals), 10)
	power := Pow(10, bigDecimal.Int64())

	var bigPower, _ = new(big.Int).SetString(fmt.Sprintf("%d", power), 10)
	var bigAmount, _ = new(big.Int).SetString(fmt.Sprintf("%d", amount), 10)

	var bigProduct = new(big.Int)
	return bigProduct.Mul(bigPower, bigAmount)
}
