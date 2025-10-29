package rollup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func randConfig() *Config {
	rng := rand.New(rand.NewSource(1234))
	randHash := func() (out [32]byte) {
		rng.Read(out[:])
		return out
	}
	randAddr := func() (out common.Address) { // we need generics...
		rng.Read(out[:])
		return out
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
		ChannelTimeoutBedrock:  123,
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
	err     error
}

func (m *mockL1Client) ChainID(context.Context) (*big.Int, error) {
	return m.chainID, nil
}

func (m *mockL1Client) L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	if m.err != nil {
		return eth.L1BlockRef{}, m.err
	}
	return eth.L1BlockRef{
		Hash:   m.Hash,
		Number: 100,
	}, nil
}

func TestValidateL1Config(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(100)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL1Config(context.TODO(), testlog.Logger(t, log.LvlInfo), &mockClient)
	assert.NoError(t, err)
}

func TestValidateL1ConfigInvalidChainIdFails(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(101)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	logger := testlog.Logger(t, log.LvlInfo)
	err := config.ValidateL1Config(context.TODO(), logger, &mockClient)
	assert.Error(t, err)
	config.L1ChainID = big.NewInt(99)
	err = config.ValidateL1Config(context.TODO(), logger, &mockClient)
	assert.Error(t, err)
}

func TestValidateL1ConfigInvalidGenesisHashFails(t *testing.T) {
	config := randConfig()
	config.L1ChainID = big.NewInt(100)
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x00}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	logger := testlog.Logger(t, log.LvlInfo)
	err := config.ValidateL1Config(context.TODO(), logger, &mockClient)
	assert.Error(t, err)
	config.Genesis.L1.Hash = [32]byte{0x02}
	err = config.ValidateL1Config(context.TODO(), logger, &mockClient)
	assert.Error(t, err)
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
	logger := testlog.Logger(t, log.LvlInfo)
	config := randConfig()
	config.Genesis.L1.Number = 100
	config.Genesis.L1.Hash = [32]byte{0x01}
	mockClient := mockL1Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.CheckL1GenesisBlockHash(context.Background(), logger, &mockClient)
	assert.NoError(t, err)
	mockClient.Hash = common.Hash{0x02}
	err = config.CheckL1GenesisBlockHash(context.Background(), logger, &mockClient)
	assert.Error(t, err)
	mockClient.Hash = common.Hash{0x00}
	err = config.CheckL1GenesisBlockHash(context.Background(), logger, &mockClient)
	assert.Error(t, err)

	mockClient.err = errors.New("block not found")
	err = config.CheckL1GenesisBlockHash(context.Background(), logger, &mockClient)
	assert.NoError(t, err)
}

