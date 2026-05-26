use crate::{
    tests::{
        BUILDER_PRIVATE_KEY, COMMIT_HASH, FLASHBLOCKS_DEPLOY_KEY, FLASHTESTATION_DEPLOY_KEY,
        Protocol, SOURCE_LOCATORS, WORKLOAD_ID, block_builder_policy::BlockBuilderPolicy,
        flashblocks_number_contract::FlashblocksNumber,
        flashtestation_registry::FlashtestationRegistry, framework::driver::ChainDriver,
        mock_dcap_attestation::MockAutomataDcapAttestationFee,
    },
    tx_signer::Signer,
};
use alloy_eips::Encodable2718;
use alloy_primitives::{Address, B256, BlockHash, TxHash, TxKind, U256, hex};
use alloy_rpc_types_eth::{Block, BlockTransactionHashes};
use alloy_sol_types::SolCall;
use core::future::Future;
use op_alloy_consensus::{OpTypedTransaction, TxDeposit};
use op_alloy_rpc_types::Transaction;
use reth_db::{
    ClientVersion, DatabaseEnv, init_db,
    mdbx::{DatabaseArguments, KILOBYTE, MEGABYTE, MaxReadTransactionDuration},
    test_utils::{ERROR_DB_CREATION, TempDatabase},
};
use reth_node_core::{args::DatadirArgs, dirs::DataDirPath, node_config::NodeConfig};
use reth_optimism_chainspec::OpChainSpec;
use std::{net::TcpListener, sync::Arc};

use super::{FUNDED_PRIVATE_KEY, TransactionBuilder};

pub trait TransactionBuilderExt {
    fn random_valid_transfer(self) -> Self;
    fn random_reverting_transaction(self) -> Self;
    fn random_big_transaction(self) -> Self;
    // flashblocks number methods
    fn deploy_flashblock_number_contract(self) -> Self;
    fn init_flashblock_number_contract(self, register_builder: bool) -> Self;
    fn add_authorized_builder(self, builder: Address) -> Self;
    // flashtestations methods
    fn deploy_flashtestation_registry_contract(self) -> Self;
    fn init_flashtestation_registry_contract(self, dcap_address: Address) -> Self;
    fn deploy_builder_policy_contract(self) -> Self;
    fn init_builder_policy_contract(self, registry_address: Address) -> Self;
    fn add_workload_to_policy(self) -> Self;
    fn deploy_mock_dcap_contract(self) -> Self;
    fn add_mock_quote(self) -> Self;
}

impl TransactionBuilderExt for TransactionBuilder {
    fn random_valid_transfer(self) -> Self {
        self.with_to(rand::random::<Address>()).with_value(1)
    }

    fn random_reverting_transaction(self) -> Self {
        self.with_create().with_input(hex!("60006000fd").into()) // PUSH1 0x00 PUSH1 0x00 REVERT
    }

    // This transaction is big in the sense that it uses a lot of gas. The exact
    // amount it uses is 86220 gas.
    fn random_big_transaction(self) -> Self {
        // PUSH13 0x63ffffffff60005260046000f3 PUSH1 0x00 MSTORE PUSH1 0x02 PUSH1 0x0d PUSH1 0x13 PUSH1 0x00 CREATE2
        self.with_create()
            .with_input(hex!("6c63ffffffff60005260046000f36000526002600d60136000f5").into())
    }

    fn deploy_flashblock_number_contract(self) -> Self {
        self.with_create()
            .with_input(FlashblocksNumber::BYTECODE.clone())
            .with_gas_limit(2_000_000) // deployment costs ~1.6 million gas
            .with_signer(flashblocks_number_signer())
    }

    fn init_flashblock_number_contract(self, register_builder: bool) -> Self {
        let builder_signer = builder_signer();
        let owner = flashblocks_number_signer();

        let init_data = FlashblocksNumber::initializeCall {
            _owner: owner.address,
            _initialBuilders: if register_builder {
                vec![builder_signer.address]
            } else {
                vec![]
            },
        }
        .abi_encode();

        self.with_input(init_data.into())
            .with_signer(flashblocks_number_signer())
    }

    fn add_authorized_builder(self, builder: Address) -> Self {
        let calldata = FlashblocksNumber::addBuilderCall { builder }.abi_encode();

        self.with_input(calldata.into())
            .with_signer(flashblocks_number_signer())
    }

    fn deploy_flashtestation_registry_contract(self) -> Self {
        self.with_create()
            .with_input(FlashtestationRegistry::BYTECODE.clone())
            .with_gas_limit(5_000_000)
            .with_signer(flashtestations_signer())
    }

