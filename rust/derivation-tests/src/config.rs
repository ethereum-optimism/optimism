//! Deterministic configuration for derivation tests.
//!
//! Loads all configuration from the generated testdata files produced by
//! op-deployer. Every value is pinned — no randomness, no `Option` fields.
//! The same config always produces the same L1/L2 chains and super roots.

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, Bytes, U256, address, b256, hex};
use kona_genesis::RollupConfig;
use std::{collections::BTreeMap, path::Path, sync::Arc};

/// Deterministic batcher private key (NOT secret — test-only, Hardhat account #1).
pub const BATCHER_PRIVATE_KEY: B256 =
    b256!("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d");

/// Deterministic sequencer private key (NOT secret — test-only, Hardhat account #2).
pub const SEQUENCER_PRIVATE_KEY: B256 =
    b256!("0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a");

/// Pre-funded test account address.
pub const PREFUNDED_ACCOUNT: Address = address!("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266");

/// Pre-funded test account private key (NOT secret — test-only).
/// This is the first Hardhat/Anvil default account.
pub const PREFUNDED_ACCOUNT_KEY: B256 =
    b256!("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80");

/// `L2ToL1MessagePasser` predeploy address.
pub const L2_TO_L1_MESSAGE_PASSER: Address = address!("0x4200000000000000000000000000000000000016");

/// Fully deterministic test configuration loaded from generated files.
///
/// Every field is pinned — no `Option` values, no randomness.
/// Creating the same config twice produces identical chains and roots.
#[derive(Debug, Clone)]
pub struct DeterministicConfig {
    /// Full L2 genesis JSON.
    pub l2_genesis: serde_json::Value,
    /// Full L1 genesis JSON.
    pub l1_genesis: serde_json::Value,
    /// Parsed rollup config.
    rollup_cfg: RollupConfig,
    /// Raw rollup JSON bytes for passing to runners.
    pub rollup_json: Vec<u8>,
    /// Raw L1 genesis JSON bytes for passing to runners.
    pub l1_genesis_json: Vec<u8>,
    /// Raw L2 genesis JSON bytes for passing to runners.
    pub l2_genesis_json: Vec<u8>,
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
    /// Fee recipient for L2 blocks (coinbase from L2 genesis).
    pub fee_recipient: Address,
    /// Batch inbox address.
    pub batch_inbox: Address,
    /// Batcher address.
    pub batcher: Address,
    /// Batcher private key.
    pub batcher_key: B256,
    /// Sequencer address — derived from `SEQUENCER_PRIVATE_KEY`.
    pub sequencer: Address,
    /// Sequencer private key.
    pub sequencer_key: B256,
    /// `OptimismPortal` (deposit contract) proxy address on L1.
    pub deposit_contract: Address,
    /// `SystemConfig` proxy address on L1.
    pub system_config: Address,
    /// L1 genesis state root (computed by go-ethereum's `Genesis.ToBlock()`).
    pub l1_genesis_state_root: B256,
    /// L2 genesis state root (computed by go-ethereum's `Genesis.ToBlock()`).
    pub l2_genesis_state_root: B256,
}

impl Default for DeterministicConfig {
    fn default() -> Self {
        let dir = std::env::var("DERIVATION_TESTDATA")
            .unwrap_or_else(|_| format!("{}/testdata/generated", env!("CARGO_MANIFEST_DIR")));
        Self::from_genesis_dir(Path::new(&dir))
    }
}

