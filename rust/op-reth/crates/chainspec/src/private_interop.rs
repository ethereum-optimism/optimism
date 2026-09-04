//! Deterministic private-chain genesis to public-projection genesis transformation.
//!
//! The source is a pinned op-deployer genesis or configured private ETH profile with interop
//! active at genesis. A custom-gas-token
//! source is supported and its CGT machinery is stripped; an ordinary ETH source is supported too,
//! and then there is nothing to strip. WHICH chain is the private one is explicit runtime
//! configuration, not something the validator can infer. This
//! is the Rust mirror of `op-private-interop/genesis/projection.go`; the two must project the
//! shared fixture to the same block hash (see the tests).

use alloc::collections::BTreeMap;
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, U256, address, b256, keccak256};
use core::{fmt, str::FromStr};

const L1_BLOCK: Address = address!("4200000000000000000000000000000000000015");
const L2_TO_L1_MESSAGE_PASSER: Address = address!("4200000000000000000000000000000000000016");
const L2_TO_L2_MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const NATIVE_ASSET_LIQUIDITY: Address = address!("4200000000000000000000000000000000000029");
const LIQUIDITY_CONTROLLER: Address = address!("420000000000000000000000000000000000002a");
const PROXY_ADMIN: Address = address!("4200000000000000000000000000000000000018");
const CLAIM_REGISTRY: Address = address!("420000000000000000000000000000000000002e");
const EVENT_REPLAYER: Address = address!("420000000000000000000000000000000000002f");
const L2_DEV_FEATURE_FLAGS: Address = address!("420000000000000000000000000000000000002d");

const IMPLEMENTATION_SLOT: B256 =
    b256!("360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");
const ADMIN_SLOT: B256 = b256!("b53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103");
const CUSTOM_GAS_TOKEN_SLOT: B256 =
    b256!("4ad9936a67aeb1898ef7b848aecdf71a1f8999fbf63ff2f5b5691cb14bedfe4d");
const DEV_FEATURE_BITMAP_SLOT: B256 =
    b256!("c8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc");
const OPTIMISM_PORTAL_INTEROP_FLAG: B256 =
    b256!("0000000000000000000000000000000000000000000000000000000000000001");
/// `L1Block.isFeatureEnabled[bytes32("INTEROP")]`: the mapping lives at storage slot 9 in both
/// `L1Block` and `L1BlockCGT`, so the slot is `keccak256(bytes32("INTEROP") ‖ uint256(9))`.
const L1_BLOCK_INTEROP_FEATURE_SLOT: B256 =
    b256!("3ebfd37456942048b852c384870d4ad41f6bbbdac131fb70bae79436d8f87a60");
const TRUE_WORD: B256 = b256!("0000000000000000000000000000000000000000000000000000000000000001");

/// keccak256 of the stock `L2ToL2CrossDomainMessenger` implementation the projection replaces.
/// Pins the contract release the projection was built against; must equal
/// `StockL2ToL2CrossDomainMessengerCodeHash` in the Go implementation.
pub(crate) const STOCK_L2_TO_L2_MESSENGER_CODE_HASH: B256 =
    b256!("6c9a755164bb4bf014b4b99358425e4480f3b022b7586b939a86f135c019acce");

const L1_BLOCK_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/L1Block.hex");
const L2_TO_L1_MESSAGE_PASSER_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/L2ToL1MessagePasser.hex");
const L2_TO_L2_MESSENGER_CODE: &str = include_str!(
    "../../../../../op-private-interop/genesis/bytecode/L2ToL2CrossDomainMessengerReplay.hex"
);
const CLAIM_REGISTRY_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/ClaimRegistry.hex");
const EVENT_REPLAYER_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/EventReplayer.hex");
const POLICY_MESSENGER_CODE: &str = include_str!(
    "../../../../../op-private-interop/genesis/bytecode/L2ToL2CrossDomainMessenger.hex"
);

/// A private-chain genesis cannot be projected safely.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GenesisProjectionError {
    /// Interop is not active at genesis: an activation block on the projection would run the
    /// stock network-upgrade bundle and replace the replay messenger.
    InteropInactive,
    /// The `L2ToL2CrossDomainMessenger` implementation is not the pinned stock release: another
    /// contract release, or a genesis that has already been projected.
    MessengerNotStock,
    /// The projection predeploys are already active: this is a public projection, not a source.
    AlreadyProjected,
}