// TestRandomConfigDescription tests that the description works for different variations of a random rollup config.
func TestRandomConfigDescription(t *testing.T) {
	t.Run("named L2", func(t *testing.T) {
		config := randConfig()
		out := config.Description(map[string]string{config.L2ChainID.String(): "foobar chain"})
		require.Contains(t, out, "foobar chain")
	})
	t.Run("named L1", func(t *testing.T) {
		config := randConfig()
		config.L1ChainID = big.NewInt(11155111)
		out := config.Description(map[string]string{config.L2ChainID.String(): "foobar chain"})
		require.Contains(t, out, "sepolia")
	})
	t.Run("unnamed", func(t *testing.T) {
		config := randConfig()
		out := config.Description(nil)
		require.Contains(t, out, "(unknown L1)")
		require.Contains(t, out, "(unknown L2)")
	})
	t.Run("regolith unset", func(t *testing.T) {
		config := randConfig()
		config.RegolithTime = nil
		out := config.Description(nil)
		require.Contains(t, out, "Regolith: (not configured)")
	})
	t.Run("regolith genesis", func(t *testing.T) {
		config := randConfig()
		config.RegolithTime = new(uint64)
		out := config.Description(nil)
		require.Contains(t, out, "Regolith: @ genesis")
	})
	t.Run("optimism forks check,  date", func(t *testing.T) {
		config := randConfig()
		r := uint64(1677119335)
		config.RegolithTime = &r
		c := uint64(1677119336)
		config.CanyonTime = &c
		d := uint64(1677119337)
		config.DeltaTime = &d
		e := uint64(1677119338)
		config.EcotoneTime = &e
		f := uint64(1677119339)
		config.FjordTime = &f
		h := uint64(1677119340)
		config.HoloceneTime = &h
		i := uint64(1677119341)
		config.IsthmusTime = &i
		j := uint64(1677119342)
		config.JovianTime = &j
		it := uint64(1677119343)
		config.InteropTime = &it

		out := config.Description(nil)
		// Don't check human-readable part of the date, it's timezone-dependent.
		// Don't make this test fail only in Australia :')
		require.Contains(t, out, fmt.Sprintf("Regolith: @ %d ~ ", r))
		require.Contains(t, out, fmt.Sprintf("Canyon: @ %d ~ ", c))
		require.Contains(t, out, fmt.Sprintf("Delta: @ %d ~ ", d))
		require.Contains(t, out, fmt.Sprintf("Ecotone: @ %d ~ ", e))
		require.Contains(t, out, fmt.Sprintf("Fjord: @ %d ~ ", f))
		require.Contains(t, out, fmt.Sprintf("Holocene: @ %d ~ ", h))
		require.Contains(t, out, fmt.Sprintf("Isthmus: @ %d ~ ", i))
		require.Contains(t, out, fmt.Sprintf("Jovian: @ %d ~ ", j))
		require.Contains(t, out, fmt.Sprintf("Interop: @ %d ~ ", it))
	})
}

// TestConfig_ActivationTime tests that all getters and setters for all scheduleable forks are
// present and working.
// It also covers the Is<ForkName>(ts) convenience methods.
func TestConfig_ActivationTime(t *testing.T) {
	for i, fork := range scheduleableForks {
		t.Run(string(fork), func(t *testing.T) {
			var cfg Config
			ts := uint64((i + 1) * 1000)
			cfg.SetActivationTime(fork, &ts)
			gts := cfg.ActivationTime(fork)
			require.NotNil(t, gts)
			require.Equal(t, ts, *gts, "activation time for fork %s", fork)

			// Reflectively call Is<ForkName>
			name := string(fork)
			methodName := "Is" + strings.ToUpper(name[:1]) + name[1:]
			m := reflect.ValueOf(&cfg).MethodByName(methodName)
			require.True(t, m.IsValid(), "method %s not found", methodName)
			out := m.Call([]reflect.Value{reflect.ValueOf(ts)})
			require.Len(t, out, 1)
			require.True(t, out[0].Bool(), "%s(%d) should be true", methodName, ts)
			prev := ts - 1
			out = m.Call([]reflect.Value{reflect.ValueOf(prev)})
			require.Len(t, out, 1)
			require.False(t, out[0].Bool(), "%s(%d) should be false", methodName, prev)
		})
	}
}