    fn init_flashtestation_registry_contract(self, dcap_address: Address) -> Self {
        let owner = flashtestations_signer();

        let init_data = FlashtestationRegistry::initializeCall {
            owner: owner.address,
            _attestationContract: dcap_address,
        }
        .abi_encode();

        self.with_input(init_data.into()).with_signer(owner)
    }

    fn deploy_builder_policy_contract(self) -> Self {
        self.with_create()
            .with_input(BlockBuilderPolicy::BYTECODE.clone())
            .with_gas_limit(3_000_000)
            .with_signer(flashtestations_signer())
    }

    fn init_builder_policy_contract(self, registry_address: Address) -> Self {
        let owner = flashtestations_signer();

        let init_data = BlockBuilderPolicy::initializeCall {
            _initialOwner: owner.address,
            _registry: registry_address,
        }
        .abi_encode();

        self.with_input(init_data.into())
            .with_signer(flashtestations_signer())
    }

    fn add_workload_to_policy(self) -> Self {
        let workload = BlockBuilderPolicy::addWorkloadToPolicyCall {
            workloadId: WORKLOAD_ID,
            commitHash: COMMIT_HASH.to_string(),
            sourceLocators: SOURCE_LOCATORS
                .iter()
                .map(|source| source.to_string())
                .collect(),
        }
        .abi_encode();

        self.with_input(workload.into())
            .with_signer(flashtestations_signer())
    }

    fn deploy_mock_dcap_contract(self) -> Self {
        self.with_create()
            .with_input(MockAutomataDcapAttestationFee::BYTECODE.clone())
            .with_gas_limit(1_000_000)
            .with_signer(flashtestations_signer())
    }

    fn add_mock_quote(self) -> Self {
        let quote = MockAutomataDcapAttestationFee::setQuoteResultCall {
            // quote from http://ns31695324.ip-141-94-163.eu:10080/attest for builder key
            rawQuote: include_bytes!("./artifacts/test-quote.bin").into(),
            _success: true,
            // response from verifyAndAttestOnChain from the real automata dcap contract on
            // unichain sepolia 0x95175096a9B74165BE0ac84260cc14Fc1c0EF5FF
            _output: include_bytes!("./artifacts/quote-output.bin").into(),
        }
        .abi_encode();
        self.with_input(quote.into())
            .with_gas_limit(500_000)
            .with_signer(flashtestations_signer())
    }
}

pub trait ChainDriverExt {
    fn fund_many(
        &self,
        addresses: Vec<Address>,
        amount: u128,
    ) -> impl Future<Output = eyre::Result<BlockHash>>;
    fn fund(&self, address: Address, amount: u128)
    -> impl Future<Output = eyre::Result<BlockHash>>;

    fn fund_accounts(
        &self,
        count: usize,
        amount: u128,
    ) -> impl Future<Output = eyre::Result<Vec<Signer>>> {
        async move {
            let accounts = (0..count).map(|_| Signer::random()).collect::<Vec<_>>();
            self.fund_many(accounts.iter().map(|a| a.address).collect(), amount)
                .await?;
            Ok(accounts)
        }
    }

    fn build_new_block_with_valid_transaction(
        &self,
    ) -> impl Future<Output = eyre::Result<(TxHash, Block<Transaction>)>>;

    fn build_new_block_with_reverting_transaction(
        &self,
    ) -> impl Future<Output = eyre::Result<(TxHash, Block<Transaction>)>>;
}

impl<P: Protocol> ChainDriverExt for ChainDriver<P> {
    async fn fund_many(&self, addresses: Vec<Address>, amount: u128) -> eyre::Result<BlockHash> {
        let mut txs = Vec::with_capacity(addresses.len());

        for address in addresses {
            let deposit = TxDeposit {
                source_hash: B256::default(),
                from: address, // Set the sender to the address of the account to seed
                to: TxKind::Create,
                mint: amount, // Amount to deposit
                value: U256::default(),
                gas_limit: 210000,
                is_system_transaction: false,
                input: Default::default(), // No input data for the deposit
            };

            let signer = Signer::random();
            let signed_tx = signer.sign_tx(OpTypedTransaction::Deposit(deposit))?;
            let signed_tx_rlp = signed_tx.encoded_2718();
            txs.push(signed_tx_rlp.into());
        }

        Ok(self.build_new_block_with_txs(txs).await?.header.hash)
    }

    async fn fund(&self, address: Address, amount: u128) -> eyre::Result<BlockHash> {
        let deposit = TxDeposit {
            source_hash: B256::default(),
            from: address, // Set the sender to the address of the account to seed
            to: TxKind::Create,
            mint: amount, // Amount to deposit
            value: U256::default(),
            gas_limit: 210000,
            is_system_transaction: false,
            input: Default::default(), // No input data for the deposit
        };

        let signer = Signer::random();
        let signed_tx = signer.sign_tx(OpTypedTransaction::Deposit(deposit))?;
        let signed_tx_rlp = signed_tx.encoded_2718();
        Ok(self
            .build_new_block_with_txs(vec![signed_tx_rlp.into()])
            .await?
            .header
            .hash)
    }