impl fmt::Display for GenesisProjectionError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InteropInactive => {
                f.write_str("genesis is not a private chain: interop is not active at genesis")
            }
            Self::MessengerNotStock => f.write_str(
                "genesis is not a private chain: L2ToL2CrossDomainMessenger implementation is not the stock release",
            ),
            Self::AlreadyProjected => f.write_str(
                "genesis is already a public projection: projection predeploys are active",
            ),
        }
    }
}

#[cfg(feature = "std")]
impl std::error::Error for GenesisProjectionError {}

/// Constructs the public-projection genesis from a private-chain genesis.
///
/// This is a pure function. The source is cloned, the embedded bytecode is fixed protocol data,
/// and the transformation performs no I/O or environment-dependent work.
///
/// The source already carries the stock interop feature set. The projection keeps it and changes
/// exactly what makes the private chain private or custom-gas-token: the CGT implementations of
/// `L1Block` and `L2ToL1MessagePasser` become the ETH ones and the CGT marker is cleared;
/// `LiquidityController` and `NativeAssetLiquidity` are deactivated; the stock messenger becomes
/// the replay messenger; `ClaimRegistry` and `EventReplayer` are installed; the gas limit is the
/// maximum and the base fee is zero.
pub fn project_genesis_from(
    private_chain_genesis: &Genesis,
) -> Result<Genesis, GenesisProjectionError> {
    validate_private_chain_genesis(private_chain_genesis)?;

    let mut projected = private_chain_genesis.clone();
    projected.gas_limit = i64::MAX as u64;
    projected.base_fee_per_gas = Some(0);

    // Only a custom-gas-token source has CGT machinery to replace. An ETH source already runs ETH
    // semantics, and touching it would swap its own implementations for this crate's embedded
    // copies.
    if is_custom_gas_token(private_chain_genesis) {
        delete_storage(&mut projected, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT);
        activate_proxy(&mut projected, L1_BLOCK, bytecode(L1_BLOCK_CODE));
        activate_proxy(
            &mut projected,
            L2_TO_L1_MESSAGE_PASSER,
            bytecode(L2_TO_L1_MESSAGE_PASSER_CODE),
        );
        deactivate_proxy(&mut projected, NATIVE_ASSET_LIQUIDITY);
        deactivate_proxy(&mut projected, LIQUIDITY_CONTROLLER);
    }

    activate_proxy(&mut projected, L2_TO_L2_MESSENGER, bytecode(L2_TO_L2_MESSENGER_CODE));
    activate_proxy(&mut projected, CLAIM_REGISTRY, bytecode(CLAIM_REGISTRY_CODE));
    activate_proxy(&mut projected, EVENT_REPLAYER, bytecode(EVENT_REPLAYER_CODE));

    Ok(projected)
}

/// Reports whether the source runs the custom-gas-token execution path.
fn is_custom_gas_token(genesis: &Genesis) -> bool {
    storage_at(genesis, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT) != B256::ZERO
}

fn validate_private_chain_genesis(genesis: &Genesis) -> Result<(), GenesisProjectionError> {
    if storage_at(genesis, L1_BLOCK, L1_BLOCK_INTEROP_FEATURE_SLOT) != TRUE_WORD {
        return Err(GenesisProjectionError::InteropInactive);
    }
    let bitmap = storage_at(genesis, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT);
    if !contains_flag(bitmap, OPTIMISM_PORTAL_INTEROP_FLAG) {
        return Err(GenesisProjectionError::InteropInactive);
    }
    let messenger_hash = implementation_code_hash(genesis, L2_TO_L2_MESSENGER);
    if messenger_hash != STOCK_L2_TO_L2_MESSENGER_CODE_HASH &&
        messenger_hash != keccak256(bytecode(POLICY_MESSENGER_CODE))
    {
        return Err(GenesisProjectionError::MessengerNotStock);
    }
    for proxy in [CLAIM_REGISTRY, EVENT_REPLAYER] {
        if storage_at(genesis, proxy, IMPLEMENTATION_SLOT) != B256::ZERO {
            return Err(GenesisProjectionError::AlreadyProjected);
        }
    }
    Ok(())
}