// TestActivations tests the activation condition of the various upgrades.
func TestActivations(t *testing.T) {
	for _, test := range []struct {
		name           string
		setUpgradeTime func(t *uint64, c *Config)
		checkEnabled   func(t uint64, c *Config) bool
	}{
		{
			name: "Regolith",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.RegolithTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsRegolith(t)
			},
		},
		{
			name: "Canyon",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.CanyonTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsCanyon(t)
			},
		},
		{
			name: "Delta",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.DeltaTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsDelta(t)
			},
		},
		{
			name: "Ecotone",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.EcotoneTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsEcotone(t)
			},
		},
		{
			name: "Fjord",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.FjordTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsFjord(t)
			},
		},
		{
			name: "Granite",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.GraniteTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsGranite(t)
			},
		},
		{
			name: "Holocene",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.HoloceneTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsHolocene(t)
			},
		},
		{
			name: "Isthmus",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.IsthmusTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsIsthmus(t)
			},
		},
		{
			name: "Jovian",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.JovianTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsJovian(t)
			},
		},
		{
			name: "Interop",
			setUpgradeTime: func(t *uint64, c *Config) {
				c.InteropTime = t
			},
			checkEnabled: func(t uint64, c *Config) bool {
				return c.IsInterop(t)
			},
		},
	} {
		tt := test
		t.Run(fmt.Sprintf("TestActivations_%s", tt.name), func(t *testing.T) {
			config := randConfig()
			test.setUpgradeTime(nil, config)
			require.False(t, tt.checkEnabled(0, config), "false if nil time, even if checking 0")
			require.False(t, tt.checkEnabled(123456, config), "false if nil time")

			test.setUpgradeTime(new(uint64), config)
			require.True(t, tt.checkEnabled(0, config), "true at zero")
			require.True(t, tt.checkEnabled(123456, config), "true for any")

			x := uint64(123)
			test.setUpgradeTime(&x, config)
			require.False(t, tt.checkEnabled(0, config))
			require.False(t, tt.checkEnabled(122, config))
			require.True(t, tt.checkEnabled(123, config))
			require.True(t, tt.checkEnabled(124, config))
		})
	}
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
	err := config.ValidateL2Config(context.TODO(), &mockClient, false)
	assert.NoError(t, err)
}

func TestValidateL2ConfigInvalidChainIdFails(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(101)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x01}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient, false)
	assert.Error(t, err)
	config.L2ChainID = big.NewInt(99)
	err = config.ValidateL2Config(context.TODO(), &mockClient, false)
	assert.Error(t, err)
}

func TestValidateL2ConfigInvalidGenesisHashFails(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(100)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x00}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient, false)
	assert.Error(t, err)
	config.Genesis.L2.Hash = [32]byte{0x02}
	err = config.ValidateL2Config(context.TODO(), &mockClient, false)
	assert.Error(t, err)
}

