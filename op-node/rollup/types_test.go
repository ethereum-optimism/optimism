package rollup

import (
	"context"
	"encoding/json"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
)

func randConfig() *Config {
	randHash := func() (out [32]byte) {
		rand.Read(out[:])
		return
	}
	randAddr := func() (out common.Address) { // we need generics...
		rand.Read(out[:])
		return
	}
	return &Config{
		Genesis: Genesis{
			L1:     eth.BlockID{Hash: randHash(), Number: 424242},
			L2:     eth.BlockID{Hash: randHash(), Number: 1337},
			L2Time: uint64(time.Now().Unix()),
			SystemConfig: eth.SystemConfig{
				BatcherAddr: randAddr(),
				Overhead:    randHash(),
				Scalar:      randHash(),
				GasLimit:    1234567,
			},
		},
		BlockTime:              2,
		MaxSequencerDrift:      100,
		SeqWindowSize:          2,
		ChannelTimeout:         123,
		L1ChainID:              big.NewInt(900),
		L2ChainID:              big.NewInt(901),
		BatchInboxAddress:      randAddr(),
		DepositContractAddress: randAddr(),
		L1SystemConfigAddress:  randAddr(),
	}
}

func TestConfigJSON(t *testing.T) {
	config := randConfig()
	data, err := json.Marshal(config)
	assert.NoError(t, err)
	var roundTripped Config
	assert.NoError(t, json.Unmarshal(data, &roundTripped))
	assert.Equal(t, &roundTripped, config)
}

type mockL1Client struct {
	chainID *big.Int
	Hash    common.Hash
	code    []byte
}

func (m *mockL1Client) ChainID(context.Context) (*big.Int, error) {
	return m.chainID, nil
}

func (m *mockL1Client) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	return eth.L1BlockRef{
		Hash:   m.Hash,
		Number: 100,
	}, nil
}

func (m *mockL1Client) CodeAt(ctx context.Context, addr common.Address, blockTag string) ([]byte, error) {
	return m.code, nil
}

func TestValidateL1Config(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(100)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}, code: []byte{0x01}}
	err := config.ValidateL1Config(context.TODO(), &mockClient)
	assert.NoError(t, err)
}

func TestValidateL1ConfigInvalidChainIdFails(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(101)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}, code: []byte{0x01}}
	err := config.ValidateL1Config(context.TODO(), &mockClient)
	assert.Error(t, err)
	config.L1ChainID = big.NewInt(99)
	err = config.ValidateL1Config(context.TODO(), &mockClient)
	assert.Error(t, err)
}

func TestValidateL1ConfigInvalidGenesisHashFails(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(100)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x00}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}, code: []byte{0x01}}
	err := config.ValidateL1Config(context.TODO(), &mockClient)
	assert.Error(t, err)
	config.Genesis.L1.Hash = [32]byte{0x02}
	err = config.ValidateL1Config(context.TODO(), &mockClient)
	assert.Error(t, err)
}

func TestValidateL1ConfigInvalidBytecodeFails(t *testing.T) {
	// TODO
}

func TestCheckL1ChainID(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(100)
	err := config.CheckL1ChainID(context.TODO(), &mockL1Client{chainID: big.NewInt(100)})
	assert.NoError(t, err)
	err = config.CheckL1ChainID(context.TODO(), &mockL1Client{chainID: big.NewInt(101)})
	assert.Error(t, err)
	err = config.CheckL1ChainID(context.TODO(), &mockL1Client{chainID: big.NewInt(99)})
	assert.Error(t, err)
}

func TestCheckL1BlockRefByNumber(t *testing.T) {
	config := randConfig()
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.CheckL1GenesisBlockHash(context.TODO(), &mockClient)
	assert.NoError(t, err)
	mockClient.Hash = common.Hash{0x02}
	err = config.CheckL1GenesisBlockHash(context.TODO(), &mockClient)
	assert.Error(t, err)
	mockClient.Hash = common.Hash{0x00}
	err = config.CheckL1GenesisBlockHash(context.TODO(), &mockClient)
	assert.Error(t, err)
}

type mockL2Client struct {
	chainID *big.Int
	Hash    common.Hash
}

func (m *mockL2Client) ChainID(context.Context) (*big.Int, error) {
	return m.chainID, nil
}

func (m *mockL2Client) L2BlockRefByNumber(ctx context.Context, number uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{
		Hash:   m.Hash,
		Number: 100,
	}, nil
}

func TestValidateL2Config(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(100)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x01}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient)
	assert.NoError(t, err)
}

func TestValidateL2ConfigInvalidChainIdFails(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(101)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x01}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient)
	assert.Error(t, err)
	config.L2ChainID = big.NewInt(99)
	err = config.ValidateL2Config(context.TODO(), &mockClient)
	assert.Error(t, err)
}

func TestValidateL2ConfigInvalidGenesisHashFails(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(100)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x00}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient)
	assert.Error(t, err)
	config.Genesis.L2.Hash = [32]byte{0x02}
	err = config.ValidateL2Config(context.TODO(), &mockClient)
	assert.Error(t, err)
}

func TestCheckL2ChainID(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(100)
	err := config.CheckL2ChainID(context.TODO(), &mockL2Client{chainID: big.NewInt(100)})
	assert.NoError(t, err)
	err = config.CheckL2ChainID(context.TODO(), &mockL2Client{chainID: big.NewInt(101)})
	assert.Error(t, err)
	err = config.CheckL2ChainID(context.TODO(), &mockL2Client{chainID: big.NewInt(99)})
	assert.Error(t, err)
}

func TestCheckL2BlockRefByNumber(t *testing.T) {
	config := randConfig()
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x01}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.CheckL2GenesisBlockHash(context.TODO(), &mockClient)
	assert.NoError(t, err)
	mockClient.Hash = common.Hash{0x02}
	err = config.CheckL2GenesisBlockHash(context.TODO(), &mockClient)
	assert.Error(t, err)
	mockClient.Hash = common.Hash{0x00}
	err = config.CheckL2GenesisBlockHash(context.TODO(), &mockClient)
	assert.Error(t, err)
}