/// Follows a proxy's EIP-1967 implementation slot and hashes the code found there. An inactive
/// proxy hashes to the empty-code hash, which no release matches.
fn implementation_code_hash(genesis: &Genesis, proxy: Address) -> B256 {
    let implementation =
        Address::from_slice(&storage_at(genesis, proxy, IMPLEMENTATION_SLOT)[12..]);
    let code = genesis
        .alloc
        .get(&implementation)
        .and_then(|account| account.code.as_ref())
        .map(|code| code.as_ref())
        .unwrap_or(&[]);
    keccak256(code)
}

/// Points a proxy at its code-namespace implementation and installs code there. A proxy whose
/// implementation lived elsewhere is re-pointed; the old implementation account is left as dead
/// code, matching the Go implementation.
fn activate_proxy(genesis: &mut Genesis, proxy: Address, code: Bytes) {
    account_at(genesis, proxy)
        .storage
        .get_or_insert_with(BTreeMap::new)
        .insert(IMPLEMENTATION_SLOT, address_word(code_namespace(proxy)));
    set_implementation(genesis, proxy, code);
}

fn set_implementation(genesis: &mut Genesis, proxy: Address, code: Bytes) {
    genesis.alloc.insert(
        code_namespace(proxy),
        GenesisAccount {
            code: Some(code),
            storage: Some(BTreeMap::from([(ADMIN_SLOT, address_word(PROXY_ADMIN))])),
            ..Default::default()
        },
    );
}

fn deactivate_proxy(genesis: &mut Genesis, proxy: Address) {
    let account = account_at(genesis, proxy);
    account.balance = U256::ZERO;
    account.storage = Some(BTreeMap::from([(ADMIN_SLOT, address_word(PROXY_ADMIN))]));
    genesis.alloc.remove(&code_namespace(proxy));
}

fn delete_storage(genesis: &mut Genesis, address: Address, slot: B256) {
    if let Some(storage) = account_at(genesis, address).storage.as_mut() {
        storage.remove(&slot);
    }
}

fn contains_flag(value: B256, flag: B256) -> bool {
    value
        .as_slice()
        .iter()
        .zip(flag.as_slice())
        .all(|(byte, flag_byte)| byte & flag_byte == *flag_byte)
}

fn account_at(genesis: &mut Genesis, address: Address) -> &mut GenesisAccount {
    genesis.alloc.entry(address).or_default()
}

fn storage_at(genesis: &Genesis, address: Address, slot: B256) -> B256 {
    genesis
        .alloc
        .get(&address)
        .and_then(|account| account.storage.as_ref())
        .and_then(|storage| storage.get(&slot))
        .copied()
        .unwrap_or_default()
}

fn code_namespace(proxy: Address) -> Address {
    let mut out = address!("c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30000").into_array();
    out[18..].copy_from_slice(&proxy.as_slice()[18..]);
    Address::from(out)
}

fn address_word(address: Address) -> B256 {
    let mut out = [0_u8; 32];
    out[12..].copy_from_slice(address.as_slice());
    B256::from(out)
}

