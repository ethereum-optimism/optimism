//! This module defines the witness generator for ETHDA SP1 host.

use anyhow::Result;
use async_trait::async_trait;
use kona_proof::l1::OracleBlobProvider;
use kona_sp1_client_utils::witness::DefaultWitnessData;
use kona_sp1_ethereum_client_utils::executor::ETHDAWitnessExecutor;
use kona_sp1_host_utils::witness_generation::{
    DefaultOracleBase, WitnessGenerator, online_blob_store::OnlineBlobStore,
    preimage_witness_collector::PreimageWitnessCollector,
};
use rkyv::to_bytes;
use sp1_sdk::SP1Stdin;

type WitnessExecutor = ETHDAWitnessExecutor<
    PreimageWitnessCollector<DefaultOracleBase>,
    OnlineBlobStore<OracleBlobProvider<DefaultOracleBase>>,
>;

/// Witness generator for ETHDA SP1 host.
#[expect(missing_debug_implementations)]
pub struct ETHDAWitnessGenerator {
    /// The witness executor.
    pub executor: WitnessExecutor,
}

#[async_trait]
impl WitnessGenerator for ETHDAWitnessGenerator {
    type WitnessData = DefaultWitnessData;
    type WitnessExecutor = WitnessExecutor;

    fn get_executor(&self) -> &Self::WitnessExecutor {
        &self.executor
    }

    fn get_sp1_stdin(&self, witness: Self::WitnessData) -> Result<SP1Stdin> {
        let mut stdin = SP1Stdin::new();
        let buffer = to_bytes::<rkyv::rancor::Error>(&witness)?;
        stdin.write_slice(&buffer);
        Ok(stdin)
    }
}