func TestValidateL2ConfigInvalidGenesisHashSkippedWhenRequested(t *testing.T) {
	config := randConfig()
	config.L2ChainID = big.NewInt(100)
	config.Genesis.L2.Number = 100
	config.Genesis.L2.Hash = [32]byte{0x00}
	mockClient := mockL2Client{chainID: big.NewInt(100), Hash: common.Hash{0x01}}
	err := config.ValidateL2Config(context.TODO(), &mockClient, true)
	assert.NoError(t, err)
	config.Genesis.L2.Hash = [32]byte{0x02}
	err = config.ValidateL2Config(context.TODO(), &mockClient, true)
	assert.NoError(t, err)
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

func TestConfig_Check(t *testing.T) {
	tests := []struct {
		name        string
		modifier    func(cfg *Config)
		expectedErr error
	}{
		{
			name:        "BlockTimeZero",
			modifier:    func(cfg *Config) { cfg.BlockTime = 0 },
			expectedErr: ErrBlockTimeZero,
		},
		{
			name:        "ChannelTimeoutBedrockZero",
			modifier:    func(cfg *Config) { cfg.ChannelTimeoutBedrock = 0 },
			expectedErr: ErrMissingChannelTimeout,
		},
		{
			name:        "SeqWindowSizeZero",
			modifier:    func(cfg *Config) { cfg.SeqWindowSize = 0 },
			expectedErr: ErrInvalidSeqWindowSize,
		},
		{
			name:        "SeqWindowSizeOne",
			modifier:    func(cfg *Config) { cfg.SeqWindowSize = 1 },
			expectedErr: ErrInvalidSeqWindowSize,
		},
		{
			name:        "NoL1Genesis",
			modifier:    func(cfg *Config) { cfg.Genesis.L1.Hash = common.Hash{} },
			expectedErr: ErrMissingGenesisL1Hash,
		},
		{
			name:        "NoL2Genesis",
			modifier:    func(cfg *Config) { cfg.Genesis.L2.Hash = common.Hash{} },
			expectedErr: ErrMissingGenesisL2Hash,
		},
		{
			name:        "GenesisHashesEqual",
			modifier:    func(cfg *Config) { cfg.Genesis.L2.Hash = cfg.Genesis.L1.Hash },
			expectedErr: ErrGenesisHashesSame,
		},
		{
			name:        "GenesisL2TimeZero",
			modifier:    func(cfg *Config) { cfg.Genesis.L2Time = 0 },
			expectedErr: ErrMissingGenesisL2Time,
		},
		{
			name:        "NoBatcherAddr",
			modifier:    func(cfg *Config) { cfg.Genesis.SystemConfig.BatcherAddr = common.Address{} },
			expectedErr: ErrMissingBatcherAddr,
		},
		{
			name:        "NoScalar",
			modifier:    func(cfg *Config) { cfg.Genesis.SystemConfig.Scalar = eth.Bytes32{} },
			expectedErr: ErrMissingScalar,
		},
		{
			name:        "NoGasLimit",
			modifier:    func(cfg *Config) { cfg.Genesis.SystemConfig.GasLimit = 0 },
			expectedErr: ErrMissingGasLimit,
		},
		{
			name:        "NoBatchInboxAddress",
			modifier:    func(cfg *Config) { cfg.BatchInboxAddress = common.Address{} },
			expectedErr: ErrMissingBatchInboxAddress,
		},
		{
			name:        "NoDepositContractAddress",
			modifier:    func(cfg *Config) { cfg.DepositContractAddress = common.Address{} },
			expectedErr: ErrMissingDepositContractAddress,
		},
		{
			name:        "NoL1ChainId",
			modifier:    func(cfg *Config) { cfg.L1ChainID = nil },
			expectedErr: ErrMissingL1ChainID,
		},
		{
			name:        "NoL2ChainId",
			modifier:    func(cfg *Config) { cfg.L2ChainID = nil },
			expectedErr: ErrMissingL2ChainID,
		},
		{
			name:        "ChainIDsEqual",
			modifier:    func(cfg *Config) { cfg.L2ChainID = cfg.L1ChainID },
			expectedErr: ErrChainIDsSame,
		},
		{
			name:        "L1ChainIdNegative",
			modifier:    func(cfg *Config) { cfg.L1ChainID = big.NewInt(-1) },
			expectedErr: ErrL1ChainIDNotPositive,
		},
		{
			name:        "L1ChainIdZero",
			modifier:    func(cfg *Config) { cfg.L1ChainID = big.NewInt(0) },
			expectedErr: ErrL1ChainIDNotPositive,
		},
		{
			name:        "L2ChainIdNegative",
			modifier:    func(cfg *Config) { cfg.L2ChainID = big.NewInt(-1) },
			expectedErr: ErrL2ChainIDNotPositive,
		},
		{
			name:        "L2ChainIdZero",
			modifier:    func(cfg *Config) { cfg.L2ChainID = big.NewInt(0) },
			expectedErr: ErrL2ChainIDNotPositive,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := randConfig()
			test.modifier(cfg)
			err := cfg.Check()
			assert.ErrorIs(t, err, test.expectedErr)
		})
	}

	forkTests := []struct {
		name        string
		modifier    func(cfg *Config)
		expectedErr error
	}{
		{
			name: "PriorForkMissing",
			modifier: func(cfg *Config) {
				ecotoneTime := uint64(1)
				cfg.EcotoneTime = &ecotoneTime
			},
			expectedErr: fmt.Errorf("fork ecotone set (to 1), but prior fork delta missing"),
		},
		{
			name: "PriorForkHasHigherOffset",
			modifier: func(cfg *Config) {
				regolithTime := uint64(2)
				canyonTime := uint64(1)
				cfg.RegolithTime = &regolithTime
				cfg.CanyonTime = &canyonTime
			},
			expectedErr: fmt.Errorf("fork canyon set to 1, but prior fork regolith has higher offset 2"),
		},
		{
			name: "PriorForkOK",
			modifier: func(cfg *Config) {
				regolithTime := uint64(1)
				canyonTime := uint64(2)
				deltaTime := uint64(3)
				ecotoneTime := uint64(4)
				fjordTime := uint64(5)
				graniteTime := uint64(6)
				holoceneTime := uint64(7)
				isthmusTime := uint64(8)
				jovianTime := uint64(9)
				interopTime := uint64(10)
				cfg.RegolithTime = &regolithTime
				cfg.CanyonTime = &canyonTime
				cfg.DeltaTime = &deltaTime
				cfg.EcotoneTime = &ecotoneTime
				cfg.FjordTime = &fjordTime
				cfg.GraniteTime = &graniteTime
				cfg.HoloceneTime = &holoceneTime
				cfg.IsthmusTime = &isthmusTime
				cfg.JovianTime = &jovianTime
				cfg.InteropTime = &interopTime
			},
			expectedErr: nil,
		},
	}

	for _, test := range forkTests {
		t.Run(test.name, func(t *testing.T) {
			cfg := randConfig()
			test.modifier(cfg)
			err := cfg.Check()
			assert.Equal(t, err, test.expectedErr)
		})
	}
}