fn bytecode(source: &str) -> Bytes {
    Bytes::from_str(source.trim())
        .unwrap_or_else(|err| panic!("invalid embedded private-interop projection bytecode: {err}"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpChainSpec;
    use reth_chainspec::EthChainSpec;

    /// The cross-language golden vector: a stock op-deployer genesis of a custom-gas-token chain
    /// with interop at genesis, reduced to the predeploys and one funded account. The Go tests pin
    /// the same two hashes (`op-private-interop/genesis/projection_test.go`).
    const PRIVATE_CHAIN_GENESIS_FIXTURE: &str = include_str!(
        "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
    );
    const PUBLIC_PROJECTION_STATE_ROOT: B256 =
        b256!("0bcc2c671be43df86d2bdfc0e2ef4a8402924e2bdfaf5e8e432eb5841282c81f");
    const PUBLIC_PROJECTION_BLOCK_HASH: B256 =
        b256!("b16205791987c1d52be25be000d27661f42aeccf033183f3a1ea396ce23539be");

    const SUPERCHAIN_ETH_BRIDGE: Address = address!("4200000000000000000000000000000000000024");
    const ETH_LIQUIDITY: Address = address!("4200000000000000000000000000000000000025");
    const CROSS_L2_INBOX: Address = address!("4200000000000000000000000000000000000022");

    fn private_chain_genesis() -> Genesis {
        serde_json::from_str(PRIVATE_CHAIN_GENESIS_FIXTURE).unwrap()
    }

    #[test]
    fn projection_is_pure_and_deterministic() {
        let private = private_chain_genesis();
        let before = private.clone();
        let first = project_genesis_from(&private).unwrap();
        let second = project_genesis_from(&private).unwrap();

        assert_eq!(private, before);
        assert_eq!(first, second);
        assert_eq!(first.gas_limit, i64::MAX as u64);
        assert_eq!(first.base_fee_per_gas, Some(0));
    }

    #[test]
    fn projection_matches_the_cross_language_golden_vector() {
        let projected = project_genesis_from(&private_chain_genesis()).unwrap();
        let spec = OpChainSpec::from_genesis(projected);

        assert_eq!(spec.genesis_header().state_root, PUBLIC_PROJECTION_STATE_ROOT);
        assert_eq!(spec.genesis_hash(), PUBLIC_PROJECTION_BLOCK_HASH);
    }

    #[test]
    fn policy_profile_matches_the_cross_language_golden_vector() {
        let mut private = private_chain_genesis();
        delete_storage(&mut private, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT);
        activate_proxy(&mut private, L1_BLOCK, bytecode(L1_BLOCK_CODE));
        activate_proxy(
            &mut private,
            L2_TO_L1_MESSAGE_PASSER,
            bytecode(L2_TO_L1_MESSAGE_PASSER_CODE),
        );
        deactivate_proxy(&mut private, NATIVE_ASSET_LIQUIDITY);
        deactivate_proxy(&mut private, LIQUIDITY_CONTROLLER);
        activate_proxy(&mut private, L2_TO_L2_MESSENGER, bytecode(POLICY_MESSENGER_CODE));
        account_at(&mut private, L2_TO_L2_MESSENGER)
            .storage
            .as_mut()
            .unwrap()
            .insert(keccak256("privateinterop.requirePaidMessages"), TRUE_WORD);
        activate_proxy(
            &mut private,
            SUPERCHAIN_ETH_BRIDGE,
            bytecode(include_str!(
                "../../../../../op-private-interop/genesis/bytecode/SuperchainETHBridge.hex"
            )),
        );
        assert_eq!(
            OpChainSpec::from_genesis(private.clone()).genesis_hash(),
            b256!("bdd4b5a0b1d41a1467f4cede7fa52f4d0f56e59cc9556f95cd75b818fb73a374")
        );
        let spec = OpChainSpec::from_genesis(project_genesis_from(&private).unwrap());
        assert_eq!(
            spec.genesis_hash(),
            b256!("a7f8b6152f13136eaac74fada0f2d43cfc84d62844bdf000d88ea36be3a53008")
        );
        assert_eq!(
            spec.genesis_header().state_root,
            b256!("abb2fb272931bef047ae2ff61312e2ad82e369573552e351e2e3e68bae5372f6")
        );
    }

    #[test]
    fn fixture_is_the_stock_shape() {
        let private = private_chain_genesis();
        assert_eq!(
            implementation_code_hash(&private, L2_TO_L2_MESSENGER),
            STOCK_L2_TO_L2_MESSENGER_CODE_HASH
        );
        assert_eq!(storage_at(&private, L1_BLOCK, L1_BLOCK_INTEROP_FEATURE_SLOT), TRUE_WORD);
        assert_ne!(storage_at(&private, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT), B256::ZERO);
        assert_eq!(private.alloc[&ETH_LIQUIDITY].balance, U256::from(u128::MAX));
    }

    #[test]
    fn projection_rewrites_only_the_public_projection_state() {
        let private = private_chain_genesis();
        let projected = project_genesis_from(&private).unwrap();

        for proxy in
            [L1_BLOCK, L2_TO_L1_MESSAGE_PASSER, L2_TO_L2_MESSENGER, CLAIM_REGISTRY, EVENT_REPLAYER]
        {
            assert_eq!(
                storage_at(&projected, proxy, IMPLEMENTATION_SLOT),
                address_word(code_namespace(proxy))
            );
            assert!(
                projected.alloc[&code_namespace(proxy)]
                    .code
                    .as_ref()
                    .is_some_and(|c| !c.is_empty())
            );
        }
        assert_ne!(
            private.alloc[&code_namespace(L1_BLOCK)].code,
            projected.alloc[&code_namespace(L1_BLOCK)].code,
            "L1BlockCGT replaced"
        );
        for proxy in [NATIVE_ASSET_LIQUIDITY, LIQUIDITY_CONTROLLER] {
            assert_eq!(storage_at(&projected, proxy, IMPLEMENTATION_SLOT), B256::ZERO);
            assert!(!projected.alloc.contains_key(&code_namespace(proxy)));
            assert_eq!(projected.alloc[&proxy].balance, U256::ZERO);
        }
        assert_eq!(storage_at(&projected, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT), B256::ZERO);
        assert_eq!(storage_at(&projected, L1_BLOCK, L1_BLOCK_INTEROP_FEATURE_SLOT), TRUE_WORD);
        assert!(contains_flag(
            storage_at(&projected, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT),
            OPTIMISM_PORTAL_INTEROP_FLAG
        ));
        for proxy in [CROSS_L2_INBOX, SUPERCHAIN_ETH_BRIDGE, ETH_LIQUIDITY] {
            assert_eq!(private.alloc[&proxy], projected.alloc[&proxy]);
            assert_eq!(
                private.alloc[&code_namespace(proxy)],
                projected.alloc[&code_namespace(proxy)]
            );
        }
        assert_eq!(projected.alloc[&ETH_LIQUIDITY].balance, U256::from(u128::MAX));
    }

    #[test]
    fn non_sources_are_rejected() {
        // An interop-active ETH source is supported: nothing to strip, implementations untouched.
        let mut eth_source = private_chain_genesis();
        delete_storage(&mut eth_source, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT);
        let projected_eth = project_genesis_from(&eth_source).unwrap();
        assert_eq!(
            eth_source.alloc[&code_namespace(L1_BLOCK)].code,
            projected_eth.alloc[&code_namespace(L1_BLOCK)].code
        );
        assert_eq!(
            storage_at(&projected_eth, L2_TO_L2_MESSENGER, IMPLEMENTATION_SLOT),
            address_word(code_namespace(L2_TO_L2_MESSENGER))
        );

        let mut inactive = private_chain_genesis();
        delete_storage(&mut inactive, L1_BLOCK, L1_BLOCK_INTEROP_FEATURE_SLOT);
        assert_eq!(project_genesis_from(&inactive), Err(GenesisProjectionError::InteropInactive));

        let mut no_flag = private_chain_genesis();
        delete_storage(&mut no_flag, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT);
        assert_eq!(project_genesis_from(&no_flag), Err(GenesisProjectionError::InteropInactive));

        let mut other_release = private_chain_genesis();
        let implementation =
            other_release.alloc.get_mut(&code_namespace(L2_TO_L2_MESSENGER)).unwrap();
        let mut code = alloc::vec![0x00_u8];
        code.extend_from_slice(implementation.code.as_ref().unwrap());
        implementation.code = Some(Bytes::from(code));
        assert_eq!(
            project_genesis_from(&other_release),
            Err(GenesisProjectionError::MessengerNotStock)
        );

        let projected = project_genesis_from(&private_chain_genesis()).unwrap();
        assert!(project_genesis_from(&projected).is_err());

        let mut half_projected = private_chain_genesis();
        activate_proxy(&mut half_projected, CLAIM_REGISTRY, Bytes::from_static(&[0xfe]));
        assert_eq!(
            project_genesis_from(&half_projected),
            Err(GenesisProjectionError::AlreadyProjected)
        );
    }
}
