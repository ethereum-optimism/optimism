use alloy_consensus::Transaction;
use alloy_network::TransactionResponse;
use alloy_primitives::{Address, U256};
use alloy_provider::{Provider, RootProvider};
use macros::{if_flashblocks, if_standard, rb_test};
use op_alloy_network::Optimism;

use crate::{
    args::{FlashblocksArgs, OpRbuilderArgs},
    flashtestations::args::FlashtestationsArgs,
    tests::{
        BLOCK_BUILDER_POLICY_ADDRESS, BundleOpts, ChainDriver, ChainDriverExt,
        FLASHBLOCKS_NUMBER_ADDRESS, FLASHTESTATION_REGISTRY_ADDRESS, LocalInstance,
        MOCK_DCAP_ADDRESS, TEE_DEBUG_ADDRESS, TransactionBuilderExt,
        block_builder_policy::BlockBuilderPolicy, builder_signer,
        flashblocks_number_contract::FlashblocksNumber,
        flashtestation_registry::FlashtestationRegistry,
    },
};

#[rb_test(args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_invalid_quote(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashtestation_contracts(&driver, &provider, false, false).await?;
    // verify not registered
    let contract = FlashtestationRegistry::new(FLASHTESTATION_REGISTRY_ADDRESS, provider.clone());
    let result = contract
        .getRegistrationStatus(TEE_DEBUG_ADDRESS)
        .call()
        .await?;
    assert!(
        !result.isValid,
        "The tee key is registered for invalid quote"
    );
    // check that only regular builder tx is in the block
    let (tx_hash, block) = driver.build_new_block_with_valid_transaction().await?;
    let txs = block.transactions.into_transactions_vec();

    if_flashblocks!(
        assert_eq!(txs.len(), 4, "Expected 4 transactions in block"); // deposit + valid tx + 2 builder tx
        // Check builder tx
        assert_eq!(
            txs[1].to(),
            Some(Address::ZERO),
            "builder tx should send to zero address"
        );
    );
    if_standard!(
        assert_eq!(txs.len(), 3, "Expected 3 transactions in block"); // deposit + valid tx + builder tx
    );
    let last_txs = &txs[txs.len() - 2..];
    // Check user transaction
    assert_eq!(
        last_txs[0].inner.tx_hash(),
        tx_hash,
        "tx hash for user transaction should match"
    );
    // Check builder tx
    assert_eq!(
        last_txs[1].to(),
        Some(Address::ZERO),
        "builder tx should send to zero address"
    );
    Ok(())
}

#[rb_test(args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_unauthorized_workload(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashtestation_contracts(&driver, &provider, true, false).await?;
    // check that only the regular builder tx is in the block
    let (tx_hash, block) = driver.build_new_block_with_valid_transaction().await?;
    let txs = block.transactions.into_transactions_vec();
    if_flashblocks!(
        assert_eq!(txs.len(), 4, "Expected 4 transactions in block"); // deposit + valid tx + 2 builder tx
        // Check builder tx
        assert_eq!(
            txs[1].to(),
            Some(Address::ZERO),
            "builder tx should send to zero address"
        );
    );
    if_standard!(
        assert_eq!(txs.len(), 3, "Expected 3 transactions in block"); // deposit + valid tx + builder tx
    );
    let last_txs = &txs[txs.len() - 2..];
    // Check user transaction
    assert_eq!(
        last_txs[0].inner.tx_hash(),
        tx_hash,
        "tx hash for user transaction should match"
    );
    // Check builder tx
    assert_eq!(
        last_txs[1].to(),
        Some(Address::ZERO),
        "builder tx should send to zero address"
    );
    Ok(())
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        flashblocks_number_contract_address: Some(FLASHBLOCKS_NUMBER_ADDRESS),
        ..Default::default()
    },
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_with_number_contract(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashblock_number_contract(&driver, &provider, true).await?;
    setup_flashtestation_contracts(&driver, &provider, true, true).await?;
    let tx = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_min(4))
        .send()
        .await?;
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    // 1 deposit tx, 1 fallback builder tx, 4 flashblocks number tx, valid tx, block proof
    let txs = block.transactions.into_transactions_vec();
    assert_eq!(txs.len(), 8, "Expected 8 transactions in block");
    // Check builder tx
    assert_eq!(
        txs[1].to(),
        Some(Address::ZERO),
        "fallback builder tx should send to zero address"
    );
    // flashblocks number contract
    for i in 2..6 {
        assert_eq!(
            txs[i].to(),
            Some(FLASHBLOCKS_NUMBER_ADDRESS),
            "builder tx should send to flashblocks number contract at index {}",
            i
        );
    }
    // check regular tx
    assert_eq!(
        txs[6].tx_hash(),
        *tx.tx_hash(),
        "bundle tx was not in block"
    );
    // check block proof tx
    assert_eq!(
        txs[7].to(),
        Some(BLOCK_BUILDER_POLICY_ADDRESS),
        "block proof tx should call block policy address"
    );
    // Verify flashblock number incremented correctly
    let contract = FlashblocksNumber::new(FLASHBLOCKS_NUMBER_ADDRESS, provider.clone());
    let current_number = contract.getFlashblockNumber().call().await?;
    assert!(
        current_number.gt(&U256::from(8)), // contract deployments incremented the number but we built at least 2 full blocks
        "Flashblock number not incremented"
    );
    Ok(())
}

