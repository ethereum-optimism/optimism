//! Module containing a [`TxDeposit`] builder for the Isthmus network upgrade transactions.
//!
//! Isthmus network upgrade transactions are defined in the [OP Stack Specs][specs].
//!
//! [specs]: https://specs.optimism.io/protocol/isthmus/derivation.html#network-upgrade-automation-transactions

use alloc::{string::String, vec::Vec};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, address, hex};
use kona_protocol::Predeploys;
use op_alloy_consensus::{TxDeposit, UpgradeDepositSource};

use crate::Hardfork;

/// The Isthmus network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Isthmus;

impl Isthmus {
    /// The depositor account address.
    pub const DEPOSITOR_ACCOUNT: Address = address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

    /// The Enable Isthmus Input Method 4Byte Signature.
    ///
    /// Derive this by running `cast sig "setIsthmus()"`.
    pub const ENABLE_ISTHMUS_INPUT: [u8; 4] = hex!("291b0383");

    /// EIP-2935 From Address
    pub const EIP2935_FROM: Address = address!("3462413Af4609098e1E27A490f554f260213D685");

    /// L1 Block Deployer Address
    pub const L1_BLOCK_DEPLOYER: Address = address!("4210000000000000000000000000000000000003");

    /// The Gas Price Oracle Deployer Address
    pub const GAS_PRICE_ORACLE_DEPLOYER: Address =
        address!("4210000000000000000000000000000000000004");

    /// The Operator Fee Vault Deployer Address
    pub const OPERATOR_FEE_VAULT_DEPLOYER: Address =
        address!("4210000000000000000000000000000000000005");

    /// The new L1 Block Address
    /// This is computed by using go-ethereum's `crypto.CreateAddress` function,
    /// with the L1 Block Deployer Address and nonce 0.
    pub const NEW_L1_BLOCK: Address = address!("ff256497d61dcd71a9e9ff43967c13fde1f72d12");

    /// The Gas Price Oracle Address
    /// This is computed by using go-ethereum's `crypto.CreateAddress` function,
    /// with the Gas Price Oracle Deployer Address and nonce 0.
    pub const GAS_PRICE_ORACLE: Address = address!("93e57a196454cb919193fa9946f14943cf733845");

    /// The Operator Fee Vault Address
    /// This is computed by using go-ethereum's `crypto.CreateAddress` function,
    /// with the Operator Fee Vault Deployer Address and nonce 0.
    pub const OPERATOR_FEE_VAULT: Address = address!("4fa2be8cd41504037f1838bce3bcc93bc68ff537");

    /// The Isthmus L1 Block Deployer Code Hash
    /// See: <https://specs.optimism.io/protocol/isthmus/derivation.html#l1block-deployment>
    pub const L1_BLOCK_DEPLOYER_CODE_HASH: B256 = alloy_primitives::b256!(
        "0x8e3fe7a416d3e5f3b7be74ddd4e7e58e516fa3f80b67c6d930e3cd7297da4a4b"
    );

    /// The Isthmus Gas Price Oracle Code Hash
    /// See: <https://specs.optimism.io/protocol/isthmus/derivation.html#gaspriceoracle-deployment>
    pub const GAS_PRICE_ORACLE_CODE_HASH: B256 = alloy_primitives::b256!(
        "0x4d195a9d7caf9fb6d4beaf80de252c626c853afd5868c4f4f8d19c9d301c2679"
    );
    /// The Isthmus Operator Fee Vault Code Hash
    /// See: <https://specs.optimism.io/protocol/isthmus/derivation.html#operator-fee-vault-deployment>
    pub const OPERATOR_FEE_VAULT_CODE_HASH: B256 = alloy_primitives::b256!(
        "0x57dc55c9c09ca456fa728f253fe7b895d3e6aae0706104935fe87c7721001971"
    );
    /// Returns the source hash for the Isthmus Gas Price Oracle activation.
    pub fn enable_isthmus_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: Gas Price Oracle Set Isthmus") }
            .source_hash()
    }

    /// Returns the source hash for the EIP-2935 block hash history contract deployment.
    pub fn block_hash_history_contract_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: EIP-2935 Contract Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the deployment of the gas price oracle contract.
    pub fn deploy_gas_price_oracle_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: Gas Price Oracle Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the deployment of the l1 block contract.
    pub fn deploy_l1_block_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: L1 Block Deployment") }.source_hash()
    }

    /// Returns the source hash for the deployment of the operator fee vault contract.
    pub fn deploy_operator_fee_vault_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: Operator Fee Vault Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the update of the l1 block proxy.
    pub fn update_l1_block_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: L1 Block Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the update of the gas price oracle proxy.
    pub fn update_gas_price_oracle_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: Gas Price Oracle Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the update of the operator fee vault proxy.
    pub fn update_operator_fee_vault_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Isthmus: Operator Fee Vault Proxy Update") }
            .source_hash()
    }

    /// Returns the raw bytecode for the L1 Block deployment.
    pub fn l1_block_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/l1_block_isthmus.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the gas price oracle deployment bytecode.
    pub fn gas_price_oracle_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/gpo_isthmus.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the gas price oracle deployment bytecode.
    pub fn operator_fee_vault_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/ofv_isthmus.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the EIP-2935 creation data.
    pub fn eip2935_creation_data() -> Bytes {
        hex::decode(include_str!("./bytecode/eip2935_isthmus.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
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
                gas_limit: 425_000,
                is_system_transaction: false,
                input: Self::l1_block_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::deploy_gas_price_oracle_source(),
                from: Self::GAS_PRICE_ORACLE_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 1_625_000,
                is_system_transaction: false,
                input: Self::gas_price_oracle_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::deploy_operator_fee_vault_source(),
                from: Self::OPERATOR_FEE_VAULT_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 500_000,
                is_system_transaction: false,
                input: Self::operator_fee_vault_deployment_bytecode(),
            },
            TxDeposit {
                source_hash: Self::update_l1_block_source(),
                from: Address::default(),
                to: TxKind::Call(Predeploys::L1_BLOCK_INFO),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_L1_BLOCK),
            },
            TxDeposit {
                source_hash: Self::update_gas_price_oracle_source(),
                from: Address::default(),
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::GAS_PRICE_ORACLE),
            },
            TxDeposit {
                source_hash: Self::update_operator_fee_vault_source(),
                from: Address::default(),
                to: TxKind::Call(Predeploys::OPERATOR_FEE_VAULT),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::OPERATOR_FEE_VAULT),
            },
            TxDeposit {
                source_hash: Self::enable_isthmus_source(),
                from: Self::DEPOSITOR_ACCOUNT,
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 90_000,
                is_system_transaction: false,
                input: Self::ENABLE_ISTHMUS_INPUT.into(),
            },
            TxDeposit {
                source_hash: Self::block_hash_history_contract_source(),
                from: Self::EIP2935_FROM,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 250_000,
                is_system_transaction: false,
                input: Self::eip2935_creation_data(),
            },
        ])
        .into_iter()
    }
}

