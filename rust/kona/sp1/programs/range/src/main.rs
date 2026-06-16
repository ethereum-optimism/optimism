//! A program to verify an Optimism L2 block STF with Ethereum DA in the zkVM.
//!
//! This binary contains the client program for executing the Optimism rollup state transition
//! across a range of blocks, which can be used to generate an on chain validity proof. Depending on
//! the compilation pipeline, it will compile to be run either in native mode or in zkVM mode. In
//! native mode, the data for verifying the batch validity is fetched from RPC, while in zkVM mode,
//! the data is supplied by the host binary to the verifiable program.

#![no_main]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::witness::{DefaultWitnessData, WitnessData};
use kona_sp1_ethereum_client_utils::executor::ETHDAWitnessExecutor;
use rkyv::rancor::Error;
use std::sync::Arc;

/// Entrypoint to the range program.
fn main() {
    #[cfg(feature = "tracing-subscriber")]
    setup_tracing();

    kona_proof::block_on(async move {
        let witness_rkyv_bytes: Vec<u8> = sp1_zkvm::io::read_vec();
        let witness_data = rkyv::from_bytes::<DefaultWitnessData, Error>(&witness_rkyv_bytes)
            .expect("Failed to deserialize witness data.");

        let (oracle, beacon) = witness_data
            .get_oracle_and_blob_provider()
            .await
            .expect("Failed to load oracle and blob provider");

        run_range_program(ETHDAWitnessExecutor::new(), oracle, beacon).await;
    });
}

use kona_proof::{l1::OracleL1ChainProvider, l2::OracleL2ChainProvider};
use kona_sp1_client_utils::{
    BlobStore,
    boot::BootInfoStruct,
    witness::{
        executor::{WitnessExecutor, get_inputs_for_pipeline},
        preimage_store::PreimageStore,
    },
};

/// Sets up tracing for the range program
#[cfg(feature = "tracing-subscriber")]
pub fn setup_tracing() {
    use anyhow::anyhow;
    use tracing::Level;

    let subscriber = tracing_subscriber::fmt().with_max_level(Level::INFO).finish();
    tracing::subscriber::set_global_default(subscriber).map_err(|e| anyhow!(e)).unwrap();
}

/// Executes the range program with the given [`WitnessExecutor`], [`PreimageStore`], and
/// [`BlobStore`].
pub async fn run_range_program<E>(executor: E, oracle: Arc<PreimageStore>, beacon: BlobStore)
where
    E: WitnessExecutor<
            O = PreimageStore,
            B = BlobStore,
            L1 = OracleL1ChainProvider<PreimageStore>,
            L2 = OracleL2ChainProvider<PreimageStore>,
        > + Send
        + Sync,
{
    ////////////////////////////////////////////////////////////////
    //                          PROLOGUE                          //
    ////////////////////////////////////////////////////////////////
    let (boot_info, input) = get_inputs_for_pipeline(oracle.clone()).await.unwrap();
    let boot_info = match input {
        Some((cursor, l1_provider, l2_provider)) => {
            let rollup_config = Arc::new(boot_info.rollup_config.clone());
            let l1_config = Arc::new(boot_info.l1_config.clone());

            let pipeline = executor
                .create_pipeline(
                    rollup_config,
                    l1_config,
                    cursor.clone(),
                    oracle,
                    beacon,
                    l1_provider,
                    l2_provider.clone(),
                )
                .await
                .unwrap();

            executor.run(boot_info, pipeline, cursor, l2_provider).await.unwrap()
        }
        None => boot_info,
    };

    sp1_zkvm::io::commit_slice(&BootInfoStruct::from(boot_info).public_values());
}
