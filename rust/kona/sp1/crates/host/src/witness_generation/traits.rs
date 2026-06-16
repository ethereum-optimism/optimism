//! Traits for witness generation in the SP1 host environment.

use std::sync::{Arc, Mutex};

use anyhow::Result;
use async_trait::async_trait;
use kona_preimage::{HintWriter, NativeChannel, OracleReader};
use kona_proof::{
    CachingOracle,
    l1::{OracleBlobProvider, OracleL1ChainProvider},
    l2::OracleL2ChainProvider,
};
use kona_sp1_client_utils::witness::{
    BlobData, WitnessData,
    executor::{WitnessExecutor, get_inputs_for_pipeline},
    preimage_store::PreimageStore,
};
use sp1_sdk::SP1Stdin;

use crate::witness_generation::{OnlineBlobStore, PreimageWitnessCollector};

/// Default type alias for the Oracle base used in witness generation.
pub type DefaultOracleBase = CachingOracle<OracleReader<NativeChannel>, HintWriter<NativeChannel>>;

/// Trait representing a witness generator that can run the witness generation process
#[async_trait]
pub trait WitnessGenerator {
    /// A [`WitnessData`] produced by the [`WitnessGenerator`].
    type WitnessData: WitnessData;
    /// A [`WitnessExecutor`] used for witness generation.
    type WitnessExecutor: WitnessExecutor<
            O = PreimageWitnessCollector<DefaultOracleBase>,
            B = OnlineBlobStore<OracleBlobProvider<DefaultOracleBase>>,
            L1 = OracleL1ChainProvider<PreimageWitnessCollector<DefaultOracleBase>>,
            L2 = OracleL2ChainProvider<PreimageWitnessCollector<DefaultOracleBase>>,
        > + Sync
        + Send;

    /// Gets the executor used for witness generation.
    fn get_executor(&self) -> &Self::WitnessExecutor;

    /// Runs derivation to generate [`WitnessData`] using the provided preimage and hint channels.
    async fn run(
        &self,
        preimage_chan: NativeChannel,
        hint_chan: NativeChannel,
    ) -> Result<Self::WitnessData> {
        let preimage_witness_store = Arc::new(Mutex::new(PreimageStore::default()));
        let blob_data = Arc::new(Mutex::new(BlobData::default()));

        let preimage_oracle = Arc::new(CachingOracle::new(
            2048,
            OracleReader::new(preimage_chan),
            HintWriter::new(hint_chan),
        ));
        let blob_provider = OracleBlobProvider::new(preimage_oracle.clone());

        let oracle = Arc::new(PreimageWitnessCollector {
            preimage_oracle: preimage_oracle.clone(),
            preimage_witness_store: preimage_witness_store.clone(),
        });
        let beacon = OnlineBlobStore { provider: blob_provider.clone(), store: blob_data.clone() };

        let (boot_info, input) = get_inputs_for_pipeline(oracle.clone()).await.unwrap();
        if let Some((cursor, l1_provider, l2_provider)) = input {
            let rollup_config = Arc::new(boot_info.rollup_config.clone());
            let l1_config = Arc::new(boot_info.l1_config.clone());
            let pipeline = self
                .get_executor()
                .create_pipeline(
                    rollup_config,
                    l1_config,
                    cursor.clone(),
                    oracle.clone(),
                    beacon,
                    l1_provider.clone(),
                    l2_provider.clone(),
                )
                .await
                .unwrap();
            self.get_executor().run(boot_info, pipeline, cursor, l2_provider).await.unwrap();
        }

        let witness = Self::WitnessData::from_parts(
            preimage_witness_store.lock().unwrap().clone(),
            blob_data.lock().unwrap().clone(),
        );

        Ok(witness)
    }

    /// Converts the given witness data into SP1 stdin format.
    fn get_sp1_stdin(&self, witness: Self::WitnessData) -> Result<SP1Stdin>;
}
