//! Deterministic private-chain genesis to public-projection genesis transformation.

use alloc::collections::BTreeMap;
use alloy_genesis::{Genesis, GenesisAccount};
use alloy_primitives::{Address, B256, Bytes, U256, address, b256};
use core::{fmt, str::FromStr};

const L1_BLOCK: Address = address!("4200000000000000000000000000000000000015");
const L2_TO_L1_MESSAGE_PASSER: Address = address!("4200000000000000000000000000000000000016");
const L2_TO_L2_MESSENGER: Address = address!("4200000000000000000000000000000000000023");
const SUPERCHAIN_ETH_BRIDGE: Address = address!("4200000000000000000000000000000000000024");
const ETH_LIQUIDITY: Address = address!("4200000000000000000000000000000000000025");
const NATIVE_ASSET_LIQUIDITY: Address = address!("4200000000000000000000000000000000000029");
const LIQUIDITY_CONTROLLER: Address = address!("420000000000000000000000000000000000002a");
const PROXY_ADMIN: Address = address!("4200000000000000000000000000000000000018");
const CLAIM_REGISTRY: Address = address!("420000000000000000000000000000000000002e");
const EVENT_REPLAYER: Address = address!("420000000000000000000000000000000000002f");
const NATIVE_MINT_BRIDGE: Address = address!("4200000000000000000000000000000000000030");
const L2_DEV_FEATURE_FLAGS: Address = address!("420000000000000000000000000000000000002d");

const IMPLEMENTATION_SLOT: B256 =
    b256!("360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");
const ADMIN_SLOT: B256 = b256!("b53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103");
const CUSTOM_GAS_TOKEN_SLOT: B256 =
    b256!("4ad9936a67aeb1898ef7b848aecdf71a1f8999fbf63ff2f5b5691cb14bedfe4d");
const DEV_FEATURE_BITMAP_SLOT: B256 =
    b256!("c8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc");
const PRIVATE_INTEROP_FLAG: B256 =
    b256!("0000000000000000000000000000000000000000000000000000001000000000");
#[cfg(test)]
const OPTIMISM_PORTAL_INTEROP_FLAG: B256 =
    b256!("0000000000000000000000000000000000000000000000000000000000000001");

const L1_BLOCK_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/L1Block.hex");
const L2_TO_L1_MESSAGE_PASSER_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/L2ToL1MessagePasser.hex");
const L2_TO_L2_MESSENGER_CODE: &str = include_str!(
    "../../../../../op-private-interop/genesis/bytecode/L2ToL2CrossDomainMessengerReplay.hex"
);
const SUPERCHAIN_ETH_BRIDGE_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/SuperchainETHBridge.hex");
const ETH_LIQUIDITY_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/ETHLiquidity.hex");
const CLAIM_REGISTRY_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/ClaimRegistry.hex");
const EVENT_REPLAYER_CODE: &str =
    include_str!("../../../../../op-private-interop/genesis/bytecode/EventReplayer.hex");

/// A private-chain genesis cannot be projected safely.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GenesisProjectionError {
    /// The native mint bridge does not identify an active private-chain genesis.
    NativeMintBridgeInactive,
    /// The custom-gas-token marker does not identify a private-chain genesis.
    CustomGasTokenDisabled,
    /// The development feature bitmap does not identify a private-chain genesis.
    PrivateInteropDisabled,
}

impl fmt::Display for GenesisProjectionError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NativeMintBridgeInactive => {
                f.write_str("genesis is not a private chain: NativeMintBridge is inactive")
            }
            Self::CustomGasTokenDisabled => {
                f.write_str("genesis is not a private chain: custom gas token is disabled")
            }
            Self::PrivateInteropDisabled => {
                f.write_str("genesis is not a private chain: private interop feature is disabled")
            }
        }
    }
}

#[cfg(feature = "std")]
impl std::error::Error for GenesisProjectionError {}

