//! Module containing a [`TxDeposit`] builder for the Interop network upgrade transactions.
//!
//! Interop network upgrade transactions are defined in the [OP Stack Specs][specs].
//!
//! [specs]: https://specs.optimism.io/interop/derivation.html#network-upgrade-transactions

use alloc::string::String;
use alloy_eips::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, address, b256, hex};
use kona_protocol::Predeploys;
use op_alloy_consensus::{TxDeposit, UpgradeDepositSource};

use crate::Hardfork;

/// The Interop network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Interop;

impl Interop {
    /// The depositor account address.
    pub const DEPOSITOR_ACCOUNT: Address = address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

    /// The deployer of the `CrossL2Inbox` contract.
    pub const CROSS_L2_INBOX_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000000");

    /// The deployer of the `L2ToL2CrossDomainMessenger` contract.
    pub const L2_TO_L2_XDM_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000001");

    /// The deployer of the `SuperchainETHBridge` contract.
    pub const SUPERCHAIN_ETH_BRIDGE_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000002");

    /// The deployer of the `ETHLiquidity` contract.
    pub const ETH_LIQUIDITY_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000003");

    /// The deployed address of the `CrossL2Inbox` implementation contract.
    pub const NEW_CROSS_L2_INBOX_IMPL: Address =
        address!("0x691300f512e48B463C2617b34Eef1A9f82EE7dBf");

    /// The code hash of the deployed `CrossL2Inbox` implementation contract.
    pub const CROSS_L2_INBOX_IMPL_CODE_HASH: B256 =
        b256!("0x0e7d028dd71bac22d1fb28966043c8d35c3232c78b7fb99fd1db112b5b60d9dd");

    /// The deployment address of the `L2ToL2CrossDomainMessenger` implementation contract.
    pub const NEW_L2_TO_L2_XDM_IMPL: Address =
        address!("0x0D0eDd0ebd0e94d218670a8De867Eb5C4d37cadD");

    /// The code hash of the deployed `L2ToL2CrossDomainMessenger` implementation contract.
    pub const L2_TO_L2_XDM_IMPL_CODE_HASH: B256 =
        b256!("0x458925c90ec70736600bef3d6529643a0e7a0a848e62626d61314c057b4a71a9");

    /// The deployed address of the `SuperchainETHBridge` implementation contract.
    pub const NEW_SUPERCHAIN_ETH_BRIDGE_IMPL: Address =
        address!("0x1913211d1257Da0C92149A0b0F07086C5E4Fc9E0");

    /// The deployed address of the `ETHLiquidity` implementation contract.
    pub const NEW_ETH_LIQUIDITY_IMPL: Address =
        address!("0xEFDdA7c9172e7E4B634d7bF97D315dDe24327fe0");

    /// The `SuperchainETHBridge` proxy predeploy address.
    pub const SUPERCHAIN_ETH_BRIDGE: Address =
        address!("0x4200000000000000000000000000000000000024");

    /// The `ETHLiquidity` proxy predeploy address.
    pub const ETH_LIQUIDITY: Address = address!("0x4200000000000000000000000000000000000025");

    /// The `fund()` selector for `ETHLiquidity`.
    pub const ETH_LIQUIDITY_FUND_SELECTOR: [u8; 4] = hex!("b60d4288");

    /// Returns the source hash for the `CrossL2Inbox` contract deployment transaction.
    pub fn deploy_cross_l2_inbox_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: CrossL2Inbox Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the `CrossL2Inbox` proxy upgrade transaction.
    pub fn upgrade_cross_l2_inbox_proxy_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: CrossL2Inbox Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the `L2ToL2CrossDomainMessenger` deployment transaction.
    pub fn deploy_l2_to_l2_xdm_source() -> B256 {
        UpgradeDepositSource {
            intent: String::from("Interop: L2ToL2CrossDomainMessenger Deployment"),
        }
        .source_hash()
    }

    /// Returns the source hash for the `L2ToL2CrossDomainMessenger` proxy upgrade transaction.
    pub fn upgrade_l2_to_l2_xdm_proxy_source() -> B256 {
        UpgradeDepositSource {
            intent: String::from("Interop: L2ToL2CrossDomainMessenger Proxy Update"),
        }
        .source_hash()
    }