func TestTimestampForBlock(t *testing.T) {
	config := randConfig()

	tests := []struct {
		name              string
		genesisTime       uint64
		genesisBlock      uint64
		blockTime         uint64
		blockNum          uint64
		expectedBlockTime uint64
	}{
		{
			name:              "FirstBlock",
			genesisTime:       100,
			genesisBlock:      0,
			blockTime:         2,
			blockNum:          0,
			expectedBlockTime: 100,
		},
		{
			name:              "SecondBlock",
			genesisTime:       100,
			genesisBlock:      0,
			blockTime:         2,
			blockNum:          1,
			expectedBlockTime: 102,
		},
		{
			name:              "NBlock",
			genesisTime:       100,
			genesisBlock:      0,
			blockTime:         2,
			blockNum:          25,
			expectedBlockTime: 150,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("TestTimestampForBlock_%s", test.name), func(t *testing.T) {
			config.Genesis.L2Time = test.genesisTime
			config.Genesis.L2.Number = test.genesisBlock
			config.BlockTime = test.blockTime

			timestamp := config.TimestampForBlock(test.blockNum)
			assert.Equal(t, timestamp, test.expectedBlockTime)
		})
	}
}

func TestForkchoiceUpdatedVersion(t *testing.T) {
	config := randConfig()
	tests := []struct {
		name           string
		canyonTime     uint64
		ecotoneTime    uint64
		attrs          *eth.PayloadAttributes
		expectedMethod eth.EngineAPIMethod
	}{
		{
			name:           "NoAttrs",
			canyonTime:     10,
			ecotoneTime:    20,
			attrs:          nil,
			expectedMethod: eth.FCUV3,
		},
		{
			name:           "BeforeCanyon",
			canyonTime:     10,
			ecotoneTime:    20,
			attrs:          &eth.PayloadAttributes{Timestamp: 5},
			expectedMethod: eth.FCUV1,
		},
		{
			name:           "Canyon",
			canyonTime:     10,
			ecotoneTime:    20,
			attrs:          &eth.PayloadAttributes{Timestamp: 15},
			expectedMethod: eth.FCUV2,
		},
		{
			name:           "Ecotone",
			canyonTime:     10,
			ecotoneTime:    20,
			attrs:          &eth.PayloadAttributes{Timestamp: 25},
			expectedMethod: eth.FCUV3,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("TestForkchoiceUpdatedVersion_%s", test.name), func(t *testing.T) {
			config.CanyonTime = &test.canyonTime
			config.EcotoneTime = &test.ecotoneTime
			assert.Equal(t, config.ForkchoiceUpdatedVersion(test.attrs), test.expectedMethod)
		})
	}
}