impl DeterministicConfig {
    /// Load config from a directory containing `genesis.json`, `rollup.json`, and
    /// `l1-genesis.json`.
    pub fn from_genesis_dir(dir: &Path) -> Self {
        let l2_genesis_json =
            std::fs::read(dir.join("genesis.json")).expect("failed to read genesis.json");
        let l1_genesis_json =
            std::fs::read(dir.join("l1-genesis.json")).expect("failed to read l1-genesis.json");
        let rollup_json =
            std::fs::read(dir.join("rollup.json")).expect("failed to read rollup.json");
        let hashes_json = std::fs::read(dir.join("genesis-hashes.json"))
            .expect("failed to read genesis-hashes.json");
        let hashes: serde_json::Value =
            serde_json::from_slice(&hashes_json).expect("failed to parse genesis-hashes.json");

        let l2_genesis: serde_json::Value =
            serde_json::from_slice(&l2_genesis_json).expect("failed to parse genesis.json");
        let l1_genesis: serde_json::Value =
            serde_json::from_slice(&l1_genesis_json).expect("failed to parse l1-genesis.json");
        let rollup_cfg: RollupConfig =
            serde_json::from_slice(&rollup_json).expect("failed to parse rollup.json");

        let l1_chain_id = rollup_cfg.l1_chain_id;
        let l2_chain_id = rollup_cfg.l2_chain_id.id();
        let genesis_timestamp = rollup_cfg.genesis.l2_time;
        let l2_block_time = rollup_cfg.block_time;
        // L1 block time and seconds_per_slot are always 12s for our test chains
        let l1_block_time = 12;
        let seconds_per_slot = 12;

        let fee_recipient = parse_address_field(&l2_genesis, "coinbase");
        let batch_inbox = rollup_cfg.batch_inbox_address;
        let deposit_contract = rollup_cfg.deposit_contract_address;
        let system_config = rollup_cfg.l1_system_config_address;

        let batcher = rollup_cfg
            .genesis
            .system_config
            .as_ref()
            .expect("rollup config must have genesis system config")
            .batcher_address;

        let batcher_key = BATCHER_PRIVATE_KEY;
        let sequencer_key = SEQUENCER_PRIVATE_KEY;
        let sequencer = address!("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC");

        // Load pre-computed genesis state roots from genesis-hashes.json.
        // These are computed by go-ethereum's Genesis.ToBlock() and are authoritative.
        let l1_genesis_state_root: B256 = hashes["l1"]["stateRoot"]
            .as_str()
            .expect("missing l1.stateRoot")
            .parse()
            .expect("invalid l1 state root");
        let l2_genesis_state_root: B256 = hashes["l2"]["stateRoot"]
            .as_str()
            .expect("missing l2.stateRoot")
            .parse()
            .expect("invalid l2 state root");

        Self {
            l2_genesis,
            l1_genesis,
            rollup_cfg,
            rollup_json,
            l1_genesis_json,
            l2_genesis_json,
            l1_chain_id,
            l2_chain_id,
            genesis_timestamp,
            l1_block_time,
            l2_block_time,
            seconds_per_slot,
            fee_recipient,
            batch_inbox,
            batcher,
            batcher_key,
            sequencer,
            sequencer_key,
            deposit_contract,
            system_config,
            l1_genesis_state_root,
            l2_genesis_state_root,
        }
    }

