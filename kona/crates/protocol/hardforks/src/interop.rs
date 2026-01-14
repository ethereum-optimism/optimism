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
    /// The deployer of the `CrossL2Inbox` contract.
    pub const CROSS_L2_INBOX_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000000");

    /// The deployer of the `L2ToL2CrossDomainMessenger` contract.
    pub const L2_TO_L2_XDM_DEPLOYER: Address =
        address!("0x4220000000000000000000000000000000000001");

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

    /// Returns the `CrossL2Inbox` deployment bytecode.
    pub fn cross_l2_inbox_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/crossl2inbox_interop.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the `L2ToL2CrossDomainMessenger` proxy upgrade bytecode.
    pub fn l2_to_l2_xdm_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/l2tol2_xdm_interop.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
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
    use crate::test_utils::check_deployment_code;

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
        assert_eq!(interop_upgrade_tx.len(), 4);

        let expected_txs: Vec<Bytes> = vec![
            hex::decode(include_str!("./bytecode/interop_tx_0.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_1.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_2.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/interop_tx_3.hex").replace("\n", ""))
                .unwrap()
                .into(),
        ];
        for (i, expected) in expected_txs.iter().enumerate() {
            assert_eq!(interop_upgrade_tx[i], *expected);
        }
    }
}
