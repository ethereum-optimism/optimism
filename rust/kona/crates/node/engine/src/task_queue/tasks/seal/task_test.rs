//! Tests for the pairing a [`SealTask`] hands to the [`InsertTask`] it runs.
//!
//! [`SealTask`]: super::SealTask
//! [`InsertTask`]: crate::InsertTask

use crate::{
    LocalSafeOrigin, SealTask,
    test_utils::{MockEngineClient, TestAttributesBuilder, test_engine_client_builder},
};
use alloy_primitives::Bytes;
use alloy_rpc_types_engine::PayloadId;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use std::sync::Arc;

fn l1(number: u64) -> BlockInfo {
    BlockInfo { number, ..Default::default() }
}

fn seal_task(attributes: OpAttributesWithParent, is_derived: bool) -> SealTask<MockEngineClient> {
    let cfg = Arc::new(RollupConfig::default());
    SealTask {
        engine: Arc::new(test_engine_client_builder().with_config(cfg.clone()).build()),
        cfg,
        payload_id: PayloadId::default(),
        attributes,
        is_attributes_derived: is_derived,
        result_tx: None,
    }
}

/// A derived block is a local-safe write, and the L1 key it is paired with comes from the
/// attributes the task already holds — not from a lookup after the fact.
#[test]
fn a_derived_block_carries_the_attributes_origin() {
    let attributes = TestAttributesBuilder::new().with_derived_from(l1(7)).build();

    assert_eq!(
        seal_task(attributes, true).local_safe_origin(),
        Some(LocalSafeOrigin::DerivedFrom(l1(7)))
    );
}

/// A sequencer-built block moves no local-safe head at all, which is the outer [`None`] rather
/// than an unpaired origin.
#[test]
fn a_sequenced_block_is_not_a_local_safe_write() {
    let attributes = TestAttributesBuilder::new().with_derived_from(l1(7)).build();

    assert_eq!(seal_task(attributes, false).local_safe_origin(), None);
}

/// The Holocene deposits-only retry re-seals a genuinely different block, but not one from a
/// different L1 origin: `as_deposits_only` copies `derived_from`, so the pairing survives the
/// fallback. This is the one path where a wrong L1 key would otherwise be possible.
#[test]
fn the_deposits_only_fallback_keeps_the_origin() {
    let attributes = TestAttributesBuilder::new()
        .with_derived_from(l1(7))
        .with_transactions(vec![
            Bytes::from_static(&[0x7e, 0x01]), // deposit
            Bytes::from_static(&[0x02, 0x01]), // eip-1559, dropped by the fallback
        ])
        .build();
    let deposits_only = attributes.as_deposits_only();

    assert!(deposits_only.is_deposits_only(), "test setup: the fallback dropped the user tx");
    assert_eq!(
        seal_task(deposits_only, true).local_safe_origin(),
        seal_task(attributes, true).local_safe_origin(),
        "the deposits-only retry must report the same L1 origin as the block it replaces"
    );
}
