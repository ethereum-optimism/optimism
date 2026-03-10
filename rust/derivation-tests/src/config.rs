//! Deterministic configuration for derivation tests.
//!
//! Every value is pinned — no randomness, no `Option` fields. The same config
//! always produces the same L1/L2 chains and super roots.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, U256, address, b256};
use kona_genesis::{ChainGenesis, HardForkConfig, RollupConfig, SystemConfig};
use std::collections::BTreeMap;

/// Fixed L1 chain ID for test chains.
pub const L1_CHAIN_ID: u64 = 900;

/// Fixed L2 chain ID for test chains.
pub const L2_CHAIN_ID: u64 = 901;

/// Fixed L1/L2 genesis timestamp (seconds since epoch).
/// Also used as the beacon genesis time.
pub const GENESIS_TIMESTAMP: u64 = 1_700_000_000;

/// Fixed L1 block time in seconds.
pub const L1_BLOCK_TIME: u64 = 12;

/// Fixed L2 block time in seconds.
pub const L2_BLOCK_TIME: u64 = 2;

/// Beacon seconds per slot — matches L1 block time.
pub const SECONDS_PER_SLOT: u64 = 12;

/// Batch inbox address where batcher submits batches.
pub const BATCH_INBOX_ADDRESS: Address = address!("0xff00000000000000000000000000000000000901");

/// Batcher address (sender of batch transactions).
pub const BATCHER_ADDRESS: Address = address!("0x3100000000000000000000000000000000000001");

/// Sequencer address (block builder / fee recipient).
pub const SEQUENCER_ADDRESS: Address = address!("0x3200000000000000000000000000000000000001");

/// Fee recipient for L2 blocks.
pub const FEE_RECIPIENT: Address = address!("0x4200000000000000000000000000000000000011");

/// `L2ToL1MessagePasser` predeploy address.
pub const L2_TO_L1_MESSAGE_PASSER: Address = address!("0x4200000000000000000000000000000000000016");

/// `L1Block` predeploy address.
pub const L1_BLOCK_ADDRESS: Address = address!("0x4200000000000000000000000000000000000015");

/// `SystemConfig` proxy address on L1.
pub const SYSTEM_CONFIG_ADDRESS: Address = address!("0x6900000000000000000000000000000000000001");

/// Deterministic batcher private key (NOT secret — test-only).
pub const BATCHER_PRIVATE_KEY: B256 =
    b256!("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");

/// Deterministic sequencer private key (NOT secret — test-only).
pub const SEQUENCER_PRIVATE_KEY: B256 =
    b256!("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d");

/// Pre-funded test account address.
pub const PREFUNDED_ACCOUNT: Address = address!("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266");

/// Pre-funded test account private key (NOT secret — test-only).
/// This is the first Hardhat/Anvil default account.
pub const PREFUNDED_ACCOUNT_KEY: B256 =
    b256!("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");

/// Pre-funded balance: 10,000 ETH in wei.
pub const PREFUNDED_BALANCE: U256 = U256::from_limbs([
    0x21E1_9E0C_9BAB_2400_0000_u128 as u64,
    (0x21E1_9E0C_9BAB_2400_0000_u128 >> 64) as u64,
    0,
    0,
]);

/// Fully deterministic test configuration.
///
/// Every field is pinned — no `Option` values, no randomness.
/// Creating the same config twice produces identical chains and roots.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DeterministicConfig {
    /// L1 chain ID.
    pub l1_chain_id: u64,
    /// L2 chain ID.
    pub l2_chain_id: u64,
    /// Genesis timestamp (seconds since epoch). Also beacon genesis time.
    pub genesis_timestamp: u64,
    /// L1 block time in seconds.
    pub l1_block_time: u64,
    /// L2 block time in seconds.
    pub l2_block_time: u64,
    /// Beacon seconds per slot.
    pub seconds_per_slot: u64,
    /// Batch inbox address.
    pub batch_inbox: Address,
    /// Batcher address.
    pub batcher: Address,
    /// Batcher private key.
    pub batcher_key: B256,
    /// Sequencer address.
    pub sequencer: Address,
    /// Sequencer private key.
    pub sequencer_key: B256,
    /// Fee recipient for L2 blocks.
    pub fee_recipient: Address,
    /// `SystemConfig` proxy address on L1.
    pub system_config: Address,
    /// Hardfork activation config.
    pub hardforks: HardForkConfig,
}

impl Default for DeterministicConfig {
    fn default() -> Self {
        Self {
            l1_chain_id: L1_CHAIN_ID,
            l2_chain_id: L2_CHAIN_ID,
            genesis_timestamp: GENESIS_TIMESTAMP,
            l1_block_time: L1_BLOCK_TIME,
            l2_block_time: L2_BLOCK_TIME,
            seconds_per_slot: SECONDS_PER_SLOT,
            batch_inbox: BATCH_INBOX_ADDRESS,
            batcher: BATCHER_ADDRESS,
            batcher_key: BATCHER_PRIVATE_KEY,
            sequencer: SEQUENCER_ADDRESS,
            sequencer_key: SEQUENCER_PRIVATE_KEY,
            fee_recipient: FEE_RECIPIENT,
            system_config: SYSTEM_CONFIG_ADDRESS,
            hardforks: HardForkConfig {
                regolith_time: Some(0),
                canyon_time: Some(0),
                delta_time: Some(0),
                ecotone_time: Some(0),
                fjord_time: Some(0),
                granite_time: Some(0),
                holocene_time: Some(0),
                isthmus_time: Some(0),
                ..Default::default()
            },
        }
    }
}

