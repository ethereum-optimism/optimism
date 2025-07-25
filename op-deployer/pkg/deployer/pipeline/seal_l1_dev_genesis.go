package pipeline

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func SealL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "seal-l1-dev-genesis")
	lgr.Info("Sealing L1 dev genesis")

	l1DevParams := intent.L1DevGenesisParams
	if l1DevParams == nil {
		env.Logger.Warn("Using dev L1 genesis without any customization")
		l1DevParams = &state.L1DevGenesisParams{}
	}
	// Create a state-dump
	dump, err := env.L1ScriptHost.StateDump()
	if err != nil {
		return fmt.Errorf("failed to dump L1 state: %w", err)
	}
	st.L1StateDump = &state.GzipData[foundry.ForgeAllocs]{
		Data: dump,
	}

	bp := &l1DevParams.BlockParams
	timestamp := bp.Timestamp
	if timestamp == 0 {
		timestamp = uint64(time.Now().Unix())
		env.Logger.Warn("Dynamically determined dev L1 genesis timestamp", "timestamp", timestamp)
	}
	excessBlobGas := bp.ExcessBlobGas

	// Create a genesis configuration template
	genesisTemplate, err := genesis.NewL1GenesisMinimal(&genesis.DevL1DeployConfigMinimal{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1GenesisBlockTimestamp:     hexutil.Uint64(timestamp),
			L1GenesisBlockGasLimit:      hexutil.Uint64(bp.GasLimit),
			L1GenesisBlockExcessBlobGas: (*hexutil.Uint64)(&excessBlobGas),
			// The rest is left to defaults
		},
		L1ChainID:          eth.ChainIDFromUInt64(intent.L1ChainID),
		L1PragueTimeOffset: l1DevParams.PragueTimeOffset,
	})
	if err != nil {
		return fmt.Errorf("failed to create dev L1 genesis template: %w", err)
	}
	// Combine the two into a valid genesis
	genesisTemplate.Alloc = dump.Accounts
	// Compute the genesis state root (by turning it into a block, which will apply the defaults, chain-config, etc.)
	l1GenesisBlock := genesisTemplate.ToBlock()
	// Cache the genesis state-root, and de-dup the state-copy we maintain, by using the state-hash attribute.
	h := l1GenesisBlock.Root()
	genesisTemplate.Alloc = nil
	genesisTemplate.StateHash = &h
	st.L1DevGenesis = genesisTemplate
	lgr.Info("Sealed L1 dev genesis", "blockHash", l1GenesisBlock.Hash(), "stateRoot", h)
	return nil
}