    /// Returns the source hash for the `SuperchainETHBridge` deployment transaction.
    pub fn deploy_superchain_eth_bridge_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: SuperchainETHBridge Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the `SuperchainETHBridge` proxy upgrade transaction.
    pub fn upgrade_superchain_eth_bridge_proxy_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: SuperchainETHBridge Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the `ETHLiquidity` deployment transaction.
    pub fn deploy_eth_liquidity_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: ETHLiquidity Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the `ETHLiquidity` proxy upgrade transaction.
    pub fn upgrade_eth_liquidity_proxy_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: ETHLiquidity Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the `ETHLiquidity` funding transaction.
    pub fn fund_eth_liquidity_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Interop: ETHLiquidity Funding") }.source_hash()
    }

    /// Returns the `CrossL2Inbox` deployment bytecode.
    pub fn cross_l2_inbox_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/crossl2inbox_interop.hex").replace('\n', ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the `L2ToL2CrossDomainMessenger` proxy upgrade bytecode.
    pub fn l2_to_l2_xdm_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/l2tol2_xdm_interop.hex").replace('\n', ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the `SuperchainETHBridge` deployment bytecode.
    pub fn superchain_eth_bridge_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/superchain_eth_bridge_interop.hex").replace('\n', ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the `ETHLiquidity` deployment bytecode.
    pub fn eth_liquidity_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/eth_liquidity_interop.hex").replace('\n', ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the calldata for the `ETHLiquidity.fund()` call.
    pub fn eth_liquidity_fund_calldata() -> Bytes {
        Bytes::copy_from_slice(&Self::ETH_LIQUIDITY_FUND_SELECTOR)
    }

    /// Returns the list of [`TxDeposit`]s for the network upgrade.
    pub fn deposits() -> impl Iterator<Item = TxDeposit> {
        ([
            TxDeposit {
                source_hash: Self::deploy_cross_l2_inbox_source(),
                from: Self::CROSS_L2_INBOX_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 420_000,
                is_system_transaction: false,
                input: Self::cross_l2_inbox_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::upgrade_cross_l2_inbox_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Predeploys::CROSS_L2_INBOX),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_CROSS_L2_INBOX_IMPL),
            },
            TxDeposit {
                source_hash: Self::deploy_l2_to_l2_xdm_source(),
                from: Self::L2_TO_L2_XDM_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 1_100_000,
                is_system_transaction: false,
                input: Self::l2_to_l2_xdm_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::upgrade_l2_to_l2_xdm_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Predeploys::L2_TO_L2_XDM),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_L2_TO_L2_XDM_IMPL),
            },
            TxDeposit {
                source_hash: Self::deploy_superchain_eth_bridge_source(),
                from: Self::SUPERCHAIN_ETH_BRIDGE_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 500_000,
                is_system_transaction: false,
                input: Self::superchain_eth_bridge_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::upgrade_superchain_eth_bridge_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Self::SUPERCHAIN_ETH_BRIDGE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_SUPERCHAIN_ETH_BRIDGE_IMPL),
            },
            TxDeposit {
                source_hash: Self::deploy_eth_liquidity_source(),
                from: Self::ETH_LIQUIDITY_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 375_000,
                is_system_transaction: false,
                input: Self::eth_liquidity_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::upgrade_eth_liquidity_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Self::ETH_LIQUIDITY),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_ETH_LIQUIDITY_IMPL),
            },
            TxDeposit {
                source_hash: Self::fund_eth_liquidity_source(),
                from: Self::DEPOSITOR_ACCOUNT,
                to: TxKind::Call(Self::ETH_LIQUIDITY),
                mint: u128::MAX,
                value: U256::from(u128::MAX),
                gas_limit: 50_000,
                is_system_transaction: false,
                input: Self::eth_liquidity_fund_calldata(),
            },
        ])
        .into_iter()
    }
}

impl Hardfork for Interop {
    /// Constructs the network upgrade transactions.
    fn txs(&self) -> impl Iterator<Item = Bytes> {
        Self::deposits().map(|tx| tx.encoded_2718().into())
    }
}

#[cfg(test)]
mod test {
    use alloc::{vec, vec::Vec};

    use super::*;
    use crate::{test_utils::check_deployment_code, upgrade_to_calldata};

    #[test]
    fn test_deploy_cross_l2_inbox_source() {
        assert_eq!(
            Interop::deploy_cross_l2_inbox_source(),
            b256!("0x6e5e214f73143df8fe6f6054a3ed7eb472d373376458a9c8aecdf23475beb616")
        );
    }

    #[test]
    fn test_upgrade_cross_l2_inbox_proxy_source() {
        assert_eq!(
            Interop::upgrade_cross_l2_inbox_proxy_source(),
            b256!("0x88c6b48354c367125a59792a93a7b60ad7cd66e516157dbba16558c68a46d3cb")
        );
    }

    #[test]
    fn test_deploy_l2_to_l2_xdm_source() {
        assert_eq!(
            Interop::deploy_l2_to_l2_xdm_source(),
            b256!("0xf5484697c7a9a791db32a3bf0763bf2ba686c77ae7d4c0a5ee8c222a92a8dcc2")
        );
    }

    #[test]
    fn test_upgrade_l2_to_l2_xdm_proxy_source() {
        assert_eq!(
            Interop::upgrade_l2_to_l2_xdm_proxy_source(),
            b256!("0xe54b4d06bbcc857f41ae00e89d820339ac5ce0034aac722c817b2873e03a7e68")
        );
    }