func TestNewPayloadVersion(t *testing.T) {
	config := randConfig()
	canyonTime := uint64(0)
	config.CanyonTime = &canyonTime
	tests := []struct {
		name           string
		ecotoneTime    uint64
		isthmusTime    uint64
		payloadTime    uint64
		expectedMethod eth.EngineAPIMethod
	}{
		{
			name:           "BeforeEcotone",
			ecotoneTime:    10,
			payloadTime:    5,
			isthmusTime:    20,
			expectedMethod: eth.NewPayloadV2,
		},
		{
			name:           "Ecotone",
			ecotoneTime:    10,
			payloadTime:    15,
			isthmusTime:    20,
			expectedMethod: eth.NewPayloadV3,
		},
		{
			name:           "Isthmus",
			ecotoneTime:    10,
			payloadTime:    25,
			isthmusTime:    20,
			expectedMethod: eth.NewPayloadV4,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("TestNewPayloadVersion_%s", test.name), func(t *testing.T) {
			config.EcotoneTime = &test.ecotoneTime
			config.IsthmusTime = &test.isthmusTime
			assert.Equal(t, config.NewPayloadVersion(test.payloadTime), test.expectedMethod)
		})
	}
}

func TestGetPayloadVersion(t *testing.T) {
	config := randConfig()
	canyonTime := uint64(0)
	config.CanyonTime = &canyonTime
	tests := []struct {
		name           string
		isthmusTime    uint64
		ecotoneTime    uint64
		payloadTime    uint64
		expectedMethod eth.EngineAPIMethod
	}{
		{
			name:           "BeforeEcotone",
			ecotoneTime:    10,
			payloadTime:    5,
			isthmusTime:    20,
			expectedMethod: eth.GetPayloadV2,
		},
		{
			name:           "Ecotone",
			ecotoneTime:    10,
			payloadTime:    15,
			isthmusTime:    20,
			expectedMethod: eth.GetPayloadV3,
		},
		{
			name:           "Isthmus",
			ecotoneTime:    10,
			payloadTime:    25,
			isthmusTime:    20,
			expectedMethod: eth.GetPayloadV4,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("TestGetPayloadVersion_%s", test.name), func(t *testing.T) {
			config.EcotoneTime = &test.ecotoneTime
			config.IsthmusTime = &test.isthmusTime
			assert.Equal(t, config.GetPayloadVersion(test.payloadTime), test.expectedMethod)
		})
	}
}

func TestConfig_IsActivationBlock(t *testing.T) {
	// Map of fork names to their config field setters
	forks := []struct {
		name    ForkName
		setTime func(cfg *Config, ts uint64)
	}{
		{Canyon, func(cfg *Config, ts uint64) { cfg.CanyonTime = &ts }},
		{Delta, func(cfg *Config, ts uint64) { cfg.DeltaTime = &ts }},
		{Ecotone, func(cfg *Config, ts uint64) { cfg.EcotoneTime = &ts }},
		{Fjord, func(cfg *Config, ts uint64) { cfg.FjordTime = &ts }},
		{Granite, func(cfg *Config, ts uint64) { cfg.GraniteTime = &ts }},
		{Holocene, func(cfg *Config, ts uint64) { cfg.HoloceneTime = &ts }},
		{Isthmus, func(cfg *Config, ts uint64) { cfg.IsthmusTime = &ts }},
		{Interop, func(cfg *Config, ts uint64) { cfg.InteropTime = &ts }},
	}

	for _, fork := range forks {
		ts := uint64(100)
		cfg := &Config{}
		fork.setTime(cfg, ts)

		t.Run(string(fork.name), func(t *testing.T) {
			// Crossing the fork boundary should return the fork name
			require.Equal(t, fork.name, cfg.IsActivationBlock(ts-1, ts))
			require.Equal(t, fork.name, cfg.IsActivationBlock(ts-1, ts+10))
			// Not crossing the fork boundary should return None
			require.Equal(t, None, cfg.IsActivationBlock(ts, ts+1))
			require.Equal(t, None, cfg.IsActivationBlock(ts+1, ts+2))
			// Before the fork
			require.Equal(t, None, cfg.IsActivationBlock(ts-10, ts-1))
		})
	}
}

