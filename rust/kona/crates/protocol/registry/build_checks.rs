//! Checks that a custom-config build runs on the merged registry.
//!
//! Included by `build.rs` at build-script compile time and by `tests/build_checks.rs` at
//! integration-test compile time, which is why it lives at the crate root instead of under
//! `src/`.

use std::collections::BTreeSet;

use kona_genesis::{
    ChainList, ClusterError, DependencySet, REGISTRY_L1_CHAIN_IDS, Superchains, aggregate_clusters,
};

/// A merged registry that a fault proof program cannot prove every chain of.
#[derive(Debug, PartialEq, Eq)]
pub(crate) enum CustomConfigError {
    /// A dependency set lists a chain that is not in the chain list.
    UnknownChain(u64),
    /// A chain is in no dependency set.
    NoDependencySet {
        /// The chain's identifier.
        identifier: String,
        /// The chain's id.
        chain_id: u64,
    },
    /// The chains' `interop` sections do not form consistent clusters.
    InconsistentInterop(ClusterError),
    /// The chains' `interop` sections declare a cluster, listed here, that is not a dependency
    /// set.
    ClusterWithoutSet(Vec<u64>),
    /// A chain group that holds a custom chain has an L1 whose config the registry does not
    /// embed.
    UnsupportedL1 {
        /// The chain group's name.
        group: String,
        /// The chain group's L1 chain id.
        l1_chain_id: u64,
    },
}

/// Checks the merged dependency sets against the merged chain list and chain configs: every set
/// lists only known chains, every chain is in a set, and every cluster that the `interop`
/// sections declare is one of the sets.
///
/// Without an embedded set for its chains, the interop program reads the set from local
/// preimage key 8, which no dispute game can load.
pub(crate) fn check_dependency_sets(
    depsets: &[DependencySet],
    chain_list: &ChainList,
    superchains: &Superchains,
) -> Result<(), CustomConfigError> {
    let known: BTreeSet<u64> = chain_list.chains.iter().map(|chain| chain.chain_id).collect();
    let covered: BTreeSet<u64> =
        depsets.iter().flat_map(|ds| ds.dependencies.keys().copied()).collect();
    if let Some(chain_id) = covered.difference(&known).next() {
        return Err(CustomConfigError::UnknownChain(*chain_id));
    }
    if let Some(chain) = chain_list.chains.iter().find(|chain| !covered.contains(&chain.chain_id)) {
        return Err(CustomConfigError::NoDependencySet {
            identifier: chain.identifier.clone(),
            chain_id: chain.chain_id,
        });
    }

    let clusters =
        aggregate_clusters(
            superchains.superchains.iter().flat_map(|superchain| &superchain.chains).filter_map(
                |chain| chain.interop.as_ref().map(|interop| (chain.chain_id, interop)),
            ),
        )
        .map_err(CustomConfigError::InconsistentInterop)?;
    if let Some(cluster) = clusters
        .iter()
        .find(|cluster| !depsets.iter().any(|ds| ds.dependencies == cluster.dependencies))
    {
        return Err(CustomConfigError::ClusterWithoutSet(
            cluster.dependencies.keys().copied().collect(),
        ));
    }
    Ok(())
}

/// Checks that every chain group holding one of `custom_chain_ids` has an L1 in
/// [`REGISTRY_L1_CHAIN_IDS`].
///
/// On any other L1 the fault proof programs read the L1 config from local preimage key 7, which
/// no dispute game can load.
pub(crate) fn check_l1s(
    superchains: &Superchains,
    custom_chain_ids: &BTreeSet<u64>,
) -> Result<(), CustomConfigError> {
    let unsupported = superchains.superchains.iter().find(|superchain| {
        !REGISTRY_L1_CHAIN_IDS.contains(&superchain.config.l1.chain_id) &&
            superchain.chains.iter().any(|chain| custom_chain_ids.contains(&chain.chain_id))
    });
    if let Some(superchain) = unsupported {
        return Err(CustomConfigError::UnsupportedL1 {
            group: superchain.name.clone(),
            l1_chain_id: superchain.config.l1.chain_id,
        });
    }
    Ok(())
}
