package actions

import (
	"bytes"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/holiman/uint256"
)

func collectAllocAddrs(alloc core.GenesisAlloc) []common.Address {
	var out []common.Address
	for addr := range alloc {
		out = append(out, addr)
	}
	// make output deterministic
	sort.Slice(out, func(i, j int) bool {
		return bytes.Compare(out[i][:], out[j][:]) < 0
	})
	return out
}

// MaskAddress emulates the address masking that happens in the "L2 alias of L1 contract" in L2 deposits.
// unaudited, use for testing only
func MaskAddress(address common.Address) common.Address {
	maskAddr := common.Address{0: 0x11, 1: 0x11, 18: 0x11, 19: 0x11}
	var mask uint256.Int
	mask.SetBytes20(maskAddr[:])
	var addr uint256.Int
	addr.SetBytes20(maskAddr[:])
	var out uint256.Int
	out.Add(&mask, &addr)
	return out.Bytes20()
}

// CollectAddresses constructs a lists of addresses that may be used as fuzzing corpora
// or random address selection.
func CollectAddresses(sd *SetupData, dp *DeployParams) (out []common.Address) {
	// This should be seeded with:
	//  - reserve 0 for selecting nil (contract creation)
	out = append(out, common.Address{})
	//  - zero address
	out = append(out, common.Address{})
	//  - addresses of signing accounts
	out = append(out, dp.Addresses.Batcher,
		dp.Addresses.Deployer,
		dp.Addresses.Proposer,
		dp.Addresses.Batcher,
		dp.Addresses.SequencerP2P,
		dp.Addresses.Alice,
		dp.Addresses.Bob,
		dp.Addresses.Mallory)
	// prefunded L1/L2 accounts for testing
	out = append(out, collectAllocAddrs(sd.L1Cfg.Alloc)...)
	out = append(out, collectAllocAddrs(sd.L2Cfg.Alloc)...)

	//  - addresses of system contracts
	out = append(out,
		sd.L1Cfg.Coinbase,
		sd.L2Cfg.Coinbase,
		sd.RollupCfg.P2PSequencerAddress,
		sd.RollupCfg.FeeRecipientAddress,
		sd.RollupCfg.BatchInboxAddress,
		sd.RollupCfg.BatchSenderAddress,
		sd.RollupCfg.DepositContractAddress,
	)
	//  - precompiles
	for i := 0; i <= 0xff; i++ {
		out = append(out, common.Address{19: byte(i)})
	}
	//  - masked L2 version of all the above
	unmasked := out[:]
	for _, addr := range unmasked {
		masked := MaskAddress(addr)
		out = append(out, masked)
	}
	return out
}
