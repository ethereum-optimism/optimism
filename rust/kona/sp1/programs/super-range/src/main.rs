//! SP1 guest scaffold for unified super-root range and consolidation proofs.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use std::sync::Arc;

use alloy_primitives::Keccak256;
use anyhow::anyhow;
use kona_sp1_client_utils::{
    BlobStore,
    log_commitment::{LogCommitmentInputs, VerifiedLogCommitments},
    super_root::{SuperInteropInputs, SuperInteropOutputs},
    witness::{DefaultWitnessData, WitnessData, preimage_store::WitnessOracle},
};
use kona_sp1_ethereum_client_utils::{
    super_consolidation::build_consolidation_outputs, super_range::build_range_outputs,
};
use rkyv::rancor::Error as RkyvError;

#[cfg(feature = "test-config-fallback")]
mod test_config;

/// Entrypoint to the unified super-root range program.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);
    #[cfg(feature = "test-config-fallback")]
    println!("{}", test_config::MARKER);

    let inputs = sp1_zkvm::io::read::<SuperInteropInputs>();
    foldhash::install_global_seed(hash_seed(&inputs));
    let outputs = kona_proof::block_on(run(inputs)).expect("super interop failed");
    sp1_zkvm::io::commit(&outputs);
}

/// The hash that keys every hash map of this guest (see `programs/foldhash-seeded`).
fn hash_seed(inputs: &SuperInteropInputs) -> u64 {
    let hash = match inputs {
        // The game's L1 head. Every batch and deposit the range derives is at or below it, and
        // each L2 block's L1 origin precedes the L1 block that includes its batch.
        SuperInteropInputs::Range(inputs) => inputs.l1_head,
        // The super root that the consolidation proves at `span.end`, which no block in the span
        // can read, and every optimistic block hash in the span. The root commits to every
        // chain's block at `span.end`; the block hashes commit to the transactions of the blocks
        // that consolidation replaces, which the root omits.
        SuperInteropInputs::Consolidation(inputs) => {
            let last = inputs
                .transitions
                .last()
                .expect("consolidation input span contains no transitions");
            let mut hasher = Keccak256::new();
            hasher.update(last.claimed_super_root_proof.super_root.hash());
            for block in inputs.transitions.iter().flat_map(|t| &t.optimistic_blocks) {
                hasher.update(block.block_hash);
            }
            hasher.finalize()
        }
    };
    // Both are keccak hashes, so their first 8 bytes are uniform.
    u64::from_le_bytes(hash[..8].try_into().expect("8 bytes"))
}

async fn run(inputs: SuperInteropInputs) -> anyhow::Result<SuperInteropOutputs> {
    let (oracle, beacon) = read_witness().await?;
    #[cfg(feature = "test-config-fallback")]
    let configs = Some(test_config::load(&inputs, oracle.as_ref()).await?);
    #[cfg(not(feature = "test-config-fallback"))]
    let configs: Option<kona_sp1_ethereum_client_utils::chain_config::ChainConfigs> = None;

    match inputs {
        SuperInteropInputs::Range(inputs) => Ok(SuperInteropOutputs::Range(
            build_range_outputs(inputs, oracle, beacon, configs.as_ref()).await?,
        )),
        SuperInteropInputs::Consolidation(inputs) => {
            let log_commitments = VerifiedLogCommitments::new(
                LogCommitmentInputs::decode(&sp1_zkvm::io::read_vec()),
                &kona_sp1_log_commitment_vkey::LOG_COMMITMENT_VKEY,
                sp1_lib::verify::verify_sp1_proof,
            );
            Ok(SuperInteropOutputs::Consolidation(
                build_consolidation_outputs(
                    inputs,
                    oracle,
                    configs.as_ref(),
                    Arc::new(log_commitments),
                )
                .await?,
            ))
        }
    }
}

async fn read_witness() -> anyhow::Result<(Arc<WitnessOracle>, BlobStore)> {
    let witness_rkyv_bytes: Vec<u8> = sp1_zkvm::io::read_vec();
    let witness_data = rkyv::from_bytes::<DefaultWitnessData, RkyvError>(&witness_rkyv_bytes)
        .map_err(|err| anyhow!("failed to deserialize super-range witness data: {err}"))?;

    witness_data
        .get_oracle_and_blob_provider()
        .await
        .map_err(|err| anyhow!("failed to load oracle and blob provider: {err}"))
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{B256, U256};
    use kona_sp1_client_utils::super_root::{
        SuperConsolidationInputs, SuperConsolidationTransitionInput, SuperInteropInputs,
        SuperOptimisticBlock, SuperOutputRoot, SuperRootProof, TimestampSpan,
    };

    use super::hash_seed;

    fn transition(timestamp: u64, block_fills: [u8; 2]) -> SuperConsolidationTransitionInput {
        let block = |chain_id: u64, fill: u8| SuperOptimisticBlock {
            chain_id: U256::from(chain_id),
            block_hash: B256::from([fill; 32]),
            output_root: B256::from([!fill; 32]),
        };
        SuperConsolidationTransitionInput {
            optimistic_blocks: vec![block(10, block_fills[0]), block(20, block_fills[1])],
            claimed_super_root_proof: SuperRootProof::new(
                timestamp,
                vec![
                    SuperOutputRoot { chain_id: 10, output_root: B256::from([0x31; 32]) },
                    SuperOutputRoot { chain_id: 20, output_root: B256::from([0x32; 32]) },
                ],
            ),
        }
    }

    /// Pins that the consolidation seed changes with every optimistic block hash in the span,
    /// including blocks before `span.end`.
    #[test]
    fn consolidation_seed_commits_to_every_optimistic_block() {
        let inputs = SuperConsolidationInputs {
            span: TimestampSpan::new(100, 101).expect("valid span"),
            previous_super_root: B256::from([0xaa; 32]),
            transitions: vec![transition(100, [0x11, 0x21]), transition(101, [0x41, 0x51])],
        };
        let seed = hash_seed(&SuperInteropInputs::Consolidation(inputs.clone()));
        for (t, b) in [(0, 0), (0, 1), (1, 0), (1, 1)] {
            let mut changed = inputs.clone();
            changed.transitions[t].optimistic_blocks[b].block_hash = B256::from([0xee; 32]);
            assert_ne!(
                hash_seed(&SuperInteropInputs::Consolidation(changed)),
                seed,
                "optimistic block {b} of transition {t}"
            );
        }
    }
}
