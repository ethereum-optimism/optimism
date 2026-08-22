//! Metric registration for the chains a supernode runs.

/// Registers the metrics of every crate the supernode runs, for one chain.
///
/// The kona crates label their own series with the chain id they were registered for, so a
/// supernode registers them once per configured chain and its series separate by chain without
/// lokahi labelling anything itself. Registration also publishes each family's zero value, so a
/// chain that has not produced a block yet is a flat line rather than a gap in a dashboard.
///
/// A single-chain kona-node does exactly this once; the only difference here is the loop.
pub(crate) fn init_chain_metrics(chain_id: u64) {
    kona_disc::Metrics::init(chain_id);
    kona_gossip::Metrics::init(chain_id);
    kona_engine::Metrics::init(chain_id);
    kona_derive::Metrics::init(chain_id);
    kona_node_service::Metrics::init(chain_id);
    kona_providers_alloy::Metrics::init(chain_id);
}