#[rb_test(args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_permit_registration(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashtestation_contracts(&driver, &provider, true, true).await?;
    // check builder does not try to register again
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let num_txs = block.transactions.len();
    if_flashblocks!(
        assert!(num_txs == 3, "Expected 3 transactions in block"); // deposit + 2 builder tx
    );
    if_standard!(
        assert!(num_txs == 2, "Expected 2 transactions in block"); // deposit + builder tx
    );
    // check that the tee signer did not send any transactions
    let balance = provider.get_balance(TEE_DEBUG_ADDRESS).await?;
    assert!(balance.is_zero());
    let nonce = provider.get_transaction_count(TEE_DEBUG_ADDRESS).await?;
    assert_eq!(nonce, 0);
    Ok(())
}

#[rb_test(args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_permit_block_proof(rbuilder: LocalInstance) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashtestation_contracts(&driver, &provider, true, true).await?;
    // check builder does not try to register again
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let num_txs = block.transactions.len();
    if_flashblocks!(
        assert!(num_txs == 4, "Expected 4 transactions in block"); // deposit + 2 builder tx + 1 block proof
    );
    if_standard!(
        assert!(num_txs == 3, "Expected 3 transactions in block"); // deposit + 2 builder tx
    );
    let last_2_txs = &block.transactions.into_transactions_vec()[num_txs - 2..];
    // Check builder tx
    assert_eq!(
        last_2_txs[0].to(),
        Some(Address::ZERO),
        "builder tx should send to zero address"
    );
    // check builder proof
    assert_eq!(
        last_2_txs[1].to(),
        Some(BLOCK_BUILDER_POLICY_ADDRESS),
        "builder tx should send to flashtestations builder policy address"
    );
    assert_eq!(
        last_2_txs[1].from(),
        builder_signer().address,
        "block proof tx should come from builder address"
    );
    // check that the tee signer did not send any transactions
    let balance = provider.get_balance(TEE_DEBUG_ADDRESS).await?;
    assert!(balance.is_zero());
    let nonce = provider.get_transaction_count(TEE_DEBUG_ADDRESS).await?;
    assert_eq!(nonce, 0);

    Ok(())
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        flashblocks_number_contract_address: Some(FLASHBLOCKS_NUMBER_ADDRESS),
        ..Default::default()
    },
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_permit_with_flashblocks_number_contract(
    rbuilder: LocalInstance,
) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashtestation_contracts(&driver, &provider, true, true).await?;
    setup_flashblock_number_contract(&driver, &provider, true).await?;
    let tx = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_min(4))
        .send()
        .await?;
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let num_txs = block.transactions.len();
    let txs = block.transactions.into_transactions_vec();
    // // 1 deposit tx, 1 regular builder tx, 4 flashblocks number tx, 1 user tx, 1 block proof tx
    assert_eq!(num_txs, 8, "Expected 8 transactions in block");
    // Check builder tx
    assert_eq!(
        txs[1].to(),
        Some(Address::ZERO),
        "builder tx should send to zero address"
    );
    // flashblocks number contract
    for i in 2..6 {
        assert_eq!(
            txs[i].to(),
            Some(FLASHBLOCKS_NUMBER_ADDRESS),
            "builder tx should send to flashblocks number contract at index {}",
            i
        );
    }
    // user tx
    assert_eq!(
        txs[6].tx_hash(),
        *tx.tx_hash(),
        "user tx should be in correct position in block"
    );
    assert_eq!(
        txs[7].to(),
        Some(BLOCK_BUILDER_POLICY_ADDRESS),
        "builder tx should send verify block builder proof tx"
    );
    // check that the tee signer did not send any transactions
    let balance = provider.get_balance(TEE_DEBUG_ADDRESS).await?;
    assert!(balance.is_zero());
    let nonce = provider.get_transaction_count(TEE_DEBUG_ADDRESS).await?;
    assert_eq!(nonce, 0);
    // Verify flashblock number incremented correctly
    let contract = FlashblocksNumber::new(FLASHBLOCKS_NUMBER_ADDRESS, provider.clone());
    let current_number = contract.getFlashblockNumber().call().await?;
    assert!(
        current_number.gt(&U256::from(4)), // contract deployments incremented the number but we built at least 1 full block
        "Flashblock number not incremented"
    );
    Ok(())
}

