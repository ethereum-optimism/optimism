//! Module containing a [`TxDeposit`] builder for the Jovian network upgrade transactions.
//!
//! Jovian network upgrade transactions are defined in the [OP Stack Specs][specs].
//!
//! [specs]: https://specs.optimism.io/protocol/jovian/derivation.html#network-upgrade-automation-transactions

use alloc::{string::String, vec::Vec};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, address, hex, keccak256};
use kona_protocol::Predeploys;
use op_alloy_consensus::{TxDeposit, UpgradeDepositSource};

use crate::{Hardfork, upgrade_to_calldata};

/// The Jovian network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Jovian;

impl Jovian {
    /// The depositor account address.
    pub const DEPOSITOR_ACCOUNT: Address = address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

    /// L1 Block Deployer Address
    pub const L1_BLOCK_DEPLOYER: Address = address!("4210000000000000000000000000000000000006");

    /// Zero address
    pub const ZERO_ADDRESS: Address = address!("0x0000000000000000000000000000000000000000");

    /// The Gas Price Oracle Deployer Address
    pub const GAS_PRICE_ORACLE_DEPLOYER: Address =
        address!("4210000000000000000000000000000000000007");

    /// Returns the source hash for the deployment of the l1 block contract.
    pub fn deploy_l1_block_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Jovian: L1 Block Deployment") }.source_hash()
    }

    /// Returns the source hash for the deployment of the gas price oracle contract.
    pub fn l1_block_proxy_update() -> B256 {
        UpgradeDepositSource { intent: String::from("Jovian: L1 Block Proxy Update") }.source_hash()
    }

    /// Returns the source hash for the deployment of the operator fee vault contract.
    pub fn gas_price_oracle() -> B256 {
        UpgradeDepositSource { intent: String::from("Jovian: Gas Price Oracle Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the update of the l1 block proxy.
    pub fn gas_price_oracle_proxy_update() -> B256 {
        UpgradeDepositSource { intent: String::from("Jovian: Gas Price Oracle Proxy Update") }
            .source_hash()
    }

    /// The Jovian L1 Block Address
    /// This is computed by using `Address::create` function,
    /// with the L1 Block Deployer Address and nonce 0.
    pub fn l1_block_address() -> Address {
        Self::L1_BLOCK_DEPLOYER.create(0)
    }

    /// The Jovian Gas Price Oracle Address
    /// This is computed by using `Address::create` function,
    /// with the Gas Price Oracle Deployer Address and nonce 0.
    pub fn gas_price_oracle_address() -> Address {
        Self::GAS_PRICE_ORACLE_DEPLOYER.create(0)
    }

    /// Returns the source hash to the enable the gas price oracle for Jovian.
    pub fn gas_price_oracle_enable_jovian() -> B256 {
        UpgradeDepositSource { intent: String::from("Jovian: Gas Price Oracle Set Jovian") }
            .source_hash()
    }

    /// Returns the raw bytecode for the L1 Block deployment.
    pub fn l1_block_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/jovian-l1-block-deployment.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the gas price oracle deployment bytecode.
    pub fn gas_price_oracle_deployment_bytecode() -> Bytes {
        hex::decode(
            include_str!("./bytecode/jovian-gas-price-oracle-deployment.hex").replace("\n", ""),
        )
        .expect("Expected hex byte string")
        .into()
    }

    /// Returns the bytecode to enable the gas price oracle for Jovian.
    pub fn gas_price_oracle_enable_jovian_bytecode() -> Bytes {
        let mut bytes = Vec::new();
        bytes.extend_from_slice(&keccak256("setJovian()")[..4]);
        bytes.into()
    }

    /// Returns the list of [`TxDeposit`]s for the network upgrade.
    pub fn deposits() -> impl Iterator<Item = TxDeposit> {
        ([
            TxDeposit {
                source_hash: Self::deploy_l1_block_source(),
                from: Self::L1_BLOCK_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 447_315,
                is_system_transaction: false,
                input: Self::l1_block_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::l1_block_proxy_update(),
                from: Self::ZERO_ADDRESS,
                to: TxKind::Call(Predeploys::L1_BLOCK_INFO),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: upgrade_to_calldata(Self::l1_block_address()),
            },
            TxDeposit {
                source_hash: Self::gas_price_oracle(),
                from: Self::GAS_PRICE_ORACLE_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 1_750_714,
                is_system_transaction: false,
                input: Self::gas_price_oracle_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::gas_price_oracle_proxy_update(),
                from: Self::ZERO_ADDRESS,
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: upgrade_to_calldata(Self::gas_price_oracle_address()),
            },
            TxDeposit {
                source_hash: Self::gas_price_oracle_enable_jovian(),
                from: Self::DEPOSITOR_ACCOUNT,
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 90_000,
                is_system_transaction: false,
                input: Self::gas_price_oracle_enable_jovian_bytecode(),
            },
        ])
        .into_iter()
    }
}

impl Hardfork for Jovian {
    /// Constructs the network upgrade transactions.
    fn txs(&self) -> impl Iterator<Item = Bytes> + '_ {
        Self::deposits().map(|tx| {
            let mut encoded = Vec::new();
            tx.encode_2718(&mut encoded);
            Bytes::from(encoded)
        })
    }
}

#[cfg(test)]
mod tests {
    use crate::test_utils::check_deployment_code;

    use super::*;
    use alloy_primitives::b256;

    #[test]
    fn test_l1_block_source_hash() {
        let expected = b256!("bb1a656f65401240fac3db12e7a79ebb954b11e62f7626eb11691539b798d3bf");
        assert_eq!(Jovian::deploy_l1_block_source(), expected);
    }

    #[test]
    fn test_l1_block_proxy_update_source_hash() {
        let expected = b256!("f3275f829340521028f9ad5bce4ecb1c64a45d448794effa2a77674627338e76");
        assert_eq!(Jovian::l1_block_proxy_update(), expected);
    }

    #[test]
    fn test_gas_price_oracle_source_hash() {
        let expected = b256!("239b7021a6c2cf3a918481242bbb5a9499057f24501539467536c691bb133962");
        assert_eq!(Jovian::gas_price_oracle(), expected);
    }

    #[test]
    fn test_upgrade_to_calldata_for_gas_price_oracle() {
        assert_eq!(
            **upgrade_to_calldata(Jovian::gas_price_oracle_address()),
            hex!("0x3659cfe60000000000000000000000004f1db3c6abd250ba86e0928471a8f7db3afd88f1")
        );
    }

    #[test]
    fn test_upgrade_to_calldata_for_l1_block_proxy_update() {
        assert_eq!(
            **upgrade_to_calldata(Jovian::l1_block_address()),
            hex!("0x3659cfe60000000000000000000000003ba4007f5c922fbb33c454b41ea7a1f11e83df2c")
        );
    }

    #[test]
    fn test_gas_price_oracle_proxy_update_source_hash() {
        let expected = b256!("a70c60aa53b8c1c0d52b39b1e901e7d7c09f7819595cb24048a6bb1983b401ff");
        assert_eq!(Jovian::gas_price_oracle_proxy_update(), expected);
    }

    #[test]
    fn test_gas_price_oracle_enable_jovian_source_hash() {
        let expected = b256!("e836db6a959371756f8941be3e962d000f7e12a32e49e2c9ca42ba177a92716c");
        assert_eq!(Jovian::gas_price_oracle_enable_jovian(), expected);
    }

    #[test]
    fn test_verify_jovian_l1_block_deployment_code_hash() {
        let txs = Jovian::deposits().collect::<Vec<_>>();
        check_deployment_code(
            txs[0].clone(),
            Jovian::l1_block_address(),
            hex!("5f885ca815d2cf27a203123e50b8ae204fdca910b6995d90b2d7700cbb9240d1").into(),
        );
    }

    #[test]
    fn test_verify_set_jovian() {
        let hash = &keccak256("setJovian()")[..4];
        assert_eq!(hash, hex!("0xb3d72079"))
    }

    #[test]
    fn test_verify_jovian_gas_price_oracle_deployment_code_hash() {
        let txs = Jovian::deposits().collect::<Vec<_>>();

        check_deployment_code(
            txs[2].clone(),
            Jovian::gas_price_oracle_address(),
            hex!("e9fc7c96c4db0d6078e3d359d7e8c982c350a513cb2c31121adf5e1e8a446614").into(),
        );
    }
}
