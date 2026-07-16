use std::ops::Range;

use alloy_chains::Chain;
use alloy_consensus::Header;
use alloy_primitives::{B256, keccak256};
use alloy_rlp::Encodable;
use kona_genesis::RollupConfig;
use kona_interop::{ChainDependency, DependencySet};
use kona_preimage::PreimageKey;
use kona_sp1_client_utils::witness::preimage_store::PreimageStore;

const OUTPUT_ROOT_WORD_BYTES: usize = 32;
const OUTPUT_ROOT_V0_BYTES: usize = 4 * OUTPUT_ROOT_WORD_BYTES;
const OUTPUT_ROOT_V0_BLOCK_HASH_RANGE: Range<usize> =
    3 * OUTPUT_ROOT_WORD_BYTES..OUTPUT_ROOT_V0_BYTES;

pub(crate) fn b256(fill: u8) -> B256 {
    B256::from([fill; 32])
}

pub(crate) fn save_header(oracle: &mut PreimageStore, header: &Header) -> B256 {
    let hash = header.hash_slow();
    let mut rlp = Vec::new();
    header.encode(&mut rlp);
    oracle.save_preimage(PreimageKey::new_keccak256(*hash), rlp).unwrap();
    hash
}

pub(crate) fn save_output_root(oracle: &mut PreimageStore, block_hash: B256) -> B256 {
    let mut output_preimage = [0u8; OUTPUT_ROOT_V0_BYTES];
    output_preimage[OUTPUT_ROOT_V0_BLOCK_HASH_RANGE].copy_from_slice(block_hash.as_slice());
    let output_root = B256::from(keccak256(output_preimage));
    oracle
        .save_preimage(PreimageKey::new_keccak256(*output_root), output_preimage.to_vec())
        .unwrap();
    output_root
}

#[allow(clippy::zero_sized_map_values)]
pub(crate) fn dependency_set(
    chain_ids: &[u64],
    override_message_expiry_window: Option<u64>,
) -> DependencySet {
    let dependencies = chain_ids.iter().map(|chain_id| (*chain_id, ChainDependency {})).collect();
    DependencySet { dependencies, override_message_expiry_window }
}

pub(crate) fn rollup_config(chain_id: u64, l1_chain_id: u64) -> RollupConfig {
    RollupConfig {
        block_time: 1,
        l1_chain_id,
        l2_chain_id: Chain::from(chain_id),
        ..Default::default()
    }
}