/// Constructs the public-projection genesis from a private-chain genesis.
///
/// This is a pure function. The source is cloned, the embedded bytecode is fixed protocol data,
/// and the transformation performs no I/O or environment-dependent work.
pub fn project_genesis_from(
    private_chain_genesis: &Genesis,
) -> Result<Genesis, GenesisProjectionError> {
    validate_private_chain_genesis(private_chain_genesis)?;

    let mut projected = private_chain_genesis.clone();
    projected.gas_limit = i64::MAX as u64;
    projected.base_fee_per_gas = Some(0);

    delete_storage(&mut projected, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT);
    clear_storage_flag(
        &mut projected,
        L2_DEV_FEATURE_FLAGS,
        DEV_FEATURE_BITMAP_SLOT,
        PRIVATE_INTEROP_FLAG,
    );
    set_implementation(&mut projected, L1_BLOCK, bytecode(L1_BLOCK_CODE));
    set_implementation(
        &mut projected,
        L2_TO_L1_MESSAGE_PASSER,
        bytecode(L2_TO_L1_MESSAGE_PASSER_CODE),
    );

    activate_proxy(&mut projected, SUPERCHAIN_ETH_BRIDGE, bytecode(SUPERCHAIN_ETH_BRIDGE_CODE));
    activate_proxy(&mut projected, ETH_LIQUIDITY, bytecode(ETH_LIQUIDITY_CODE));
    account_at(&mut projected, ETH_LIQUIDITY).balance = U256::from(u128::MAX);

    deactivate_proxy(&mut projected, NATIVE_ASSET_LIQUIDITY);
    deactivate_proxy(&mut projected, LIQUIDITY_CONTROLLER);
    deactivate_proxy(&mut projected, NATIVE_MINT_BRIDGE);

    activate_proxy(&mut projected, L2_TO_L2_MESSENGER, bytecode(L2_TO_L2_MESSENGER_CODE));
    activate_proxy(&mut projected, CLAIM_REGISTRY, bytecode(CLAIM_REGISTRY_CODE));
    activate_proxy(&mut projected, EVENT_REPLAYER, bytecode(EVENT_REPLAYER_CODE));

    Ok(projected)
}

fn validate_private_chain_genesis(genesis: &Genesis) -> Result<(), GenesisProjectionError> {
    if storage_at(genesis, NATIVE_MINT_BRIDGE, IMPLEMENTATION_SLOT) == B256::ZERO {
        return Err(GenesisProjectionError::NativeMintBridgeInactive);
    }
    if storage_at(genesis, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT) == B256::ZERO {
        return Err(GenesisProjectionError::CustomGasTokenDisabled);
    }
    let bitmap = storage_at(genesis, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT);
    if !contains_flag(bitmap, PRIVATE_INTEROP_FLAG) {
        return Err(GenesisProjectionError::PrivateInteropDisabled);
    }
    Ok(())
}

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

