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
	dump := env.Forge.Genesis.Allocs.Copy()
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
		L1ChainID:             eth.ChainIDFromUInt64(intent.L1ChainID),
		L1PragueTimeOffset:    l1DevParams.PragueTimeOffset,
		L1OsakaTimeOffset:     l1DevParams.OsakaTimeOffset,
		L1BPO1TimeOffset:      l1DevParams.BPO1TimeOffset,
		L1BPO2TimeOffset:      l1DevParams.BPO2TimeOffset,
		L1BPO3TimeOffset:      l1DevParams.BPO3TimeOffset,
		L1BPO4TimeOffset:      l1DevParams.BPO4TimeOffset,
		L1BPO5TimeOffset:      l1DevParams.BPO5TimeOffset,
		L1AmsterdamTimeOffset: l1DevParams.AmsterdamTimeOffset,
		BlobScheduleConfig:    l1DevParams.BlobSchedule,
	})
	if err != nil {
		return fmt.Errorf("failed to create dev L1 genesis template: %w", err)
	}
	genesisTemplate.Alloc = dump.Accounts
	l1GenesisBlock := genesisTemplate.ToBlock()
	st.L1DevGenesis = genesisTemplate
	lgr.Info("Sealed L1 dev genesis", "blockHash", l1GenesisBlock.Hash(), "stateRoot", l1GenesisBlock.Root())
	return nil
}