func TestConfigImplementsBlockType(t *testing.T) {
	config := randConfig()
	isthmusTime := uint64(100)
	config.IsthmusTime = &isthmusTime
	tests := []struct {
		name                       string
		blockTime                  uint64
		hasOptimismWithdrawalsRoot bool
	}{
		{
			name:                       "BeforeIsthmus",
			blockTime:                  uint64(99),
			hasOptimismWithdrawalsRoot: false,
		},
		{
			name:                       "AtIsthmus",
			blockTime:                  uint64(100),
			hasOptimismWithdrawalsRoot: true,
		},
		{
			name:                       "AfterIsthmus",
			blockTime:                  uint64(200),
			hasOptimismWithdrawalsRoot: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("TestHasOptimismWithdrawalsRoot_%s", test.name), func(t *testing.T) {
			assert.Equal(t, config.HasOptimismWithdrawalsRoot(test.blockTime), test.hasOptimismWithdrawalsRoot)
		})
	}
}

func TestConfig_ProbablyMissingPectraBlobSchedule(t *testing.T) {
	hol, sep := params.HoleskyChainConfig, params.SepoliaChainConfig

	for _, tt := range []struct {
		name                   string
		pectraBlobScheduleTime *uint64
		l2genesisTime          uint64
		l1chainID              *big.Int
		expMissing             bool
	}{
		{
			name:                   "sepolia-ok",
			pectraBlobScheduleTime: u64ptr(1742486400), // sepolia superchain
			l2genesisTime:          1691802540,         // op-sepolia
			l1chainID:              sep.ChainID,
		},
		{
			name:                   "holesky-ok",
			pectraBlobScheduleTime: u64ptr(1742486400), // sepolia superchain
			l2genesisTime:          1691802540,         // op-sepolia
			l1chainID:              hol.ChainID,
		},
		{
			name:          "sepolia-missing",
			l2genesisTime: 1691802540, // op-sepolia
			l1chainID:     sep.ChainID,
			expMissing:    true,
		},
		{
			name:          "holesky-missing",
			l2genesisTime: 1691802540, // op-sepolia
			l1chainID:     hol.ChainID,
			expMissing:    true,
		},
		{
			name:          "sepolia-young-genesis",
			l2genesisTime: *sep.PragueTime + 1,
			l1chainID:     sep.ChainID,
		},
		{
			name:          "holesky-young-genesis",
			l2genesisTime: *hol.PragueTime + 1,
			l1chainID:     hol.ChainID,
		},
		{
			name:          "other-missing-ok",
			l2genesisTime: 1691802540,
			l1chainID:     big.NewInt(1),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Genesis: Genesis{
					L2Time: tt.l2genesisTime,
				},
				PectraBlobScheduleTime: tt.pectraBlobScheduleTime,
				L1ChainID:              tt.l1chainID,
			}
			assert.Equal(t, tt.expMissing, cfg.ProbablyMissingPectraBlobSchedule())
		})
	}
}

// TestConfig_IsForkActive validates the generic IsForkActive API across forks.
func TestConfig_IsForkActive(t *testing.T) {
	for _, fork := range scheduleableForks {
		t.Run(string(fork), func(t *testing.T) {
			var cfg Config
			// nil activation time => always inactive
			require.False(t, cfg.IsForkActive(fork, 0))
			require.False(t, cfg.IsForkActive(fork, 123))

			// activation at genesis (0) => active for all timestamps
			zero := uint64(0)
			cfg.SetActivationTime(fork, &zero)
			require.True(t, cfg.IsForkActive(fork, 0))
			require.True(t, cfg.IsForkActive(fork, 999999))

			// activation at specific timestamp
			at := uint64(100)
			cfg.SetActivationTime(fork, &at)
			require.False(t, cfg.IsForkActive(fork, 99))
			require.True(t, cfg.IsForkActive(fork, 100))
			require.True(t, cfg.IsForkActive(fork, 101))
		})
	}
}

