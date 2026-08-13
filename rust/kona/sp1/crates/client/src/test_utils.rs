//! Test support for SP1 client utilities.

use alloy_primitives::{B256, U256, address};

use crate::super_root::{
    SuperAggregationInputs, SuperConsolidationOutputs, SuperConsolidationTransition,
    SuperOptimisticBlock, SuperOutputRoot, SuperRangeOutputs, SuperRangeTransition, SuperRootProof,
    TimestampSpan, hash_super_root_proof,
};

fn output(chain_id: u64, fill: u8) -> SuperOutputRoot {
    SuperOutputRoot { chain_id, output_root: B256::from([fill; 32]) }
}

fn optimistic(chain_id: u64, block_fill: u8, output_fill: u8) -> SuperOptimisticBlock {
    SuperOptimisticBlock {
        chain_id: U256::from(chain_id),
        block_hash: B256::from([block_fill; 32]),
        output_root: B256::from([output_fill; 32]),
    }
}

fn consolidation_transition(
    timestamp: u64,
    optimistic_blocks: Vec<SuperOptimisticBlock>,
    super_root_fill: u8,
) -> SuperConsolidationTransition {
    SuperConsolidationTransition {
        timestamp,
        optimistic_blocks,
        super_root: B256::from([super_root_fill; 32]),
    }
}

/// Returns the canonical valid two-chain fixture for super-aggregation tests.
pub fn valid_aggregation_inputs() -> SuperAggregationInputs {
    let starting_super_root_proof =
        SuperRootProof::new(99, vec![output(10, 0x01), output(20, 0x02)]);
    let starting_root_hash =
        hash_super_root_proof(&starting_super_root_proof).expect("starting root hashes");
    let final_super_root = B256::from([0x55; 32]);
    let timestamp_100 = vec![optimistic(10, 0x11, 0x12), optimistic(20, 0x21, 0x22)];
    let timestamp_101 = vec![optimistic(10, 0x31, 0x32), optimistic(20, 0x41, 0x42)];

    SuperAggregationInputs {
        l1_head: B256::from([0x99; 32]),
        starting_root_hash,
        starting_super_root_proof,
        root_claim: final_super_root,
        l2_sequence_number: 101,
        prover: address!("0x1234567890123456789012345678901234567890"),
        range_outputs: vec![
            SuperRangeOutputs {
                span: TimestampSpan::new(100, 100).expect("valid span"),
                l1_head: B256::from([0x99; 32]),
                previous_super_roots: vec![starting_root_hash],
                transitions: vec![
                    SuperRangeTransition { timestamp: 100, optimistic_block: timestamp_100[0] },
                    SuperRangeTransition { timestamp: 100, optimistic_block: timestamp_100[1] },
                ],
            },
            SuperRangeOutputs {
                span: TimestampSpan::new(101, 101).expect("valid span"),
                l1_head: B256::from([0x99; 32]),
                previous_super_roots: vec![B256::from([0x44; 32])],
                transitions: vec![
                    SuperRangeTransition { timestamp: 101, optimistic_block: timestamp_101[0] },
                    SuperRangeTransition { timestamp: 101, optimistic_block: timestamp_101[1] },
                ],
            },
        ],
        consolidation_outputs: vec![
            SuperConsolidationOutputs {
                span: TimestampSpan::new(100, 100).expect("valid span"),
                previous_super_root: starting_root_hash,
                transitions: vec![consolidation_transition(100, timestamp_100, 0x44)],
            },
            SuperConsolidationOutputs {
                span: TimestampSpan::new(101, 101).expect("valid span"),
                previous_super_root: B256::from([0x44; 32]),
                transitions: vec![consolidation_transition(101, timestamp_101, 0x55)],
            },
        ],
    }
}

/// Returns the canonical valid single-chain fixture for super-aggregation tests.
pub fn valid_single_chain_aggregation_inputs() -> SuperAggregationInputs {
    let starting_super_root_proof = SuperRootProof::new(99, vec![output(10, 0x01)]);
    let starting_root_hash =
        hash_super_root_proof(&starting_super_root_proof).expect("starting root hashes");
    let final_super_root = B256::from([0x44; 32]);
    let timestamp_100 = vec![optimistic(10, 0x11, 0x12)];

    SuperAggregationInputs {
        l1_head: B256::from([0x99; 32]),
        starting_root_hash,
        starting_super_root_proof,
        root_claim: final_super_root,
        l2_sequence_number: 100,
        prover: address!("0x1234567890123456789012345678901234567890"),
        range_outputs: vec![SuperRangeOutputs {
            span: TimestampSpan::new(100, 100).expect("valid span"),
            l1_head: B256::from([0x99; 32]),
            previous_super_roots: vec![starting_root_hash],
            transitions: vec![SuperRangeTransition {
                timestamp: 100,
                optimistic_block: timestamp_100[0],
            }],
        }],
        consolidation_outputs: vec![SuperConsolidationOutputs {
            span: TimestampSpan::new(100, 100).expect("valid span"),
            previous_super_root: starting_root_hash,
            transitions: vec![consolidation_transition(100, timestamp_100, 0x44)],
        }],
    }
}