impl Hardfork for Isthmus {
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
    use alloc::vec;
    use alloy_primitives::b256;

    #[test]
    fn test_l1_block_source_hash() {
        let expected = b256!("3b2d0821ca2411ad5cd3595804d1213d15737188ae4cbd58aa19c821a6c211bf");
        assert_eq!(Isthmus::deploy_l1_block_source(), expected);
    }

    #[test]
    fn test_gas_price_oracle_source_hash() {
        let expected = b256!("fc70b48424763fa3fab9844253b4f8d508f91eb1f7cb11a247c9baec0afb8035");
        assert_eq!(Isthmus::deploy_gas_price_oracle_source(), expected);
    }

    #[test]
    fn test_operator_fee_vault_source_hash() {
        let expected = b256!("107a570d3db75e6110817eb024f09f3172657e920634111ce9875d08a16daa96");
        assert_eq!(Isthmus::deploy_operator_fee_vault_source(), expected);
    }

    #[test]
    fn test_l1_block_update_source_hash() {
        let expected = b256!("ebe8b5cb10ca47e0d8bda8f5355f2d66711a54ddeb0ef1d30e29418c9bf17a0e");
        assert_eq!(Isthmus::update_l1_block_source(), expected);
    }

    #[test]
    fn test_gas_price_oracle_update_source_hash() {
        let expected = b256!("ecf2d9161d26c54eda6b7bfdd9142719b1e1199a6e5641468d1bf705bc531ab0");
        assert_eq!(Isthmus::update_gas_price_oracle_source(), expected);
    }

    #[test]
    fn test_operator_fee_vault_update_source_hash() {
        let expected = b256!("ad74e1adb877ccbe176b8fa1cc559388a16e090ddbe8b512f5b37d07d887a927");
        assert_eq!(Isthmus::update_operator_fee_vault_source(), expected);
    }

    #[test]
    fn test_enable_isthmus_source() {
        let expected = b256!("3ddf4b1302548dd92939826e970f260ba36167f4c25f18390a5e8b194b295319");
        assert_eq!(Isthmus::enable_isthmus_source(), expected);
    }

    #[test]
    fn test_isthmus_txs_encoded() {
        let isthmus_upgrade_tx = Isthmus.txs().collect::<Vec<_>>();
        assert_eq!(isthmus_upgrade_tx.len(), 8);

        let expected_txs: Vec<Bytes> = vec![
            hex::decode(include_str!("./bytecode/isthmus_tx_0.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_1.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_2.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_3.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_4.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_5.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_6.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/isthmus_tx_7.hex").replace("\n", ""))
                .unwrap()
                .into(),
        ];
        for (i, expected) in expected_txs.iter().enumerate() {
            assert_eq!(isthmus_upgrade_tx[i], *expected);
        }
    }
    #[test]
    fn test_verify_isthmus_l1_block_deployment_code_hash() {
        let txs = Isthmus::deposits().collect::<Vec<_>>();
        check_deployment_code(
            txs[0].clone(),
            Isthmus::NEW_L1_BLOCK,
            Isthmus::L1_BLOCK_DEPLOYER_CODE_HASH,
        );
    }
    #[test]
    fn test_verify_isthmus_gas_price_oracle_deployment_code_hash() {
        let txs = Isthmus::deposits().collect::<Vec<_>>();

        check_deployment_code(
            txs[1].clone(),
            Isthmus::GAS_PRICE_ORACLE,
            Isthmus::GAS_PRICE_ORACLE_CODE_HASH,
        );
    }
    #[test]
    fn test_verify_isthmus_operator_fee_vault_deployment_code_hash() {
        let txs = Isthmus::deposits().collect::<Vec<_>>();

        check_deployment_code(
            txs[2].clone(),
            Isthmus::OPERATOR_FEE_VAULT,
            Isthmus::OPERATOR_FEE_VAULT_CODE_HASH,
        );
    }
}