    #[test]
    fn test_deploy_superchain_eth_bridge_source() {
        assert_eq!(
            Interop::deploy_superchain_eth_bridge_source(),
            b256!("0x53eccc738e298d613b3c3dcc8ad1d9e9626945a2f7b005252c2b57837176d960")
        );
    }

    #[test]
    fn test_upgrade_superchain_eth_bridge_proxy_source() {
        assert_eq!(
            Interop::upgrade_superchain_eth_bridge_proxy_source(),
            b256!("0x50684989256294e3c64949ea1cf5bad586c7e6b91b8b7f21ee9ef7086efe60db")
        );
    }

    #[test]
    fn test_deploy_eth_liquidity_source() {
        assert_eq!(
            Interop::deploy_eth_liquidity_source(),
            b256!("0xceec4ed75501efd5830d25045e10014464155345d91a8c78dba77aed02d5b08b")
        );
    }

    #[test]
    fn test_upgrade_eth_liquidity_proxy_source() {
        assert_eq!(
            Interop::upgrade_eth_liquidity_proxy_source(),
            b256!("0x8c6c281c65cba9a9286233c61c3a1b4d606b899b1aee3b3a7221fd5212b22822")
        );
    }

    #[test]
    fn test_fund_eth_liquidity_source() {
        assert_eq!(
            Interop::fund_eth_liquidity_source(),
            b256!("0xa9b2a45c225d10db0a0a092d024192968cef10170a82f9d67d2bf0264d0c0555")
        );
    }

    #[test]
    fn test_deploy_cross_l2_inbox_address_and_code() {
        let txs = Interop::deposits().collect::<Vec<_>>();
        check_deployment_code(
            txs[0].clone(),
            Interop::NEW_CROSS_L2_INBOX_IMPL,
            Interop::CROSS_L2_INBOX_IMPL_CODE_HASH,
        );
    }

    #[test]
    fn test_deploy_l2_to_l2_xdm_address_and_code() {
        let txs = Interop::deposits().collect::<Vec<_>>();
        check_deployment_code(
            txs[2].clone(),
            Interop::NEW_L2_TO_L2_XDM_IMPL,
            Interop::L2_TO_L2_XDM_IMPL_CODE_HASH,
        );
    }

    #[test]
    fn test_interop_txs_encoded() {
        let interop_upgrade_tx = Interop.txs().collect::<Vec<_>>();
        assert_eq!(interop_upgrade_tx.len(), 9);

        let expected_txs: Vec<Bytes> = vec![
            hex::decode(include_str!("./bytecode/interop_tx_0.hex").replace('\n', ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_1.hex").replace('\n', ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_2.hex").replace('\n', ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_3.hex").replace('\n', ""))
                .unwrap()
                .into(),
            TxDeposit {
                source_hash: Interop::deploy_superchain_eth_bridge_source(),
                from: Interop::SUPERCHAIN_ETH_BRIDGE_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 500_000,
                is_system_transaction: false,
                input: Interop::superchain_eth_bridge_deployment_bytecode(),
            }
            .encoded_2718()
            .into(),
            TxDeposit {
                source_hash: Interop::upgrade_superchain_eth_bridge_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Interop::SUPERCHAIN_ETH_BRIDGE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: upgrade_to_calldata(Interop::NEW_SUPERCHAIN_ETH_BRIDGE_IMPL),
            }
            .encoded_2718()
            .into(),
            TxDeposit {
                source_hash: Interop::deploy_eth_liquidity_source(),
                from: Interop::ETH_LIQUIDITY_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 375_000,
                is_system_transaction: false,
                input: Interop::eth_liquidity_deployment_bytecode(),
            }
            .encoded_2718()
            .into(),
            TxDeposit {
                source_hash: Interop::upgrade_eth_liquidity_proxy_source(),
                from: Address::ZERO,
                to: TxKind::Call(Interop::ETH_LIQUIDITY),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: upgrade_to_calldata(Interop::NEW_ETH_LIQUIDITY_IMPL),
            }
            .encoded_2718()
            .into(),
            TxDeposit {
                source_hash: Interop::fund_eth_liquidity_source(),
                from: Interop::DEPOSITOR_ACCOUNT,
                to: TxKind::Call(Interop::ETH_LIQUIDITY),
                mint: u128::MAX,
                value: U256::from(u128::MAX),
                gas_limit: 50_000,
                is_system_transaction: false,
                input: Interop::eth_liquidity_fund_calldata(),
            }
            .encoded_2718()
            .into(),
        ];
        for (i, expected) in expected_txs.iter().enumerate() {
            assert_eq!(interop_upgrade_tx[i], *expected);
        }
    }
}
