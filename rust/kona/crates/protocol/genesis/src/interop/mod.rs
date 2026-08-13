//! Interop-related genesis types shared by the registry and the interop crate.

use alloc::{
    collections::{BTreeMap, BTreeSet},
    vec::Vec,
};

mod constants;
pub use constants::MESSAGE_EXPIRY_WINDOW;

mod depset;
pub use depset::{ChainDependency, DependencySet};

mod config;
pub use config::InteropConfig;

/// Errors raised by [`aggregate_clusters`].
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum ClusterError {
    /// Chain `a` claims `b` is in its cluster but `b`'s declared cluster differs.
    #[error(
        "inconsistent interop cluster: chain {a} and chain {b} declare different dependency sets"
    )]
    Inconsistent {
        /// First chain id involved in the conflict.
        a: u64,
        /// Second chain id involved in the conflict.
        b: u64,
    },
    /// Chain `a` references chain `b` in its cluster but `b` does not declare any interop config.
    #[error("chain {a} declares dependency on chain {b}, but {b} has no [interop] section")]
    DanglingMember {
        /// The chain that declared the dependency.
        a: u64,
        /// The dependency target that has no interop config.
        b: u64,
    },
}

/// Aggregate per-cluster [`DependencySet`]s from a set of chain interop sections.
///
/// Each cluster is the set of chain ids that mutually declare each other in their
/// `[interop.dependencies]` blocks. Returns one [`DependencySet`] per cluster.
/// Errors when a cluster is internally inconsistent (a chain in the cluster declares a
/// different `interop.dependencies`) or when a chain references a peer that has no
/// `[interop]` section.
///
/// The output is deterministic: clusters are sorted by their minimum chain id.
///
/// `override_message_expiry_window` is always `None` on the returned depsets — the
/// chain TOML schema has no override field, so any override must come through a
/// custom-config `depsets.json`. If the schema grows one, [`InteropConfig`] and this
/// function need to change together.
pub fn aggregate_clusters<'a, I>(chains: I) -> Result<Vec<DependencySet>, ClusterError>
where
    I: IntoIterator<Item = (u64, &'a InteropConfig)>,
{
    let by_chain: BTreeMap<u64, &InteropConfig> = chains.into_iter().collect();

    // Validate every dependency target has its own interop section.
    for (chain_id, cfg) in &by_chain {
        for dep_id in cfg.dependencies.keys() {
            if !by_chain.contains_key(dep_id) {
                return Err(ClusterError::DanglingMember { a: *chain_id, b: *dep_id });
            }
        }
    }

    // Validate that every chain in a cluster declares the same dependency set.
    // Two chains are in the same cluster iff they declare equal `dependencies` maps
    // (and every chain in the cluster appears as a key in those maps).
    for (chain_id, cfg) in &by_chain {
        for dep_id in cfg.dependencies.keys() {
            let peer_cfg = by_chain.get(dep_id).expect("dangling check above guarantees presence");
            if peer_cfg.dependencies != cfg.dependencies {
                return Err(ClusterError::Inconsistent { a: *chain_id, b: *dep_id });
            }
        }
    }

    // Group chains by the (canonical) cluster identifier — the set of dependency keys.
    let mut visited: BTreeSet<u64> = BTreeSet::new();
    let mut clusters: Vec<DependencySet> = Vec::new();
    for (chain_id, cfg) in &by_chain {
        if visited.contains(chain_id) {
            continue;
        }
        if cfg.dependencies.is_empty() {
            visited.insert(*chain_id);
            continue;
        }
        for member in cfg.dependencies.keys() {
            visited.insert(*member);
        }
        clusters.push(DependencySet {
            dependencies: cfg.dependencies.clone(),
            override_message_expiry_window: None,
        });
    }

    // Sort clusters by their min chain id for determinism.
    clusters.sort_by_key(|ds| ds.dependencies.keys().next().copied().unwrap_or_default());

    Ok(clusters)
}

/// Adds a self-only [`DependencySet`] for every chain in `chain_ids` that no cluster in
/// `clusters` covers, keeping the output sorted by minimum chain id.
///
/// A chain that declares no interop dependencies is its own single-member cluster; without
/// an embedded depset, the fault proof program falls back to host-supplied data. Mirrors
/// Go's `op-core/superchain.GetDepset`, except that Go defaults only on an absent
/// `[interop]` section, not an empty one.
#[allow(clippy::zero_sized_map_values)]
pub fn with_single_chain_defaults<I>(
    mut clusters: Vec<DependencySet>,
    chain_ids: I,
) -> Vec<DependencySet>
where
    I: IntoIterator<Item = u64>,
{
    let covered: BTreeSet<u64> =
        clusters.iter().flat_map(|ds| ds.dependencies.keys().copied()).collect();

    let uncovered: BTreeSet<u64> =
        chain_ids.into_iter().filter(|id| !covered.contains(id)).collect();

    for chain_id in uncovered {
        clusters.push(DependencySet {
            dependencies: BTreeMap::from([(chain_id, ChainDependency {})]),
            override_message_expiry_window: None,
        });
    }

    clusters.sort_by_key(|ds| ds.dependencies.keys().next().copied().unwrap_or_default());
    clusters
}

#[cfg(test)]
#[allow(clippy::zero_sized_map_values)]
mod tests {
    use super::*;
    use alloc::vec;

    fn config(deps: &[u64]) -> InteropConfig {
        let dependencies = deps.iter().map(|id| (*id, ChainDependency {})).collect();
        InteropConfig { dependencies }
    }

    #[test]
    fn aggregate_empty_input() {
        let chains: Vec<(u64, &InteropConfig)> = vec![];
        let out = aggregate_clusters(chains).unwrap();
        assert!(out.is_empty());
    }

    #[test]
    fn aggregate_single_chain_cluster_of_one() {
        // A chain whose only dependency is itself.
        let cfg = config(&[1]);
        let out = aggregate_clusters(vec![(1u64, &cfg)]).unwrap();
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].dependencies.keys().copied().collect::<Vec<_>>(), vec![1]);
    }

    #[test]
    fn aggregate_single_cluster_of_n() {
        let cfg_a = config(&[1, 2, 3]);
        let cfg_b = config(&[1, 2, 3]);
        let cfg_c = config(&[1, 2, 3]);
        let out = aggregate_clusters(vec![(1u64, &cfg_a), (2u64, &cfg_b), (3u64, &cfg_c)]).unwrap();
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].dependencies.keys().copied().collect::<Vec<_>>(), vec![1, 2, 3]);
    }

    #[test]
    fn aggregate_two_disjoint_clusters() {
        let a1 = config(&[1, 2]);
        let a2 = config(&[1, 2]);
        let b1 = config(&[10, 11]);
        let b2 = config(&[10, 11]);
        let out =
            aggregate_clusters(vec![(1u64, &a1), (2u64, &a2), (10u64, &b1), (11u64, &b2)]).unwrap();
        assert_eq!(out.len(), 2);
        assert_eq!(out[0].dependencies.keys().copied().collect::<Vec<_>>(), vec![1, 2]);
        assert_eq!(out[1].dependencies.keys().copied().collect::<Vec<_>>(), vec![10, 11]);
    }

    #[test]
    fn aggregate_inconsistent_cluster() {
        // Chain 1 declares {1,2} but chain 2 only declares {2}.
        let a = config(&[1, 2]);
        let b = config(&[2]);
        let err = aggregate_clusters(vec![(1u64, &a), (2u64, &b)]).unwrap_err();
        assert_eq!(err, ClusterError::Inconsistent { a: 1, b: 2 });
    }

    #[test]
    fn single_chain_defaults_fill_uncovered_chains() {
        let cfg_a = config(&[1, 2]);
        let cfg_b = config(&[1, 2]);
        let clusters = aggregate_clusters(vec![(1u64, &cfg_a), (2u64, &cfg_b)]).unwrap();

        let out = with_single_chain_defaults(clusters, vec![1, 2, 5, 3]);

        // Sorted by min chain id: the {1,2} cluster, then self-only 3 and 5.
        let members: Vec<Vec<u64>> =
            out.iter().map(|ds| ds.dependencies.keys().copied().collect()).collect();
        assert_eq!(members, vec![vec![1, 2], vec![3], vec![5]]);
        assert!(out.iter().all(|ds| ds.override_message_expiry_window.is_none()));
    }

    #[test]
    fn single_chain_defaults_are_idempotent() {
        let out = with_single_chain_defaults(vec![], vec![7]);
        assert_eq!(with_single_chain_defaults(out.clone(), vec![7]), out);
    }

    #[test]
    fn single_chain_defaults_dedupe_repeated_chain_ids() {
        let out = with_single_chain_defaults(vec![], vec![7, 7]);
        assert_eq!(out.len(), 1);
    }

    #[test]
    fn aggregate_dangling_member() {
        // Chain 1 declares {1, 9} but chain 9 has no interop section.
        let a = config(&[1, 9]);
        let err = aggregate_clusters(vec![(1u64, &a)]).unwrap_err();
        assert_eq!(err, ClusterError::DanglingMember { a: 1, b: 9 });
    }
}