    /// Returns the parsed rollup config.
    pub const fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_cfg
    }

    /// Extract the L1 chain config from the L1 genesis JSON.
    pub fn l1_chain_config(&self) -> alloy_genesis::ChainConfig {
        let config = &self.l1_genesis["config"];
        serde_json::from_value(config.clone()).expect("failed to parse L1 chain config")
    }

    /// Build a reth `ChainSpec` from the L1 genesis JSON for use with `EthEvmConfig`.
    pub fn l1_chain_spec(&self) -> Arc<reth_chainspec::ChainSpec> {
        let genesis: alloy_genesis::Genesis =
            serde_json::from_value(self.l1_genesis.clone()).expect("L1 genesis must parse");
        Arc::new(genesis.into())
    }

    /// Extract the L2 genesis allocs as a map.
    pub fn l2_genesis_allocs(&self) -> BTreeMap<Address, GenesisAccount> {
        parse_genesis_allocs(&self.l2_genesis)
    }

    /// Extract the L1 genesis allocs as a map.
    pub fn l1_genesis_allocs(&self) -> BTreeMap<Address, GenesisAccount> {
        parse_genesis_allocs(&self.l1_genesis)
    }

    /// Extract L2 genesis header fields from the L2 genesis JSON.
    ///
    /// Returns (`timestamp`, `gas_limit`, `base_fee`, `extra_data`, `coinbase`, `difficulty`,
    /// `mix_hash`).
    pub fn l2_genesis_header_fields(&self) -> (u64, u64, u64, Bytes, Address, U256, B256) {
        let g = &self.l2_genesis;
        let timestamp = parse_u64_field(g, "timestamp");
        let gas_limit = parse_u64_field(g, "gasLimit");
        let base_fee = parse_u64_field(g, "baseFeePerGas");
        let extra_data = parse_bytes_field(g, "extraData");
        let coinbase = parse_address_field(g, "coinbase");
        let difficulty = parse_u256_field(g, "difficulty");
        let mix_hash = parse_b256_field(g, "mixHash");
        (timestamp, gas_limit, base_fee, extra_data, coinbase, difficulty, mix_hash)
    }

    /// Extract L1 genesis header fields from the L1 genesis JSON.
    ///
    /// Returns (`timestamp`, `gas_limit`, `base_fee`, `extra_data`, `coinbase`, `difficulty`,
    /// `mix_hash`).
    pub fn l1_genesis_header_fields(&self) -> (u64, u64, u64, Bytes, Address, U256, B256) {
        let g = &self.l1_genesis;
        let timestamp = parse_u64_field(g, "timestamp");
        let gas_limit = parse_u64_field(g, "gasLimit");
        let base_fee = parse_u64_field(g, "baseFeePerGas");
        let extra_data = parse_bytes_field(g, "extraData");
        let coinbase = parse_address_field(g, "coinbase");
        let difficulty = parse_u256_field(g, "difficulty");
        let mix_hash = parse_b256_field(g, "mixHash");
        (timestamp, gas_limit, base_fee, extra_data, coinbase, difficulty, mix_hash)
    }

    /// Check if the L2 genesis JSON has a `blobGasUsed` field.
    pub fn l2_genesis_has_blob_gas_used(&self) -> bool {
        self.l2_genesis.get("blobGasUsed").and_then(|v| v.as_str()).is_some()
    }

    /// Check if the L1 genesis JSON has a `blobGasUsed` field.
    pub fn l1_genesis_has_blob_gas_used(&self) -> bool {
        self.l1_genesis.get("blobGasUsed").and_then(|v| v.as_str()).is_some()
    }

    /// Get the L2 genesis excess blob gas (0 if present in JSON).
    pub fn l2_genesis_excess_blob_gas(&self) -> Option<u64> {
        self.l2_genesis
            .get("excessBlobGas")
            .and_then(|v| v.as_str())
            .map(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).unwrap_or(0))
    }

    /// Get the L1 genesis excess blob gas.
    pub fn l1_genesis_excess_blob_gas(&self) -> Option<u64> {
        self.l1_genesis
            .get("excessBlobGas")
            .and_then(|v| v.as_str())
            .map(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).unwrap_or(0))
    }

    /// Get the EIP-1559 denominator from the `chain_op_config`.
    pub const fn eip1559_denominator(&self) -> u32 {
        self.rollup_cfg.chain_op_config.eip1559_denominator as u32
    }

    /// Get the EIP-1559 elasticity from the `chain_op_config`.
    pub const fn eip1559_elasticity(&self) -> u32 {
        self.rollup_cfg.chain_op_config.eip1559_elasticity as u32
    }

    /// Get the minimum base fee from the rollup config's genesis system config.
    pub fn min_base_fee(&self) -> u64 {
        self.rollup_cfg.genesis.system_config.as_ref().and_then(|sc| sc.min_base_fee).unwrap_or(0)
    }
}

/// Parse genesis allocs from a genesis JSON value.
fn parse_genesis_allocs(genesis: &serde_json::Value) -> BTreeMap<Address, GenesisAccount> {
    let alloc = genesis["alloc"].as_object().expect("genesis JSON must have alloc object");

    let mut result = BTreeMap::new();
    for (addr_hex, value) in alloc {
        let addr: Address = addr_hex
            .parse()
            .unwrap_or_else(|e| panic!("failed to parse alloc address {addr_hex}: {e}"));

        let balance = value.get("balance").and_then(|v| v.as_str()).map_or(U256::ZERO, |s| {
            s.strip_prefix("0x").map_or_else(
                || s.parse().unwrap_or(U256::ZERO),
                |stripped| U256::from_str_radix(stripped, 16).unwrap_or(U256::ZERO),
            )
        });

        let nonce = value
            .get("nonce")
            .and_then(|v| v.as_str())
            .map_or(0u64, |s| u64::from_str_radix(s.trim_start_matches("0x"), 16).unwrap_or(0));

        let code = value.get("code").and_then(|v| v.as_str()).filter(|s| s.len() > 2).map(|s| {
            Bytes::from(hex::decode(s.trim_start_matches("0x")).expect("invalid code hex"))
        });

        let storage = value.get("storage").and_then(|v| v.as_object()).map(|obj| {
            obj.iter()
                .map(|(k, v)| {
                    let slot: B256 = k.parse().expect("invalid storage key");
                    let val: B256 = v
                        .as_str()
                        .expect("storage value must be string")
                        .parse()
                        .expect("invalid storage value");
                    (slot, val)
                })
                .collect::<BTreeMap<_, _>>()
        });

        let nonce_opt = (nonce > 0).then_some(nonce);
        let mut account = GenesisAccount::default().with_balance(balance).with_nonce(nonce_opt);
        if let Some(code) = code {
            account.code = Some(code);
        }
        if let Some(storage) = storage {
            account.storage = Some(storage);
        }
        result.insert(addr, account);
    }

    result
}

