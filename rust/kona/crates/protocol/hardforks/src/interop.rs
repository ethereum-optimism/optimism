//! Module containing the Interop network upgrade activation.
//!
//! Interop's activation block consists of:
//!  1. A pre-bundle `L1Block.setFeature(INTEROP)` call.
//!  2. The interop NUT bundle to upgrade the contracts and deploy interop specific contracts.
//!  3. A post-bundle `ETHLiquidity` funding deposit with mint and value set to `u128::MAX`.

use alloc::{string::String, vec::Vec};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, Bytes, TxKind, U256, address, keccak256};
use kona_protocol::Predeploys;
use op_alloy_consensus::{TxDeposit, UpgradeDepositSource};

use crate::Hardfork;

include!(concat!(env!("OUT_DIR"), "/interop_nut_bundle.rs"));

/// The depositor account that may invoke `L1Block.setFeature`.
/// Matches the per-fork constant used by ecotone/isthmus/jovian.
const DEPOSITOR_ACCOUNT: Address = address!("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

/// Gas limit for the pre-bundle `setFeature(INTEROP)` wrapper tx.
const SET_FEATURE_GAS: u64 = 100_000;
/// Gas limit for the post-bundle `ETHLiquidity` funding wrapper tx.
const ETH_LIQUIDITY_FUND_GAS: u64 = 50_000;

/// `bytes32` representation of the INTEROP feature constant (right-padded ASCII).
const INTEROP_FEATURE: [u8; 32] = {
    let mut buf = [0u8; 32];
    let bytes = b"INTEROP";
    let mut i = 0;
    while i < bytes.len() {
        buf[i] = bytes[i];
        i += 1;
    }
    buf
};

/// The Interop hardfork.
#[derive(Debug, Default, Clone, Copy)]
pub struct Interop;

impl Interop {
    /// Returns the pre-bundle `L1Block.setFeature(INTEROP)` deposit.
    fn set_feature_tx() -> TxDeposit {
        let selector = &keccak256(b"setFeature(bytes32)")[..4];
        let mut data = Vec::with_capacity(4 + 32);
        data.extend_from_slice(selector);
        data.extend_from_slice(&INTEROP_FEATURE);

        let source =
            UpgradeDepositSource { intent: String::from("Interop pre: setFeature(INTEROP)") };
        TxDeposit {
            source_hash: source.source_hash(),
            from: DEPOSITOR_ACCOUNT,
            to: TxKind::Call(Predeploys::L1_BLOCK_INFO),
            mint: 0,
            value: U256::ZERO,
            gas_limit: SET_FEATURE_GAS,
            is_system_transaction: false,
            input: Bytes::from(data),
        }
    }

    /// Returns the post-bundle `ETHLiquidity` funding deposit.
    fn eth_liquidity_funding_tx() -> TxDeposit {
        let selector = Bytes::copy_from_slice(&keccak256(b"fund()")[..4]);
        let amount: u128 = u128::MAX;

        let source =
            UpgradeDepositSource { intent: String::from("Interop post: ETHLiquidity Funding") };
        TxDeposit {
            source_hash: source.source_hash(),
            from: DEPOSITOR_ACCOUNT,
            to: TxKind::Call(Predeploys::ETH_LIQUIDITY),
            mint: amount,
            value: U256::from(amount),
            gas_limit: ETH_LIQUIDITY_FUND_GAS,
            is_system_transaction: false,
            input: selector,
        }
    }

    /// Returns all deposit transactions for the Interop activation block.
    fn deposits() -> Vec<TxDeposit> {
        let bundle = interop_nut_bundle();
        let bundle_deposits =
            bundle.to_deposit_transactions().expect("Interop NUT bundle is invalid");
        let mut deposits = Vec::with_capacity(2 + bundle_deposits.len());
        deposits.push(Self::set_feature_tx());
        deposits.extend(bundle_deposits);
        deposits.push(Self::eth_liquidity_funding_tx());
        deposits
    }
}

impl Hardfork for Interop {
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_ {
        Self::deposits().into_iter().map(|tx| {
            let mut encoded = Vec::new();
            tx.encode_2718(&mut encoded);
            Bytes::from(encoded)
        })
    }

    fn upgrade_gas(&self) -> u64 {
        let bundle_gas = interop_nut_bundle().total_gas();
        SET_FEATURE_GAS + bundle_gas + ETH_LIQUIDITY_FUND_GAS
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn deposits_have_correct_count() {
        // 1 setFeature + 28 bundle txs + 1 ETHLiquidity funding = 30
        assert_eq!(Interop::deposits().len(), 30);
    }

    #[test]
    fn first_tx_is_set_feature() {
        let deps = Interop::deposits();
        assert_eq!(deps[0].to, TxKind::Call(Predeploys::L1_BLOCK_INFO));
        assert_eq!(deps[0].mint, 0);
        assert_eq!(deps[0].value, U256::ZERO);
        let expected =
            UpgradeDepositSource { intent: String::from("Interop pre: setFeature(INTEROP)") }
                .source_hash();
        assert_eq!(deps[0].source_hash, expected);
    }

    #[test]
    fn last_tx_is_eth_liquidity_funding_with_max_mint_and_value() {
        let deps = Interop::deposits();
        let last = deps.last().unwrap();
        assert_eq!(last.to, TxKind::Call(Predeploys::ETH_LIQUIDITY));
        assert_eq!(last.mint, u128::MAX);
        assert_eq!(last.value, U256::from(u128::MAX));
    }

    #[test]
    fn upgrade_gas_sums_all_three_pieces() {
        let bundle_gas = interop_nut_bundle().total_gas();
        let total = (Interop {}).upgrade_gas();
        assert_eq!(total, SET_FEATURE_GAS + bundle_gas + ETH_LIQUIDITY_FUND_GAS);
    }

    #[test]
    fn first_bundle_tx_uses_qualified_intent() {
        let deps = Interop::deposits();
        // deps[0] = wrapper, deps[1] = first bundle tx
        // The build script generates the bundle with a capitalized fork name
        // ("interop" → "Interop"), so the qualified intent on the kona side is
        // "Interop 0: ...". (Note: the op-node Go side uses lowercase
        // forks.Interop. This pre-existing capitalization difference also
        // applies to Karst.)
        let expected_intent = "Interop 0: Deploy StorageSetter Implementation";
        let expected = UpgradeDepositSource { intent: String::from(expected_intent) }.source_hash();
        assert_eq!(deps[1].source_hash, expected);
    }
}