    async fn build_new_block_with_valid_transaction(
        &self,
    ) -> eyre::Result<(TxHash, Block<Transaction>)> {
        let tx = self
            .create_transaction()
            .random_valid_transfer()
            .send()
            .await?;
        Ok((*tx.tx_hash(), self.build_new_block().await?))
    }

    async fn build_new_block_with_reverting_transaction(
        &self,
    ) -> eyre::Result<(TxHash, Block<Transaction>)> {
        let tx = self
            .create_transaction()
            .random_reverting_transaction()
            .send()
            .await?;

        Ok((*tx.tx_hash(), self.build_new_block().await?))
    }
}

pub trait BlockTransactionsExt {
    fn includes(&self, txs: &impl AsTxs) -> bool;
}

impl BlockTransactionsExt for Block<Transaction> {
    fn includes(&self, txs: &impl AsTxs) -> bool {
        txs.as_txs()
            .into_iter()
            .all(|tx| self.transactions.hashes().any(|included| included == tx))
    }
}

impl BlockTransactionsExt for BlockTransactionHashes<'_, Transaction> {
    fn includes(&self, txs: &impl AsTxs) -> bool {
        let mut included_tx_iter = self.clone();
        txs.as_txs()
            .iter()
            .all(|tx| included_tx_iter.any(|included| included == *tx))
    }
}

pub trait OpRbuilderArgsTestExt {
    fn test_default() -> Self;
}

impl OpRbuilderArgsTestExt for crate::args::OpRbuilderArgs {
    fn test_default() -> Self {
        let mut default = Self::default();
        default.flashblocks.flashblocks_port = 0; // randomize port
        default
    }
}

pub trait AsTxs {
    fn as_txs(&self) -> Vec<TxHash>;
}

impl AsTxs for TxHash {
    fn as_txs(&self) -> Vec<TxHash> {
        vec![*self]
    }
}

impl AsTxs for Vec<TxHash> {
    fn as_txs(&self) -> Vec<TxHash> {
        self.clone()
    }
}

pub fn create_test_db(config: NodeConfig<OpChainSpec>) -> Arc<TempDatabase<DatabaseEnv>> {
    let path = reth_node_core::dirs::MaybePlatformPath::<DataDirPath>::from(
        reth_db::test_utils::tempdir_path(),
    );
    let db_config = config.with_datadir_args(DatadirArgs {
        datadir: path.clone(),
        ..Default::default()
    });
    let data_dir = path.unwrap_or_chain_default(db_config.chain.chain(), db_config.datadir.clone());
    let path = data_dir.db();
    let db = init_db(
        path.as_path(),
        DatabaseArguments::new(ClientVersion::default())
            .with_max_read_transaction_duration(Some(MaxReadTransactionDuration::Unbounded))
            .with_geometry_max_size(Some(4 * MEGABYTE))
            .with_growth_step(Some(4 * KILOBYTE)),
    )
    .expect(ERROR_DB_CREATION);
    Arc::new(TempDatabase::new(db, path))
}

/// Gets an available port by first binding to port 0 -- instructing the OS to
/// find and assign one. Then the listener is dropped when this goes out of
/// scope, freeing the port for the next time this function is called.
pub fn get_available_port() -> u16 {
    TcpListener::bind("127.0.0.1:0")
        .expect("Failed to bind to random port")
        .local_addr()
        .expect("Failed to get local address")
        .port()
}

pub fn builder_signer() -> Signer {
    Signer::try_from_secret(
        BUILDER_PRIVATE_KEY
            .parse()
            .expect("invalid hardcoded builder private key"),
    )
    .expect("Failed to create signer from hardcoded builder private key")
}

pub fn funded_signer() -> Signer {
    Signer::try_from_secret(
        FUNDED_PRIVATE_KEY
            .parse()
            .expect("invalid hardcoded funded private key"),
    )
    .expect("Failed to create signer from hardcoded funded private key")
}

pub fn flashblocks_number_signer() -> Signer {
    Signer::try_from_secret(
        FLASHBLOCKS_DEPLOY_KEY
            .parse()
            .expect("invalid hardcoded flashblocks number deployer private key"),
    )
    .expect("Failed to create signer from hardcoded flashblocks number deployer private key")
}

pub fn flashtestations_signer() -> Signer {
    Signer::try_from_secret(
        FLASHTESTATION_DEPLOY_KEY
            .parse()
            .expect("invalid hardcoded flashtestations deployer private key"),
    )
    .expect("Failed to create signer from hardcoded flashtestations deployer private key")
}
