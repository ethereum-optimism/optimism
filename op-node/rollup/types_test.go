package rollup

import (
	"encoding/json"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
)

func randConfig() *Config {
	randHash := func() (out common.Hash) {
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
		},
		BlockTime:           2,
		MaxSequencerDrift:   100,
		SeqWindowSize:       2,
		L1ChainID:           big.NewInt(900),
		FeeRecipientAddress: randAddr(),
		BatchInboxAddress:   randAddr(),
		BatchSenderAddress:  randAddr(),
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
