package genesis

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/c2h5oh/datasize"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/chain"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon-lib/kv/mdbx"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/turbo/trie"
	"github.com/ledgerwatch/log/v3"
	"golang.org/x/exp/slices"
)

func NewGenesis(path string) (*types.Genesis, error) {
	// load genesis config
	genesisCfgFile, err := os.Open(path)
	if err != nil {
		log.Error("failed to open genesis config file", "err", err)
		return nil, err
	}
	defer genesisCfgFile.Close()

	genesis := new(types.Genesis)
	if err := json.NewDecoder(genesisCfgFile).Decode(genesis); err != nil {
		log.Error("failed to decode genesis config file", "err", err)
		return nil, err
	}
	return genesis, nil
}

// This middle layer is used to convert the genesis account format from geth to erigon
type LegacyGenesisAccount struct {
	Code    string                 `json:"code,omitempty"`
	Storage map[common.Hash]string `json:"storage,omitempty"`
	Nonce   uint64                 `json:"nonce,omitempty"`
}

func NewAlloc(path string) (*types.GenesisAlloc, error) {
	// load alloc file
	file, err := os.Open(path)
	if err != nil {
		log.Error("failed to open alloc file", "err", err)
		return nil, err
	}
	defer file.Close()

	bytes, _ := io.ReadAll(file)
	genesisAlloc, err := MigrateAlloc(bytes)
	if err != nil {
		log.Error("failed to migrate alloc", "err", err)
		return nil, err
	}
	return &genesisAlloc, nil
}

func MigrateAlloc(bytes []byte) (types.GenesisAlloc, error) {
	var legacyAlloc map[common.Address]LegacyGenesisAccount
	if err := json.Unmarshal(bytes, &legacyAlloc); err != nil {
		return nil, err
	}
	genesisAlloc := make(types.GenesisAlloc)
	for addr, account := range legacyAlloc {
		storage := make(map[common.Hash]common.Hash)
		for k, v := range account.Storage {
			storage[k] = common.HexToHash(v)
		}
		genesisAlloc[addr] = types.GenesisAccount{
			Code:    common.FromHex(account.Code),
			Balance: common.Big0,
			Nonce:   account.Nonce,
			Storage: storage,
		}
	}
	return genesisAlloc, nil
}

var genesisTmpDB kv.RwDB
var genesisDBLock sync.Mutex

// This function is from erigon/core/genesis_write.go
func AllocToGenesis(g *types.Genesis, head *types.Header) (*state.IntraBlockState, common.Hash, error) {
	var statedb *state.IntraBlockState
	wg := sync.WaitGroup{}
	wg.Add(1)

	var err error
	var root common.Hash

	go func() { // we may run inside write tx, can't open 2nd write tx in same goroutine
		// TODO(yperbasis): use memdb.MemoryMutation instead
		defer wg.Done()
		genesisDBLock.Lock()
		defer genesisDBLock.Unlock()
		if genesisTmpDB == nil {
			genesisTmpDB = mdbx.NewMDBX(log.New()).InMem("").MapSize(2 * datasize.GB).MustOpen()
		}
		var tx kv.RwTx
		if tx, err = genesisTmpDB.BeginRw(context.Background()); err != nil {
			return
		}
		defer tx.Rollback()
		r, w := state.NewDbStateReader(tx), state.NewDbStateWriter(tx, 0)
		statedb = state.New(r)

		hasConstructorAllocation := false
		for _, account := range g.Alloc {
			if len(account.Constructor) > 0 {
				hasConstructorAllocation = true
				break
			}
		}
		// See https://github.com/NethermindEth/nethermind/blob/master/src/Nethermind/Nethermind.Consensus.AuRa/InitializationSteps/LoadGenesisBlockAuRa.cs
		if hasConstructorAllocation && g.Config.Aura != nil {
			statedb.CreateAccount(common.Address{}, false)
		}

		keys := sortedAllocKeys(g.Alloc)
		for _, key := range keys {
			addr := common.BytesToAddress([]byte(key))
			account := g.Alloc[addr]

			var (
				balance  *uint256.Int
				overflow bool
			)
			if account.Balance == nil {
				balance = uint256.NewInt(0)
			} else {
				balance, overflow = uint256.FromBig(account.Balance)
				if overflow {
					panic("overflow at genesis allocs")
				}
			}

			statedb.AddBalance(addr, balance)
			statedb.SetCode(addr, account.Code)
			statedb.SetNonce(addr, account.Nonce)
			for key, value := range account.Storage {
				key := key
				val := uint256.NewInt(0).SetBytes(value.Bytes())
				statedb.SetState(addr, &key, *val)
			}

			if len(account.Constructor) > 0 {
				if _, err = core.SysCreate(addr, account.Constructor, *g.Config, statedb, head); err != nil {
					return
				}
			}

			if len(account.Code) > 0 || len(account.Storage) > 0 || len(account.Constructor) > 0 {
				statedb.SetIncarnation(addr, state.FirstContractIncarnation)
			}
		}

		// apply all the changes
		if err = statedb.FinalizeTx(&chain.Rules{}, w); err != nil {
			return
		}

		root, err = trie.CalcRoot("transition", tx)
		if err != nil {
			return
		}

	}()

	wg.Wait()

	if err != nil {
		return nil, common.Hash{}, err
	}

	return statedb, root, nil
}

func sortedAllocKeys(m types.GenesisAlloc) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = string(k.Bytes())
		i++
	}
	slices.Sort(keys)
	return keys
}
