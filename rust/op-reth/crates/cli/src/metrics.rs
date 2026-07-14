//! Metrics registered by the OP-Reth CLI.

use metrics::{describe_gauge, gauge};
use reth_chainspec::{ForkCondition, Hardforks};
use reth_optimism_chainspec::{OpChainSpec, OpHardfork};

const HARDFORK_ACTIVATION: &str = "chain_hardfork_activation";

/// Registers activation metrics for the configured chain's L2 (OP Stack) hardforks.
///
/// Only [`OpHardfork`] variants are reported; the underlying L1 (Ethereum) hardforks are omitted.
pub fn register_hardfork_activation_metrics(chain_spec: &OpChainSpec) {
    describe_gauge!(
        HARDFORK_ACTIVATION,
        "Configured chain L2 hardfork activation block or timestamp"
    );

    for hardfork in OpHardfork::VARIANTS {
        let Some((activation_basis, value)) = activation(&chain_spec.fork(*hardfork)) else {
            continue;
        };

        // Lowercase to match op-node's fork label values (e.g. "canyon").
        gauge!(
            HARDFORK_ACTIVATION,
            "fork" => hardfork.name().to_lowercase(),
            "activation_basis" => activation_basis
        )
        .set(value as f64);
    }
}

const fn activation(condition: &ForkCondition) -> Option<(&'static str, u64)> {
    match condition {
        ForkCondition::Block(block) => Some(("block", *block)),
        ForkCondition::TTD { activation_block_number, .. } => {
            Some(("block", *activation_block_number))
        }
        ForkCondition::Timestamp(timestamp) => Some(("timestamp", *timestamp)),
        ForkCondition::Never => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use metrics_exporter_prometheus::PrometheusBuilder;
    use reth_chainspec::{ChainHardforks, EthereumHardfork, ForkCondition, Hardfork};
    use reth_node_metrics::recorder::try_install_prometheus_recorder_with_builder;
    use reth_optimism_chainspec::{OP_DEV, OpChainSpecBuilder, OpHardfork};

    #[test]
    fn registers_only_l2_hardfork_activation_metrics() {
        let recorder = try_install_prometheus_recorder_with_builder(PrometheusBuilder::new())
            .unwrap_or_else(|_| reth_node_metrics::recorder::install_prometheus_recorder());

        let chain_spec = OpChainSpecBuilder::default()
            .chain(OP_DEV.chain)
            .genesis(OP_DEV.genesis.clone())
            .with_forks(ChainHardforks::new(vec![
                // An L1 fork that must not be reported.
                (EthereumHardfork::Shanghai.boxed(), ForkCondition::Timestamp(20)),
                (OpHardfork::Bedrock.boxed(), ForkCondition::Block(10)),
                (OpHardfork::Canyon.boxed(), ForkCondition::Timestamp(30)),
                (OpHardfork::Regolith.boxed(), ForkCondition::Never),
            ]))
            .build();

        register_hardfork_activation_metrics(&chain_spec);

        let exposition = recorder.handle().render();
        assert!(
            exposition.contains(
                r#"reth_chain_hardfork_activation{fork="bedrock",activation_basis="block"} 10"#
            ),
            "{exposition}"
        );
        assert!(
            exposition.contains(
                r#"reth_chain_hardfork_activation{fork="canyon",activation_basis="timestamp"} 30"#
            ),
            "{exposition}"
        );
        assert!(
            !exposition.contains(r#"fork="regolith""#),
            "Never forks must be omitted: {exposition}"
        );
        assert!(
            !exposition.contains(r#"fork="shanghai""#),
            "L1 (Ethereum) forks must be omitted: {exposition}"
        );
    }
}