#[rb_test(flashblocks, args = OpRbuilderArgs {
    chain_block_time: 1000,
    enable_revert_protection: true,
    flashblocks: FlashblocksArgs {
        flashblocks_number_contract_address: Some(FLASHBLOCKS_NUMBER_ADDRESS),
        flashblocks_number_contract_use_permit: true,
        ..Default::default()
    },
    flashtestations: FlashtestationsArgs {
        flashtestations_enabled: true,
        registry_address: Some(FLASHTESTATION_REGISTRY_ADDRESS),
        builder_policy_address: Some(BLOCK_BUILDER_POLICY_ADDRESS),
        debug: true,
        enable_block_proofs: true,
        ..Default::default()
    },
    ..Default::default()
})]
async fn test_flashtestations_permit_with_flashblocks_number_permit(
    rbuilder: LocalInstance,
) -> eyre::Result<()> {
    let driver = rbuilder.driver().await?;
    let provider = rbuilder.provider().await?;
    setup_flashblock_number_contract(&driver, &provider, false).await?;
    setup_flashtestation_contracts(&driver, &provider, true, true).await?;
    // Verify flashblock number is not incremented and builder address is not authorized
    let contract = FlashblocksNumber::new(FLASHBLOCKS_NUMBER_ADDRESS, provider.clone());
    let current_number = contract.getFlashblockNumber().call().await?;
    assert!(
        current_number.is_zero(), // contract deployments incremented the number but we built at least 1 full block
        "Flashblock number should not be incremented"
    );
    let is_authorized = contract.isBuilder(builder_signer().address).call().await?;
    assert!(!is_authorized, "builder should not be authorized");

    // add tee signer address to authorized builders
    let add_builder_tx = driver
        .create_transaction()
        .add_authorized_builder(TEE_DEBUG_ADDRESS)
        .with_to(FLASHBLOCKS_NUMBER_ADDRESS)
        .with_bundle(BundleOpts::default().with_flashblock_number_min(4))
        .send()
        .await?;
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    provider
        .get_transaction_receipt(*add_builder_tx.tx_hash())
        .await?
        .expect("add builder tx not mined");
    let num_txs = block.transactions.len();
    let txs = block.transactions.into_transactions_vec();
    // 1 deposit tx, 5 regular builder tx, 1 add builder tx, 1 block proof tx
    assert_eq!(num_txs, 8, "Expected 8 transactions in block");
    // Check no transactions to the flashblocks number contract as tee signer is not authorized
    for i in 1..6 {
        assert_eq!(
            txs[i].to(),
            Some(Address::ZERO),
            "builder tx should send to flashblocks number contract at index {}",
            i
        );
    }
    // add builder tx
    assert_eq!(
        txs[6].tx_hash(),
        *add_builder_tx.tx_hash(),
        "add builder tx should be in correct position in block"
    );
    assert_eq!(
        txs[7].to(),
        Some(BLOCK_BUILDER_POLICY_ADDRESS),
        "builder tx should send verify block builder proof"
    );

    let tx = driver
        .create_transaction()
        .random_valid_transfer()
        .with_bundle(BundleOpts::default().with_flashblock_number_min(4))
        .send()
        .await?;
    let block = driver.build_new_block_with_current_timestamp(None).await?;
    let txs = block.transactions.into_transactions_vec();
    // 1 deposit tx, 1 regular builder tx, 4 flashblocks builder tx, 1 user tx, 1 block proof tx
    assert_eq!(txs.len(), 8, "Expected 8 transactions in block");
    // flashblocks number contract
    for i in 2..6 {
        assert_eq!(
            txs[i].to(),
            Some(FLASHBLOCKS_NUMBER_ADDRESS),
            "builder tx should send to flashblocks number contract at index {}",
            i
        );
    }
    // user tx
    assert_eq!(
        txs[6].tx_hash(),
        *tx.tx_hash(),
        "user tx should be in correct position in block"
    );
    // check that the tee signer did not send any transactions
    let balance = provider.get_balance(TEE_DEBUG_ADDRESS).await?;
    assert!(balance.is_zero());
    let nonce = provider.get_transaction_count(TEE_DEBUG_ADDRESS).await?;
    assert_eq!(nonce, 0);
    // Verify flashblock number incremented correctly
    let contract = FlashblocksNumber::new(FLASHBLOCKS_NUMBER_ADDRESS, provider.clone());
    let current_number = contract.getFlashblockNumber().call().await?;
    assert_eq!(
        current_number,
        U256::from(4),
        "Flashblock number not incremented correctly"
    );
    Ok(())
}

