package state

import (
	"context"
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/datadir"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/ledgerwatch/erigon/node"
	"github.com/ledgerwatch/erigon/node/nodecfg"
	"github.com/ledgerwatch/erigon/p2p"
	"github.com/ledgerwatch/log/v3"
	"github.com/stretchr/testify/require"
)

var (
	testNodeKey, _ = crypto.GenerateKey()
	logger         = log.New()
)

func testNodeConfig(t *testing.T) *nodecfg.Config {
	return &nodecfg.Config{
		Name: "test node",
		P2P:  p2p.Config{PrivateKey: testNodeKey},
		Dirs: datadir.New(t.TempDir()),
	}
}

func TestGetStorage(t *testing.T) {
	addr := common.HexToAddress("0x4200000000000000000000000000000000000000")
	incarnation := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	storageKey := ether.CalcOVMETHStorageKey(addr)
	tableKey := append(addr.Bytes(), incarnation...)
	tableKey = append(tableKey, storageKey.Bytes()...)
	tableVal := []byte("test")

	stack, err := node.New(context.Background(), testNodeConfig(t), logger)
	require.NoError(t, err)
	defer stack.Close()

	db, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
	require.NoError(t, err)
	defer db.Close()

	if err = db.Update(context.Background(), func(tx kv.RwTx) error {
		return tx.Put(kv.PlainState, tableKey, tableVal)
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginRo(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	value, err := GetStorage(tx, addr, storageKey)
	require.NoError(t, err)
	require.Equal(t, common.BytesToHash(tableVal), *value)
}

func TestGetAccount(t *testing.T) {
	addr := common.HexToAddress("0x4200000000000000000000000000000000000000")
	tableVal := []byte("test")

	stack, err := node.New(context.Background(), testNodeConfig(t), logger)
	require.NoError(t, err)
	defer stack.Close()

	db, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
	require.NoError(t, err)
	defer db.Close()

	if err = db.Update(context.Background(), func(tx kv.RwTx) error {
		return tx.Put(kv.PlainState, addr.Bytes(), tableVal)
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginRo(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	value, err := GetAccount(tx, addr)
	require.NoError(t, err)
	require.Equal(t, tableVal, value)
}

func TestGetContractCode(t *testing.T) {
	addr := common.HexToAddress("0x4200000000000000000000000000000000000000")
	incarnation := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	contractKey := append(addr.Bytes(), incarnation...)
	contractHash := []byte("test")
	contractCode := []byte("contract code")

	stack, err := node.New(context.Background(), testNodeConfig(t), logger)
	require.NoError(t, err)
	defer stack.Close()

	db, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
	require.NoError(t, err)
	defer db.Close()

	if err = db.Update(context.Background(), func(tx kv.RwTx) error {
		if err := tx.Put(kv.PlainContractCode, contractKey, contractHash); err != nil {
			return err
		}
		if err := tx.Put(kv.Code, contractHash, contractCode); err != nil {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginRo(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	actualContractCode, err := GetContractCode(tx, addr)
	require.NoError(t, err)
	require.Equal(t, actualContractCode, actualContractCode)
}

func TestGetContractCodeHash(t *testing.T) {
	addr := common.HexToAddress("0x4200000000000000000000000000000000000000")
	incarnation := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	contractKey := append(addr.Bytes(), incarnation...)
	contractHash := []byte("test")

	stack, err := node.New(context.Background(), testNodeConfig(t), logger)
	require.NoError(t, err)
	defer stack.Close()

	db, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
	require.NoError(t, err)
	defer db.Close()

	if err = db.Update(context.Background(), func(tx kv.RwTx) error {
		return tx.Put(kv.PlainContractCode, contractKey, contractHash)
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginRo(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	actualContractHash, err := GetContractCodeHash(tx, addr)
	require.NoError(t, err)
	require.Equal(t, common.BytesToHash(contractHash), *actualContractHash)
}

func TestForEachStorage(t *testing.T) {
	addr := common.HexToAddress("0x4200000000000000000000000000000000000000")
	incarnation := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tableKey := append(addr.Bytes(), incarnation...)
	tables := []struct {
		key   []byte
		value []byte
	}{
		{append(tableKey, ether.CalcStorageKey(common.Address{1}, common.Big0).Bytes()...), []byte{1}},
		{append(tableKey, ether.CalcStorageKey(common.Address{2}, common.Big0).Bytes()...), []byte{1}},
		{append(tableKey, ether.CalcStorageKey(common.Address{3}, common.Big0).Bytes()...), []byte{1}},
	}

	stack, err := node.New(context.Background(), testNodeConfig(t), logger)
	require.NoError(t, err)
	defer stack.Close()

	db, err := node.OpenDatabase(context.Background(), stack.Config(), kv.ChainDB, "", false, logger)
	require.NoError(t, err)
	defer db.Close()

	if err = db.Update(context.Background(), func(tx kv.RwTx) error {
		for _, table := range tables {
			if err := tx.Put(kv.PlainState, table.key, table.value); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginRo(context.Background())
	require.NoError(t, err)
	defer tx.Rollback()

	expMap := make(map[common.Hash]common.Hash)
	for _, table := range tables {
		expMap[common.BytesToHash(table.key[28:])] = common.BytesToHash(table.value)
	}

	err = ForEachStorage(tx, addr, func(key, value common.Hash) bool {
		require.Equal(t, expMap[key], value)
		delete(expMap, key)
		return true
	})

	require.NoError(t, err)
	require.Equal(t, 0, len(expMap))
}
