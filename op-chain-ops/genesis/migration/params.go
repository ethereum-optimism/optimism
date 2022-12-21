package migration

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// Params contains the configuration parameters used for verifying
// the integrity of the migration.
type Params struct {
	// ExpectedSupplyDelta is the expected delta between the total supply of OVM ETH,
	// and ETH we were able to migrate. This is used to account for supply bugs in
	//previous regenesis events.
	ExpectedSupplyDelta *big.Int

	// IgnoredWithdrawalSlots is a map of storage slots to ignore while validation withdrawal
	// witness data. The witness data generation script does not take into account reverts,
	// so we need to ignore certain storage slots that don't exist in the state as a result
	// of a revert.
	IgnoredWithdrawalSlots map[common.Hash]bool
}

var ParamsByChainID = map[int]*Params{
	1: {
		// Regenesis 4 (Nov 11 2021) contained a supply bug such that the total OVM ETH
		// supply was 1.628470012 ETH greater than the sum balance of every account migrated
		// / during the regenesis. A further 0.0012 ETH was incorrectly not removed from the
		// total supply by accidental invocations of the Saurik bug (https://www.saurik.com/optimism.html).
		new(big.Int).SetUint64(1627270011999999992),

		map[common.Hash]bool{},
	},
	5: {
		new(big.Int),

		// The below reverted messages are current as of 12-21-2022. We may encounter more of these as
		// we get closer to the migration. Raw log output is includes below for validation purpose:
		//
		// slot=0xc954e2dce86e2a3108afa2c8a51605cfec4616b3844c319fb5ab34687c4432d0 nonce=106,121 target=0x636Af16bf2f682dD3109e60102b8E1A089FedAa8 sender=0x4200000000000000000000000000000000000010 data=1532ec3400000000000000000000000011ac2061d90349a9093557b664eb17c2641373e100000000000000000000000011ac2061d90349a9093557b664eb17c2641373e10000000000000000000000000000000000000000000000000de0b6b3a764000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000
		// slot=0x96aece98277986b1eb9daeac9f41ced7e3807954a6f1c9c8d28a431d2c9cfd0f nonce=110,285 target=0xf775c20445c058a1B5FfE9779B7E5405D6C251D9 sender=0xb58b8327cf95b6eDD13c731c8Db7C606C203f9f6 data=d80de97817e0f6859a3f931e5b28a484bccd495664ba6b3fe30ab7b06b0d942f39fd1246d20111a5e8dbd8fefe35a3acac7607c185066352687fb29e3591fdabf3b3b40e000000000000000000000000000000000000000000000000000001d1a94a2000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000638ce355
		// slot=0xeec05e97eb2bab6ed45f7b2ff16b361a6e88c84a06a6159c5bd5313a6d76ba44 nonce=110,032 target=0xf775c20445c058a1B5FfE9779B7E5405D6C251D9 sender=0xb58b8327cf95b6eDD13c731c8Db7C606C203f9f6 data=d80de97817e0f6859a3f931e5b28a484bccd495664ba6b3fe30ab7b06b0d942f39fd1246d20111a5e8dbd8fefe35a3acac7607c185066352687fb29e3591fdabf3b3b40e000000000000000000000000000000000000000000000000000001d1a94a2000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000638a44aa
		// slot=0x5ea5dac2edc32c171208d789677e234368006f234e7677b662979b451ba2d300 nonce=110,032 target=0xf775c20445c058a1B5FfE9779B7E5405D6C251D9 sender=0xb58b8327cf95b6eDD13c731c8Db7C606C203f9f6 data=d80de97817e0f6859a3f931e5b28a484bccd495664ba6b3fe30ab7b06b0d942f39fd1246d20111a5e8dbd8fefe35a3acac7607c185066352687fb29e3591fdabf3b3b40e000000000000000000000000000000000000000000000000000001d1a94a2000000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000638a460d
		map[common.Hash]bool{
			common.HexToHash("0xc954e2dce86e2a3108afa2c8a51605cfec4616b3844c319fb5ab34687c4432d0"): true,
			common.HexToHash("0x96aece98277986b1eb9daeac9f41ced7e3807954a6f1c9c8d28a431d2c9cfd0f"): true,
			common.HexToHash("0xeec05e97eb2bab6ed45f7b2ff16b361a6e88c84a06a6159c5bd5313a6d76ba44"): true,
			common.HexToHash("0x5ea5dac2edc32c171208d789677e234368006f234e7677b662979b451ba2d300"): true,
		},
	},
}
