pub mod flashtestation_registry {
    use crate::tests::framework::contracts::block_builder_policy::BlockBuilderPolicy::TD10ReportBody;
    use alloy_sol_types::sol;

    sol!(
        // https://github.com/flashbots/flashtestations/tree/7cc7f68492fe672a823dd2dead649793aac1f216
        #[sol(rpc, abi)]
        FlashtestationRegistry,
        "src/tests/framework/artifacts/contracts/FlashtestationRegistry.json",
    );
}

pub mod block_builder_policy {
    use crate::tests::framework::contracts::block_builder_policy::BlockBuilderPolicy::TD10ReportBody;
    use alloy_sol_types::sol;

    sol!(
        // https://github.com/flashbots/flashtestations/tree/7cc7f68492fe672a823dd2dead649793aac1f216
        #[sol(rpc, abi)]
        BlockBuilderPolicy,
        "src/tests/framework/artifacts/contracts/BlockBuilderPolicy.json",
    );
}

pub mod flashblocks_number_contract {
    use alloy_sol_types::sol;
    sol!(
        // https://github.com/Uniswap/flashblocks_number_contract/tree/c21ca0aedc3ff4d1eecf20cd55abeb984080bc78
        #[sol(rpc, abi)]
        FlashblocksNumber,
        "src/tests/framework/artifacts/contracts/FlashblocksNumberContract.json",
    );
}

pub mod mock_dcap_attestation {
    use alloy_sol_types::sol;
    sol!(
        // https://github.com/flashbots/flashtestations/tree/7cc7f68492fe672a823dd2dead649793aac1f216
        #[sol(rpc, abi)]
        MockAutomataDcapAttestationFee,
        "src/tests/framework/artifacts/contracts/MockAutomataDcapAttestationFee.json",
    );
}