// TestConfig_ActivationBlockAndForFork combines tests for IsActivationBlock and IsActivationBlockForFork.
func TestConfig_ActivationBlockAndForFork(t *testing.T) {
	for _, fork := range scheduleableForks {
		t.Run(string(fork), func(t *testing.T) {
			cfg := Config{BlockTime: 2}
			ts := uint64(100)
			cfg.SetActivationTime(fork, &ts)

			// IsActivationBlock should detect boundary crossing irrespective of block time granularity
			require.Equal(t, fork, cfg.IsActivationBlock(ts-1, ts))
			require.Equal(t, fork, cfg.IsActivationBlock(ts-1, ts+10))
			require.Equal(t, None, cfg.IsActivationBlock(ts, ts+1))
			require.Equal(t, None, cfg.IsActivationBlock(ts+1, ts+2))

			// IsActivationBlockForFork should be true for the first block(s) at/after activation,
			// i.e. for times in [ts, ts+BlockTime-1], and false otherwise.
			bt := cfg.BlockTime
			require.Greater(t, bt, uint64(0))
			require.True(t, cfg.IsActivationBlockForFork(ts, fork))
			require.True(t, cfg.IsActivationBlockForFork(ts+(bt-1), fork))
			require.False(t, cfg.IsActivationBlockForFork(ts-1, fork))
			require.False(t, cfg.IsActivationBlockForFork(ts+bt, fork))
		})
	}
}

// TestConfig_ActivateAt validates ActivateAt behavior, including Bedrock special-case.
func TestConfig_ActivateAt(t *testing.T) {
	// Activate at Ecotone with a non-zero timestamp
	t.Run("Ecotone", func(t *testing.T) {
		var cfg Config
		ts := uint64(100)
		cfg.ActivateAt(Ecotone, ts)
		require.Equal(t, Config{
			RegolithTime: ptr.Zero64,
			CanyonTime:   ptr.Zero64,
			DeltaTime:    ptr.Zero64,
			EcotoneTime:  &ts,
		}, cfg)
	})

	t.Run("LatestFork", func(t *testing.T) {
		var cfg Config
		ts := uint64(100)
		cfg.ActivateAt(LatestFork, ts)
		for _, f := range scheduleableForks {
			at := cfg.ActivationTime(f)
			require.NotNil(t, at)
			if f == LatestFork {
				require.EqualValues(t, ts, *at)
			} else {
				// prior forks at genesis (0)
				require.Zero(t, *at)
			}
		}
	})

	// Bedrock special case: disable all scheduleable forks
	t.Run("Bedrock", func(t *testing.T) {
		var cfg Config
		cfg.ActivateAt(Bedrock, 0)
		for _, f := range scheduleableForks {
			require.Nil(t, cfg.ActivationTime(f))
		}
		require.Zero(t, cfg)
	})
}

// TestConfig_ActivateAtGenesis validates ActivateAtGenesis behavior.
func TestConfig_ActivateAtGenesis(t *testing.T) {
	// Activate Ecotone at genesis
	t.Run("Ecotone", func(t *testing.T) {
		var cfg Config
		cfg.ActivateAtGenesis(Ecotone)
		require.Equal(t, Config{
			RegolithTime: ptr.Zero64,
			CanyonTime:   ptr.Zero64,
			DeltaTime:    ptr.Zero64,
			EcotoneTime:  ptr.Zero64,
		}, cfg)
	})

	t.Run("LatestFork", func(t *testing.T) {
		var cfg Config
		cfg.ActivateAtGenesis(LatestFork)
		for _, f := range scheduleableForks {
			at := cfg.ActivationTime(f)
			require.NotNil(t, at)
			require.Zero(t, *at)
		}
	})

	// Bedrock special case: disable all scheduleable forks
	t.Run("Bedrock", func(t *testing.T) {
		var cfg Config
		cfg.ActivateAtGenesis(Bedrock)
		for _, f := range scheduleableForks {
			require.Nil(t, cfg.ActivationTime(f))
		}
		require.Zero(t, cfg)
	})
}