fn parse_u64_field(json: &serde_json::Value, field: &str) -> u64 {
    json.get(field)
        .and_then(|v| v.as_str())
        .map(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).unwrap_or(0))
        .unwrap_or(0)
}

fn parse_u256_field(json: &serde_json::Value, field: &str) -> U256 {
    json.get(field)
        .and_then(|v| v.as_str())
        .map(|s| {
            s.strip_prefix("0x").map_or_else(
                || s.parse().unwrap_or(U256::ZERO),
                |stripped| U256::from_str_radix(stripped, 16).unwrap_or(U256::ZERO),
            )
        })
        .unwrap_or(U256::ZERO)
}

fn parse_b256_field(json: &serde_json::Value, field: &str) -> B256 {
    json.get(field)
        .and_then(|v| v.as_str())
        .map(|s| s.parse().unwrap_or(B256::ZERO))
        .unwrap_or(B256::ZERO)
}

fn parse_address_field(json: &serde_json::Value, field: &str) -> Address {
    json.get(field)
        .and_then(|v| v.as_str())
        .map(|s| s.parse().unwrap_or(Address::ZERO))
        .unwrap_or(Address::ZERO)
}

fn parse_bytes_field(json: &serde_json::Value, field: &str) -> Bytes {
    json.get(field)
        .and_then(|v| v.as_str())
        .filter(|s| s.len() > 2)
        .map(|s| {
            Bytes::from(hex::decode(s.trim_start_matches("0x")).expect("invalid hex in field"))
        })
        .unwrap_or_default()
}

/// Build Holocene-formatted extra data for L2 block headers.
///
/// Format: `[version_byte, denominator_be32, elasticity_be32]` (9 bytes total).
/// Go's `DecodeHoloceneExtraData` requires exactly 9 bytes.
pub fn holocene_extra_data(denominator: u32, elasticity: u32) -> Bytes {
    let mut extra = [0u8; 9];
    extra[1..5].copy_from_slice(&denominator.to_be_bytes());
    extra[5..9].copy_from_slice(&elasticity.to_be_bytes());
    Bytes::from(extra.to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_is_deterministic() {
        let a = DeterministicConfig::default();
        let b = DeterministicConfig::default();
        // Same rollup config hashes
        let hash_a = serde_json::to_vec(a.rollup_config()).unwrap();
        let hash_b = serde_json::to_vec(b.rollup_config()).unwrap();
        assert_eq!(hash_a, hash_b, "two loads should produce identical rollup configs");
    }

    #[test]
    fn generated_files_load_successfully() {
        let config = DeterministicConfig::default();
        assert_eq!(config.l1_chain_id, 900);
        assert_eq!(config.l2_chain_id, 901);
        assert_eq!(config.genesis_timestamp, 1_700_000_000);
        assert_eq!(config.l2_block_time, 2);
        assert_eq!(config.l1_block_time, 12);
    }

    #[test]
    fn rollup_config_loaded_correctly() {
        let config = DeterministicConfig::default();
        let rollup = config.rollup_config();
        assert_eq!(rollup.l1_chain_id, 900);
        assert_eq!(rollup.l2_chain_id, alloy_chains::Chain::from_id(901));
        assert_eq!(rollup.block_time, 2);
        assert!(rollup.hardforks.ecotone_time.is_some());
        assert!(rollup.hardforks.isthmus_time.is_some());
    }

    #[test]
    fn l2_genesis_allocs_are_populated() {
        let config = DeterministicConfig::default();
        let allocs = config.l2_genesis_allocs();
        assert!(
            allocs.len() > 100,
            "expected many allocs from op-deployer genesis, got {}",
            allocs.len()
        );
        assert!(
            allocs.contains_key(&L2_TO_L1_MESSAGE_PASSER),
            "should contain L2ToL1MessagePasser"
        );
    }

    #[test]
    fn l1_chain_config_parses() {
        let config = DeterministicConfig::default();
        let chain_config = config.l1_chain_config();
        assert_eq!(chain_config.chain_id, 900);
        assert_eq!(chain_config.shanghai_time, Some(0));
        assert_eq!(chain_config.cancun_time, Some(0));
    }

    #[test]
    fn l1_genesis_allocs_are_populated() {
        let config = DeterministicConfig::default();
        let allocs = config.l1_genesis_allocs();
        assert!(
            allocs.len() > 10,
            "expected many allocs from op-deployer L1 genesis, got {}",
            allocs.len()
        );
    }

    #[test]
    fn l1_chain_spec_parses() {
        let config = DeterministicConfig::default();
        let chain_spec = config.l1_chain_spec();
        assert_eq!(chain_spec.chain().id(), 900);
    }
}