impl DeterministicConfig {
    /// Create a config with specific hardfork settings.
    pub fn with_hardforks(hardforks: HardForkConfig) -> Self {
        Self { hardforks, ..Self::default() }
    }

    /// Build the L2 genesis account allocations.
    ///
    /// Includes predeploy contracts and pre-funded test accounts.
    pub fn l2_genesis_allocs(&self) -> BTreeMap<Address, GenesisAccount> {
        let mut allocs = BTreeMap::new();

        // Pre-funded test account
        allocs.insert(PREFUNDED_ACCOUNT, GenesisAccount::default().with_balance(PREFUNDED_BALANCE));

        // L2ToL1MessagePasser predeploy — empty contract with storage
        allocs.insert(L2_TO_L1_MESSAGE_PASSER, GenesisAccount::default().with_balance(U256::ZERO));

        // L1Block predeploy — empty contract
        allocs.insert(L1_BLOCK_ADDRESS, GenesisAccount::default().with_balance(U256::ZERO));

        allocs
    }

    /// Build the OP Stack `RollupConfig` from this deterministic config.
    ///
    /// Genesis L1/L2 block references are computed deterministically from the config
    /// so they match the blocks produced by `L1ChainBuilder` and `L2ChainBuilder`.
    pub fn rollup_config(&self) -> RollupConfig {
        use alloy_consensus::Header;
        use alloy_eips::eip1898::BlockNumHash;
        use alloy_primitives::Sealable;

        use crate::state::roots::EMPTY_ROOT_HASH;

        // Compute the L1 genesis block hash deterministically (must match L1ChainBuilder::new)
        let l1_genesis_header = Header {
            number: 0,
            timestamp: self.genesis_timestamp,
            state_root: EMPTY_ROOT_HASH,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            withdrawals_root: Some(EMPTY_ROOT_HASH),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let l1_genesis_hash = l1_genesis_header.seal_slow().hash();

        // Compute the L2 genesis state root and block hash deterministically
        // (must match L2ChainBuilder::new)
        let mut state = crate::state::TestStateDb::new();
        state.init_genesis(&self.l2_genesis_allocs());
        let genesis_state_root = state.snapshot().state_root;

        let l2_genesis_header = Header {
            number: 0,
            timestamp: self.genesis_timestamp,
            state_root: genesis_state_root,
            transactions_root: EMPTY_ROOT_HASH,
            receipts_root: EMPTY_ROOT_HASH,
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let l2_genesis_hash = l2_genesis_header.seal_slow().hash();

        RollupConfig {
            genesis: ChainGenesis {
                l1: BlockNumHash { number: 0, hash: l1_genesis_hash },
                l2: BlockNumHash { number: 0, hash: l2_genesis_hash },
                l2_time: self.genesis_timestamp,
                system_config: Some(SystemConfig {
                    batcher_address: self.batcher,
                    gas_limit: 30_000_000,
                    ..Default::default()
                }),
            },
            l1_chain_id: self.l1_chain_id,
            l2_chain_id: alloy_chains::Chain::from_id(self.l2_chain_id),
            block_time: self.l2_block_time,
            seq_window_size: 3600,
            max_sequencer_drift: 600,
            channel_timeout: 300,
            batch_inbox_address: self.batch_inbox,
            l1_system_config_address: self.system_config,
            hardforks: self.hardforks,
            ..Default::default()
        }
    }

    /// Build an L1 chain config (alloy `ChainConfig`) for kona-host's `--l1-config-path`.
    pub fn l1_chain_config(&self) -> alloy_genesis::ChainConfig {
        alloy_genesis::ChainConfig {
            chain_id: self.l1_chain_id,
            // Enable post-merge (Shanghai+) so withdrawals_root is expected
            shanghai_time: Some(0),
            cancun_time: Some(0),
            ..Default::default()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_is_deterministic() {
        let a = DeterministicConfig::default();
        let b = DeterministicConfig::default();
        assert_eq!(a, b);
    }

    #[test]
    fn config_has_no_options() {
        let config = DeterministicConfig::default();
        assert_eq!(config.l1_chain_id, L1_CHAIN_ID);
        assert_eq!(config.l2_chain_id, L2_CHAIN_ID);
        assert_eq!(config.genesis_timestamp, GENESIS_TIMESTAMP);
        assert_eq!(config.l1_block_time, L1_BLOCK_TIME);
        assert_eq!(config.l2_block_time, L2_BLOCK_TIME);
        assert_eq!(config.seconds_per_slot, SECONDS_PER_SLOT);
    }

    #[test]
    fn hardfork_config_all_active() {
        let config = DeterministicConfig::default();
        assert_eq!(config.hardforks.regolith_time, Some(0));
        assert_eq!(config.hardforks.isthmus_time, Some(0));
    }

    #[test]
    fn genesis_allocs_include_prefunded_account() {
        let config = DeterministicConfig::default();
        let allocs = config.l2_genesis_allocs();
        assert!(allocs.contains_key(&PREFUNDED_ACCOUNT));
        assert!(allocs.contains_key(&L2_TO_L1_MESSAGE_PASSER));
        assert!(allocs.contains_key(&L1_BLOCK_ADDRESS));
    }

    #[test]
    fn rollup_config_has_hardforks() {
        let config = DeterministicConfig::default();
        let rollup = config.rollup_config();
        assert_eq!(rollup.l2_chain_id, alloy_chains::Chain::from_id(L2_CHAIN_ID));
        assert_eq!(rollup.block_time, L2_BLOCK_TIME);
        assert!(rollup.hardforks.ecotone_time.is_some());
        assert!(rollup.hardforks.isthmus_time.is_some());
    }
}
