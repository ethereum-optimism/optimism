//! Tests for the checks that `build.rs` runs on a custom-config build.

#![allow(missing_docs, clippy::zero_sized_map_values)]

#[path = "../build_checks.rs"]
mod build_checks;

use std::collections::BTreeSet;

use build_checks::{CustomConfigError, check_dependency_sets, check_l1s};
use kona_genesis::{
    Chain, ChainConfig, ChainDependency, ChainList, DependencySet, InteropConfig, Superchain,
    SuperchainConfig, SuperchainL1Info, Superchains,
};

const A: u64 = 901;
const B: u64 = 902;
const SEPOLIA: u64 = 11_155_111;

fn depset(chain_ids: &[u64]) -> DependencySet {
    DependencySet {
        dependencies: chain_ids.iter().map(|id| (*id, ChainDependency {})).collect(),
        override_message_expiry_window: None,
    }
}

fn chain_list(chain_ids: &[u64]) -> ChainList {
    ChainList {
        chains: chain_ids
            .iter()
            .map(|id| Chain {
                identifier: format!("custom/{id}"),
                chain_id: *id,
                ..Default::default()
            })
            .collect(),
    }
}

/// One chain group named `custom` on L1 `l1_chain_id`, holding each chain with its `interop`
/// dependencies.
fn superchains(l1_chain_id: u64, chains: &[(u64, Option<&[u64]>)]) -> Superchains {
    let chains = chains
        .iter()
        .map(|(chain_id, interop)| ChainConfig {
            chain_id: *chain_id,
            interop: interop.map(|dependencies| InteropConfig {
                dependencies: dependencies.iter().map(|id| (*id, ChainDependency {})).collect(),
            }),
            ..Default::default()
        })
        .collect();
    Superchains {
        superchains: vec![Superchain {
            name: "custom".to_string(),
            config: SuperchainConfig {
                l1: SuperchainL1Info { chain_id: l1_chain_id, ..Default::default() },
                ..Default::default()
            },
            chains,
        }],
    }
}

#[test]
fn sets_covering_every_chain_pass() {
    let chains = chain_list(&[A, B]);
    let configs = superchains(SEPOLIA, &[(A, None), (B, None)]);
    assert_eq!(check_dependency_sets(&[depset(&[A, B])], &chains, &configs), Ok(()));
    assert_eq!(check_dependency_sets(&[depset(&[A]), depset(&[B])], &chains, &configs), Ok(()));
}

#[test]
fn a_chain_in_no_set_fails() {
    let chains = chain_list(&[A, B]);
    let configs = superchains(SEPOLIA, &[(A, None), (B, None)]);
    assert_eq!(
        check_dependency_sets(&[depset(&[A])], &chains, &configs),
        Err(CustomConfigError::NoDependencySet { identifier: format!("custom/{B}"), chain_id: B })
    );
    assert_eq!(
        check_dependency_sets(&[], &chains, &configs),
        Err(CustomConfigError::NoDependencySet { identifier: format!("custom/{A}"), chain_id: A })
    );
}

#[test]
fn a_set_listing_an_unknown_chain_fails() {
    let chains = chain_list(&[A]);
    let configs = superchains(SEPOLIA, &[(A, None)]);
    assert_eq!(
        check_dependency_sets(&[depset(&[A, B])], &chains, &configs),
        Err(CustomConfigError::UnknownChain(B))
    );
}

#[test]
fn an_interop_cluster_must_be_one_of_the_sets() {
    let chains = chain_list(&[A, B]);
    let configs = superchains(SEPOLIA, &[(A, Some(&[A, B])), (B, Some(&[A, B]))]);
    assert_eq!(check_dependency_sets(&[depset(&[A, B])], &chains, &configs), Ok(()));
    assert_eq!(
        check_dependency_sets(&[depset(&[A]), depset(&[B])], &chains, &configs),
        Err(CustomConfigError::ClusterWithoutSet(vec![A, B]))
    );
}

#[test]
fn a_custom_chain_group_needs_an_embedded_l1_config() {
    let custom: BTreeSet<u64> = [A].into();
    assert_eq!(check_l1s(&superchains(SEPOLIA, &[(A, None)]), &custom), Ok(()));
    assert_eq!(
        check_l1s(&superchains(900, &[(A, None)]), &custom),
        Err(CustomConfigError::UnsupportedL1 { group: "custom".to_string(), l1_chain_id: 900 })
    );
    // Only groups holding a custom chain are checked.
    assert_eq!(check_l1s(&superchains(900, &[(B, None)]), &custom), Ok(()));
}
