//! CLI Options

use kona_genesis::RollupConfig;

/// Metrics to record various CLI options.
#[derive(Debug, Clone, PartialEq)]
pub struct CliMetrics;

impl CliMetrics {
    /// The identifier for the cli metrics gauge.
    pub const IDENTIFIER: &'static str = "kona_cli_opts";

    /// The P2P Scoring level (disabled if "off").
    pub const P2P_PEER_SCORING_LEVEL: &'static str = "kona_node_peer_scoring_level";

    /// Whether P2P Topic Scoring is enabled.
    pub const P2P_TOPIC_SCORING_ENABLED: &'static str = "kona_node_topic_scoring_enabled";

    /// Whether P2P banning is enabled.
    pub const P2P_BANNING_ENABLED: &'static str = "kona_node_banning_enabled";

    /// The value for peer redialing.
    pub const P2P_PEER_REDIALING: &'static str = "kona_node_peer_redialing";

    /// Whether flood publishing is enabled.
    pub const P2P_FLOOD_PUBLISH: &'static str = "kona_node_flood_publish";

    /// The interval to send FINDNODE requests through discv5.
    pub const P2P_DISCOVERY_INTERVAL: &'static str = "kona_node_discovery_interval";

    /// The IP to advertise via P2P.
    pub const P2P_ADVERTISE_IP: &'static str = "kona_node_advertise_ip";

    /// The advertised tcp port via P2P.
    pub const P2P_ADVERTISE_TCP_PORT: &'static str = "kona_node_advertise_tcp";

    /// The advertised udp port via P2P.
    pub const P2P_ADVERTISE_UDP_PORT: &'static str = "kona_node_advertise_udp";

    /// The low-tide peer count.
    pub const P2P_PEERS_LO: &'static str = "kona_node_peers_lo";

    /// The high-tide peer count.
    pub const P2P_PEERS_HI: &'static str = "kona_node_peers_hi";

    /// The gossip mesh d option.
    pub const P2P_GOSSIP_MESH_D: &'static str = "kona_node_gossip_mesh_d";

    /// The gossip mesh d lo option.
    pub const P2P_GOSSIP_MESH_D_LO: &'static str = "kona_node_gossip_mesh_d_lo";

    /// The gossip mesh d hi option.
    pub const P2P_GOSSIP_MESH_D_HI: &'static str = "kona_node_gossip_mesh_d_hi";

    /// The gossip mesh d lazy option.
    pub const P2P_GOSSIP_MESH_D_LAZY: &'static str = "kona_node_gossip_mesh_d_lazy";

    /// The duration to ban peers.
    pub const P2P_BAN_DURATION: &'static str = "kona_node_ban_duration";

    /// Hardfork activation times.
    pub const HARDFORK_ACTIVATION_TIMES: &'static str = "kona_node_hardforks";

    /// Top-level rollup config settings.
    pub const ROLLUP_CONFIG: &'static str = "kona_node_rollup_config";
}

/// Initializes metrics for the rollup config.
pub fn init_rollup_config_metrics(config: &RollupConfig) {
    metrics::describe_gauge!(
        CliMetrics::ROLLUP_CONFIG,
        "Rollup configuration settings for the OP Stack"
    );
    metrics::describe_gauge!(
        CliMetrics::HARDFORK_ACTIVATION_TIMES,
        "Activation times for hardforks in the OP Stack"
    );

    metrics::gauge!(
        CliMetrics::ROLLUP_CONFIG,
        &[
            ("l1_genesis_block_num", config.genesis.l1.number.to_string()),
            ("l2_genesis_block_num", config.genesis.l2.number.to_string()),
            ("genesis_l2_time", config.genesis.l2_time.to_string()),
            ("l1_chain_id", config.l1_chain_id.to_string()),
            ("l2_chain_id", config.l2_chain_id.to_string()),
            ("block_time", config.block_time.to_string()),
            ("max_sequencer_drift", config.max_sequencer_drift.to_string()),
            ("sequencer_window_size", config.seq_window_size.to_string()),
            ("channel_timeout", config.channel_timeout.to_string()),
            ("granite_channel_timeout", config.granite_channel_timeout.to_string()),
            ("batch_inbox_address", config.batch_inbox_address.to_string()),
            ("deposit_contract_address", config.deposit_contract_address.to_string()),
            ("l1_system_config_address", config.l1_system_config_address.to_string()),
            ("protocol_versions_address", config.protocol_versions_address.to_string()),
        ]
    )
    .set(1);

    for (fork_name, activation_time) in config.hardforks.iter() {
        // Set the value of the metric for the given hardfork, using `-1` as a signal that the
        // fork is not scheduled.
        metrics::gauge!(CliMetrics::HARDFORK_ACTIVATION_TIMES, "fork" => fork_name)
            .set(activation_time.map(|t| t as f64).unwrap_or(-1f64));
    }
}