async fn setup_flashtestation_contracts(
    driver: &ChainDriver,
    provider: &RootProvider<Optimism>,
    add_quote: bool,
    authorize_workload: bool,
) -> eyre::Result<()> {
    // deploy the mock contract and register a mock quote
    let mock_dcap_deploy_tx = driver
        .create_transaction()
        .deploy_mock_dcap_contract()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // deploy the flashtestations registry contract
    let flashtestations_registry_tx = driver
        .create_transaction()
        .deploy_flashtestation_registry_contract()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // init the flashtestation registry contract
    let init_registry = driver
        .create_transaction()
        .init_flashtestation_registry_contract(MOCK_DCAP_ADDRESS)
        .with_to(FLASHTESTATION_REGISTRY_ADDRESS)
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // deploy the block builder policy contract
    let block_builder_policy_tx = driver
        .create_transaction()
        .deploy_builder_policy_contract()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // init the builder block policy contract
    let init_builder_policy = driver
        .create_transaction()
        .init_builder_policy_contract(FLASHTESTATION_REGISTRY_ADDRESS)
        .with_to(BLOCK_BUILDER_POLICY_ADDRESS)
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // include the deployment and initialization in a block
    driver.build_new_block_with_current_timestamp(None).await?;

    if add_quote {
        // Add test quote
        let mock_quote_tx = driver
            .create_transaction()
            .add_mock_quote()
            .with_to(MOCK_DCAP_ADDRESS)
            .with_bundle(BundleOpts::default())
            .send()
            .await?;
        driver.build_new_block_with_current_timestamp(None).await?;
        provider
            .get_transaction_receipt(*mock_quote_tx.tx_hash())
            .await?
            .expect("add mock quote not mined");
        // verify registered
        let registry_contract =
            FlashtestationRegistry::new(FLASHTESTATION_REGISTRY_ADDRESS, provider.clone());
        let registration = registry_contract
            .getRegistration(TEE_DEBUG_ADDRESS)
            .call()
            .await?;
        assert!(registration._0, "The tee key is not registered");
    }

    if authorize_workload {
        // add the workload id to the block builder policy
        let add_workload = driver
            .create_transaction()
            .add_workload_to_policy()
            .with_to(BLOCK_BUILDER_POLICY_ADDRESS)
            .with_bundle(BundleOpts::default())
            .send()
            .await?;
        driver.build_new_block_with_current_timestamp(None).await?;
        provider
            .get_transaction_receipt(*add_workload.tx_hash())
            .await?
            .expect("add workload to builder policy tx not mined");
        // verify workload id added
        let policy_contract =
            BlockBuilderPolicy::new(BLOCK_BUILDER_POLICY_ADDRESS, provider.clone());
        let is_allowed = policy_contract
            .isAllowedPolicy(TEE_DEBUG_ADDRESS)
            .call()
            .await?;
        assert!(is_allowed.allowed, "The policy is not allowed")
    }

    // Verify mock dcap contract deployment
    let receipt = provider
        .get_transaction_receipt(*mock_dcap_deploy_tx.tx_hash())
        .await?
        .expect("mock dcap contract deployment not mined");
    let mock_dcap_address = receipt
        .inner
        .contract_address
        .expect("contract receipt does not contain flashblock number contract address");
    assert_eq!(
        mock_dcap_address, MOCK_DCAP_ADDRESS,
        "mock dcap contract address mismatch"
    );
    // verify flashtestations registry contract deployment
    let receipt = provider
        .get_transaction_receipt(*flashtestations_registry_tx.tx_hash())
        .await?;
    let flashtestations_registry_address = receipt
        .expect("flashtestations registry contract deployment not mined")
        .inner
        .contract_address
        .expect("contract receipt does not contain flashtestations registry contract address");
    assert_eq!(
        flashtestations_registry_address, FLASHTESTATION_REGISTRY_ADDRESS,
        "flashtestations registry contract address mismatch"
    );
    // verify flashtestations registry contract initialization
    provider
        .get_transaction_receipt(*init_registry.tx_hash())
        .await?
        .expect("init registry tx not mined");

    // verify block builder policy contract deployment
    let receipt = provider
        .get_transaction_receipt(*block_builder_policy_tx.tx_hash())
        .await?;
    let block_builder_policy_address = receipt
        .expect("block builder policy contract deployment not mined")
        .inner
        .contract_address
        .expect("contract receipt does not contain block builder policy contract address");
    assert_eq!(
        block_builder_policy_address, BLOCK_BUILDER_POLICY_ADDRESS,
        "block builder policy contract address mismatch"
    );
    // verify block builder policy contract initialization
    provider
        .get_transaction_receipt(*init_builder_policy.tx_hash())
        .await?
        .expect("init builder policy tx not mined");

    Ok(())
}