fn clear_storage_flag(genesis: &mut Genesis, address: Address, slot: B256, flag: B256) {
    let mut value = [0_u8; 32];
    value.copy_from_slice(storage_at(genesis, address, slot).as_slice());
    for (byte, flag_byte) in value.iter_mut().zip(flag.as_slice()) {
        *byte &= !flag_byte;
    }
    account_at(genesis, address)
        .storage
        .get_or_insert_with(BTreeMap::new)
        .insert(slot, B256::from(value));
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

    const PRIVATE_CHAIN_GENESIS_FIXTURE: &str = include_str!(
        "../../../../../op-private-interop/genesis/testdata/private-chain-genesis.json"
    );
    const PUBLIC_PROJECTION_STATE_ROOT: B256 =
        b256!("b4a3530d2e27fd008c94ba70c47760c3109a032d3c646872591ca622b4a59cae");
    const PUBLIC_PROJECTION_BLOCK_HASH: B256 =
        b256!("aaca1c33c16e560b035482ad57fd6f38436d9c5cf45faf57e3f3cb3512335347");

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
        let private: Genesis = serde_json::from_str(PRIVATE_CHAIN_GENESIS_FIXTURE).unwrap();
        let projected = project_genesis_from(&private).unwrap();
        let spec = OpChainSpec::from_genesis(projected);

        assert_eq!(spec.genesis_header().state_root, PUBLIC_PROJECTION_STATE_ROOT);
        assert_eq!(spec.genesis_hash(), PUBLIC_PROJECTION_BLOCK_HASH);
    }

    #[test]
    fn projection_activates_public_and_removes_private_predeploys() {
        let projected = project_genesis_from(&private_chain_genesis()).unwrap();
        for proxy in [
            SUPERCHAIN_ETH_BRIDGE,
            ETH_LIQUIDITY,
            L2_TO_L2_MESSENGER,
            CLAIM_REGISTRY,
            EVENT_REPLAYER,
        ] {
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
        for proxy in [NATIVE_ASSET_LIQUIDITY, LIQUIDITY_CONTROLLER, NATIVE_MINT_BRIDGE] {
            assert_eq!(storage_at(&projected, proxy, IMPLEMENTATION_SLOT), B256::ZERO);
            assert!(!projected.alloc.contains_key(&code_namespace(proxy)));
        }
        assert_eq!(storage_at(&projected, L1_BLOCK, CUSTOM_GAS_TOKEN_SLOT), B256::ZERO);
        assert!(!contains_flag(
            storage_at(&projected, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT),
            PRIVATE_INTEROP_FLAG
        ));
        assert!(contains_flag(
            storage_at(&projected, L2_DEV_FEATURE_FLAGS, DEV_FEATURE_BITMAP_SLOT),
            OPTIMISM_PORTAL_INTEROP_FLAG
        ));
        assert_eq!(projected.alloc[&ETH_LIQUIDITY].balance, U256::from(u128::MAX));
    }

    #[test]
    fn ordinary_genesis_is_rejected() {
        let mut private = private_chain_genesis();
        private
            .alloc
            .get_mut(&NATIVE_MINT_BRIDGE)
            .unwrap()
            .storage
            .as_mut()
            .unwrap()
            .remove(&IMPLEMENTATION_SLOT);
        assert_eq!(
            project_genesis_from(&private),
            Err(GenesisProjectionError::NativeMintBridgeInactive)
        );
    }

    fn private_chain_genesis() -> Genesis {
        let mut genesis = Genesis {
            gas_limit: 30_000_000,
            base_fee_per_gas: Some(1_000_000_000),
            ..Default::default()
        };
        for proxy in [
            L1_BLOCK,
            L2_TO_L1_MESSAGE_PASSER,
            L2_TO_L2_MESSENGER,
            SUPERCHAIN_ETH_BRIDGE,
            ETH_LIQUIDITY,
            NATIVE_ASSET_LIQUIDITY,
            LIQUIDITY_CONTROLLER,
            CLAIM_REGISTRY,
            EVENT_REPLAYER,
            NATIVE_MINT_BRIDGE,
        ] {
            genesis.alloc.insert(
                proxy,
                GenesisAccount {
                    code: Some(Bytes::from_static(&[0x60, 0x00])),
                    storage: Some(BTreeMap::from([(ADMIN_SLOT, address_word(PROXY_ADMIN))])),
                    ..Default::default()
                },
            );
        }
        for proxy in [NATIVE_ASSET_LIQUIDITY, LIQUIDITY_CONTROLLER, NATIVE_MINT_BRIDGE] {
            genesis
                .alloc
                .get_mut(&proxy)
                .unwrap()
                .storage
                .as_mut()
                .unwrap()
                .insert(IMPLEMENTATION_SLOT, address_word(code_namespace(proxy)));
            genesis.alloc.insert(
                code_namespace(proxy),
                GenesisAccount { code: Some(Bytes::from_static(&[0xfe])), ..Default::default() },
            );
        }
        genesis
            .alloc
            .get_mut(&L1_BLOCK)
            .unwrap()
            .storage
            .as_mut()
            .unwrap()
            .insert(CUSTOM_GAS_TOKEN_SLOT, B256::with_last_byte(1));
        genesis.alloc.insert(
            L2_DEV_FEATURE_FLAGS,
            GenesisAccount {
                storage: Some(BTreeMap::from([(
                    DEV_FEATURE_BITMAP_SLOT,
                    B256::from(U256::from(0x10_0000_0001_u64)),
                )])),
                ..Default::default()
            },
        );
        genesis
    }
}
