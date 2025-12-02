//! Module containing a [`TxDeposit`] builder for the Ecotone network upgrade transactions.

use alloc::{string::String, vec::Vec};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, address, hex};
use kona_protocol::Predeploys;
use op_alloy_consensus::{TxDeposit, UpgradeDepositSource};

use crate::Hardfork;

/// The Ecotone network upgrade transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct Ecotone;

impl Ecotone {
    /// The Gas Price Oracle Address
    /// This is computed by using go-ethereum's `crypto.CreateAddress` function,
    /// with the Gas Price Oracle Deployer Address and nonce 0.
    pub const GAS_PRICE_ORACLE: Address = address!("b528d11cc114e026f138fe568744c6d45ce6da7a");

    /// The depositor account address.
    pub const DEPOSITOR_ACCOUNT: Address = address!("DeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

    /// The Enable Ecotone Input Method 4Byte Signature
    pub const ENABLE_ECOTONE_INPUT: [u8; 4] = hex!("22b90ab3");

    /// L1 Block Deployer Address
    pub const L1_BLOCK_DEPLOYER: Address = address!("4210000000000000000000000000000000000000");

    /// The Gas Price Oracle Deployer Address
    pub const GAS_PRICE_ORACLE_DEPLOYER: Address =
        address!("4210000000000000000000000000000000000001");

    /// The new L1 Block Address
    /// This is computed by using go-ethereum's `crypto.CreateAddress` function,
    /// with the L1 Block Deployer Address and nonce 0.
    pub const NEW_L1_BLOCK: Address = address!("07dbe8500fc591d1852b76fee44d5a05e13097ff");

    /// EIP-4788 From Address
    pub const EIP4788_FROM: Address = address!("0B799C86a49DEeb90402691F1041aa3AF2d3C875");

    /// The L1 Block Deployer Code Hash
    /// See: <https://specs.optimism.io/protocol/ecotone/derivation.html#l1block-deployment>
    pub const L1_BLOCK_DEPLOYER_CODE_HASH: B256 = alloy_primitives::b256!(
        "0xc88a313aa75dc4fbf0b6850d9f9ae41e04243b7008cf3eadb29256d4a71c1dfd"
    );
    /// The Gas Price Oracle Code Hash
    /// See: <https://specs.optimism.io/protocol/ecotone/derivation.html#gaspriceoracle-deployment>
    pub const GAS_PRICE_ORACLE_CODE_HASH: B256 = alloy_primitives::b256!(
        "0x8b71360ea773b4cfaf1ae6d2bd15464a4e1e2e360f786e475f63aeaed8da0ae5"
    );

    /// Returns the source hash for the deployment of the l1 block contract.
    pub fn deploy_l1_block_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Ecotone: L1 Block Deployment") }.source_hash()
    }