async fn setup_flashblock_number_contract(
    driver: &ChainDriver,
    provider: &RootProvider<Optimism>,
    authorize_builder: bool,
) -> eyre::Result<()> {
    // Deploy flashblocks number contract
    let deploy_tx = driver
        .create_transaction()
        .deploy_flashblock_number_contract()
        .with_bundle(BundleOpts::default())
        .send()
        .await?;

    // Initialize contract
    let init_tx = driver
        .create_transaction()
        .init_flashblock_number_contract(authorize_builder)
        .with_to(FLASHBLOCKS_NUMBER_ADDRESS)
        .with_bundle(BundleOpts::default())
        .send()
        .await?;
    driver.build_new_block_with_current_timestamp(None).await?;

    // Verify contract deployment
    let receipt = provider
        .get_transaction_receipt(*deploy_tx.tx_hash())
        .await?
        .expect("flashblock number contract deployment not mined");
    let contract_address = receipt
        .inner
        .contract_address
        .expect("contract receipt does not contain flashblock number contract address");
    assert_eq!(
        contract_address, FLASHBLOCKS_NUMBER_ADDRESS,
        "Flashblocks number contract address mismatch"
    );

    // Verify initialization
    provider
        .get_transaction_receipt(*init_tx.tx_hash())
        .await?
        .expect("init tx not mined");

    Ok(())
}
