package ether

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

var (
	// AddressPreimagePrefix is the byte prefix of address preimages
	// in Geth's database.
	AddressPreimagePrefix = []byte("addr-preimage-")

	// ErrStopIteration will stop iterators early when returned from the
	// iterator's callback.
	ErrStopIteration = errors.New("iteration stopped")

	// MintTopic is the topic for mint events on OVM ETH.
	MintTopic = common.HexToHash("0x0f6798a560793a54c3bcfe86a93cde1e73087d944c0ea20544137d4121396885")
)

type AddressCB func(address common.Address) error
type AddressCBWithHead func(address common.Address, headNum uint64) error
type AllowanceCB func(owner, spender common.Address) error

// IterateDBAddresses iterates over each address in Geth's address
// preimage database, calling the callback with the address.
func IterateDBAddresses(db ethdb.Database, cb AddressCB) error {
	iter := db.NewIterator(AddressPreimagePrefix, nil)
	for iter.Next() {
		if iter.Error() != nil {
			return iter.Error()
		}

		addr := common.BytesToAddress(bytes.TrimPrefix(iter.Key(), AddressPreimagePrefix))
		cbErr := cb(addr)
		if cbErr == ErrStopIteration {
			return nil
		}
		if cbErr != nil {
			return cbErr
		}
	}
	return iter.Error()
}

// IterateAddrList iterates over each address in an address list,
// calling the callback with the address.
func IterateAddrList(r io.Reader, cb AddressCB) error {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		addrStr := scan.Text()
		if !common.IsHexAddress(addrStr) {
			return fmt.Errorf("invalid address %s", addrStr)
		}
		err := cb(common.HexToAddress(addrStr))
		if err == ErrStopIteration {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// IterateAllowanceList iterates over each address in an allowance list,
// calling the callback with the owner and the spender.
func IterateAllowanceList(r io.Reader, cb AllowanceCB) error {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := scan.Text()
		splits := strings.Split(line, ",")
		if len(splits) != 2 {
			return fmt.Errorf("invalid allowance %s", line)
		}
		owner := splits[0]
		spender := splits[1]
		if !common.IsHexAddress(owner) {
			return fmt.Errorf("invalid address %s", owner)
		}
		if !common.IsHexAddress(spender) {
			return fmt.Errorf("invalid address %s", spender)
		}
		err := cb(common.HexToAddress(owner), common.HexToAddress(spender))
		if err == ErrStopIteration {
			return nil
		}
	}
	return nil
}

// IterateMintEvents iterates over each mint event in the database starting
// from head and stopping at genesis.
func IterateMintEvents(db ethdb.Database, headNum uint64, cb AddressCBWithHead) error {
	for headNum > 0 {
		hash := rawdb.ReadCanonicalHash(db, headNum)
		receipts, err := migration.ReadLegacyReceipts(db, hash, headNum)
		if err != nil {
			return err
		}
		for _, receipt := range receipts {
			for _, l := range receipt.Logs {
				if l.Address != predeploys.LegacyERC20ETHAddr {
					continue
				}

				if common.BytesToHash(l.Topics[0].Bytes()) != MintTopic {
					continue
				}

				err := cb(common.BytesToAddress(l.Topics[1][12:]), headNum)
				if errors.Is(err, ErrStopIteration) {
					return nil
				}
				if err != nil {
					return err
				}
			}
		}

		headNum--
	}
	return nil
}
