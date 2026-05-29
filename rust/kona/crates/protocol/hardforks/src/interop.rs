//! Module containing the Interop network upgrade activation.
//!
//! Interop's activation block always executes the Interop NUT bundle. Chains in a
//! multi-chain dependency set additionally wrap the bundle with:
//!  1. A pre-bundle `L1Block.setFeature(INTEROP)` call.
//!  2. A post-bundle `ETHLiquidity` funding deposit with mint and value set to `u128::MAX`.

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
/// Bootstrap mint and value for the post-bundle `ETHLiquidity` funding deposit.
const ETH_LIQUIDITY_FUND_AMOUNT: u128 = u128::MAX;

/// `bytes32` representation of the INTEROP feature constant (right-padded ASCII).
const INTEROP_FEATURE: [u8; 32] = *b"INTEROP\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0\0";

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

        let source =
            UpgradeDepositSource { intent: String::from("Interop post: ETHLiquidity Funding") };
        TxDeposit {
            source_hash: source.source_hash(),
            from: DEPOSITOR_ACCOUNT,
            to: TxKind::Call(Predeploys::ETH_LIQUIDITY),
            mint: ETH_LIQUIDITY_FUND_AMOUNT,
            value: U256::from(ETH_LIQUIDITY_FUND_AMOUNT),
            gas_limit: ETH_LIQUIDITY_FUND_GAS,
            is_system_transaction: false,
            input: selector,
        }
    }

    /// Returns the JSON bundle deposit transactions that execute on every
    /// Interop activation block, even for single-chain dependency sets.
    fn bundle_deposits() -> Vec<TxDeposit> {
        let bundle = interop_nut_bundle();
        bundle.to_deposit_transactions().expect("Interop NUT bundle is invalid")
    }

    /// Returns all deposit transactions for the Interop activation block.
    pub fn deposits(activate_interop_contracts: bool) -> Vec<TxDeposit> {
        let bundle_deposits = Self::bundle_deposits();
        let wrapper_count = if activate_interop_contracts { 2 } else { 0 };
        let mut deposits = Vec::with_capacity(wrapper_count + bundle_deposits.len());
        if activate_interop_contracts {
            deposits.push(Self::set_feature_tx());
        }
        deposits.extend(bundle_deposits);
        if activate_interop_contracts {
            deposits.push(Self::eth_liquidity_funding_tx());
        }
        deposits
    }

    /// Returns the encoded Interop activation transactions for a chain.
    pub fn txs_for_activation(
        &self,
        activate_interop_contracts: bool,
    ) -> impl Iterator<Item = Bytes> + '_ {
        Self::deposits(activate_interop_contracts).into_iter().map(|tx| {
            let mut encoded = Vec::new();
            tx.encode_2718(&mut encoded);
            Bytes::from(encoded)
        })
    }

    /// Returns the additional gas required by the Interop activation transactions.
    pub fn upgrade_gas_for_activation(&self, activate_interop_contracts: bool) -> u64 {
        let mut gas = interop_nut_bundle().total_gas();
        if activate_interop_contracts {
            gas += SET_FEATURE_GAS + ETH_LIQUIDITY_FUND_GAS;
        }
        gas
    }
}

impl Hardfork for Interop {
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_ {
        self.txs_for_activation(true)
    }

    fn upgrade_gas(&self) -> u64 {
        self.upgrade_gas_for_activation(true)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn deposits_have_correct_count() {
        // 1 setFeature + 28 bundle txs + 1 ETHLiquidity funding = 30
        assert_eq!(Interop::deposits(true).len(), 30);
        assert_eq!(Interop::deposits(false).len(), 28);
    }

    #[test]
    fn first_tx_is_set_feature() {
        let deps = Interop::deposits(true);
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
        let deps = Interop::deposits(true);
        let last = deps.last().unwrap();
        assert_eq!(last.to, TxKind::Call(Predeploys::ETH_LIQUIDITY));
        assert_eq!(last.mint, u128::MAX);
        assert_eq!(last.value, U256::from(u128::MAX));
    }

    #[test]
    fn upgrade_gas_sums_all_three_pieces() {
        let bundle_gas = interop_nut_bundle().total_gas();
        let interop = Interop {};
        let total = interop.upgrade_gas_for_activation(true);
        assert_eq!(total, SET_FEATURE_GAS + bundle_gas + ETH_LIQUIDITY_FUND_GAS);
        assert_eq!(interop.upgrade_gas_for_activation(false), bundle_gas);
    }

    #[test]
    fn first_bundle_tx_uses_qualified_intent() {
        let deps = Interop::deposits(true);
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