    /// Returns the source hash for the deployment of the gas price oracle contract.
    pub fn deploy_gas_price_oracle_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Ecotone: Gas Price Oracle Deployment") }
            .source_hash()
    }

    /// Returns the source hash for the update of the l1 block proxy.
    pub fn update_l1_block_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Ecotone: L1 Block Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the update of the gas price oracle proxy.
    pub fn update_gas_price_oracle_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Ecotone: Gas Price Oracle Proxy Update") }
            .source_hash()
    }

    /// Returns the source hash for the Ecotone Beacon Block Roots Contract deployment.
    pub fn beacon_roots_source() -> B256 {
        UpgradeDepositSource {
            intent: String::from("Ecotone: beacon block roots contract deployment"),
        }
        .source_hash()
    }

    /// Returns the source hash for the Ecotone Gas Price Oracle activation.
    pub fn enable_ecotone_source() -> B256 {
        UpgradeDepositSource { intent: String::from("Ecotone: Gas Price Oracle Set Ecotone") }
            .source_hash()
    }

    /// Returns the EIP-4788 creation data.
    pub fn eip4788_creation_data() -> Bytes {
        hex::decode(include_str!("./bytecode/eip4788_ecotone.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the raw bytecode for the L1 Block deployment.
    pub fn l1_block_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/l1_block_ecotone.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the gas price oracle deployment bytecode.
    pub fn ecotone_gas_price_oracle_deployment_bytecode() -> Bytes {
        hex::decode(include_str!("./bytecode/gpo_ecotone.hex").replace("\n", ""))
            .expect("Expected hex byte string")
            .into()
    }

    /// Returns the list of [`TxDeposit`]s for the Ecotone network upgrade.
    pub fn deposits() -> impl Iterator<Item = TxDeposit> {
        ([
            // Deploy the L1 Block contract for Ecotone.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#l1block-deployment>
            TxDeposit {
                source_hash: Self::deploy_l1_block_source(),
                from: Self::L1_BLOCK_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 375_000,
                is_system_transaction: false,
                input: Self::l1_block_deployment_bytecode(),
            },
            // Deploy the Gas Price Oracle contract for Ecotone.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#gaspriceoracle-deployment>
            TxDeposit {
                source_hash: Self::deploy_gas_price_oracle_source(),
                from: Self::GAS_PRICE_ORACLE_DEPLOYER,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 1_000_000,
                is_system_transaction: false,
                input: Self::ecotone_gas_price_oracle_deployment_bytecode(),
            },
            // Updates the l1 block proxy to point to the new L1 Block contract.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#l1block-proxy-update>
            TxDeposit {
                source_hash: Self::update_l1_block_source(),
                from: Address::ZERO,
                to: TxKind::Call(Predeploys::L1_BLOCK_INFO),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::NEW_L1_BLOCK),
            },
            // Updates the gas price oracle proxy to point to the new Gas Price Oracle contract.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#gaspriceoracle-proxy-update>
            TxDeposit {
                source_hash: Self::update_gas_price_oracle_source(),
                from: Address::ZERO,
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 50_000,
                is_system_transaction: false,
                input: super::upgrade_to_calldata(Self::GAS_PRICE_ORACLE),
            },
            // Enables the Ecotone Gas Price Oracle.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#gaspriceoracle-enable-ecotone>
            TxDeposit {
                source_hash: Self::enable_ecotone_source(),
                from: Self::DEPOSITOR_ACCOUNT,
                to: TxKind::Call(Predeploys::GAS_PRICE_ORACLE),
                mint: 0,
                value: U256::ZERO,
                gas_limit: 80_000,
                is_system_transaction: false,
                input: Self::ENABLE_ECOTONE_INPUT.into(),
            },
            // Deploys the beacon block roots contract.
            // See: <https://specs.optimism.io/protocol/ecotone/derivation.html#beacon-block-roots-contract-deployment-eip-4788>
            TxDeposit {
                source_hash: Self::beacon_roots_source(),
                from: Self::EIP4788_FROM,
                to: TxKind::Create,
                mint: 0,
                value: U256::ZERO,
                gas_limit: 250_000,
                is_system_transaction: false,
                input: Self::eip4788_creation_data(),
            },
        ])
        .into_iter()
    }
}

impl Hardfork for Ecotone {
    /// Constructs the Ecotone network upgrade transactions.
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

    #[test]
    fn test_deploy_l1_block_source() {
        assert_eq!(
            Ecotone::deploy_l1_block_source(),
            hex!("877a6077205782ea15a6dc8699fa5ebcec5e0f4389f09cb8eda09488231346f8")
        );
    }
    #[test]
    fn test_verify_ecotone_l1_deployment_code_hash() {
        let txs = Ecotone::deposits().collect::<Vec<_>>();

        check_deployment_code(
            txs[0].clone(),
            Ecotone::NEW_L1_BLOCK,
            Ecotone::L1_BLOCK_DEPLOYER_CODE_HASH,
        );
    }

    #[test]
    fn test_verify_ecotone_gas_price_oracle_deployment_code_hash() {
        let txs = Ecotone::deposits().collect::<Vec<_>>();

        check_deployment_code(
            txs[1].clone(),
            Ecotone::GAS_PRICE_ORACLE,
            Ecotone::GAS_PRICE_ORACLE_CODE_HASH,
        );
    }

    #[test]
    fn test_deploy_gas_price_oracle_source() {
        assert_eq!(
            Ecotone::deploy_gas_price_oracle_source(),
            hex!("a312b4510adf943510f05fcc8f15f86995a5066bd83ce11384688ae20e6ecf42")
        );
    }

    #[test]
    fn test_update_l1_block_source() {
        assert_eq!(
            Ecotone::update_l1_block_source(),
            hex!("18acb38c5ff1c238a7460ebc1b421fa49ec4874bdf1e0a530d234104e5e67dbc")
        );
    }

    #[test]
    fn test_update_gas_price_oracle_source() {
        assert_eq!(
            Ecotone::update_gas_price_oracle_source(),
            hex!("ee4f9385eceef498af0be7ec5862229f426dec41c8d42397c7257a5117d9230a")
        );
    }

    #[test]
    fn test_enable_ecotone_source() {
        assert_eq!(
            Ecotone::enable_ecotone_source(),
            hex!("0c1cb38e99dbc9cbfab3bb80863380b0905290b37eb3d6ab18dc01c1f3e75f93")
        );
    }

    #[test]
    fn test_beacon_block_roots_source() {
        assert_eq!(
            Ecotone::beacon_roots_source(),
            hex!("69b763c48478b9dc2f65ada09b3d92133ec592ea715ec65ad6e7f3dc519dc00c")
        );
    }

    #[test]
    fn test_ecotone_txs_encoded() {
        let ecotone_upgrade_tx = Ecotone.txs().collect::<Vec<_>>();
        assert_eq!(ecotone_upgrade_tx.len(), 6);

        let expected_txs: Vec<Bytes> = vec![
            hex::decode(include_str!("./bytecode/ecotone_tx_0.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/ecotone_tx_1.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/ecotone_tx_2.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/ecotone_tx_3.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/ecotone_tx_4.hex").replace("\n", ""))
                .unwrap()
                .into(),
            hex::decode(include_str!("./bytecode/ecotone_tx_5.hex").replace("\n", ""))
                .unwrap()
                .into(),
        ];
        for (i, expected) in expected_txs.iter().enumerate() {
            assert_eq!(ecotone_upgrade_tx[i], *expected);
        }
    }
}
