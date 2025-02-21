use metrics::{Counter, Histogram};
use metrics_derive::Metrics;

#[derive(Metrics)]
#[metrics(scope = "rpc")]
pub struct ServerMetrics {
    #[metric(describe = "Count of forkchoice_updated_v3 calls proxied to the builder")]
    pub fcu_count: Counter,

    #[metric(describe = "Count of new_payload_v3 calls proxied to the builder")]
    pub new_payload_count: Counter,

    #[metric(describe = "Count of get_payload_v3 calls proxied to the builder")]
    pub get_payload_count: Counter,

    #[metric(describe = "Latency for `engine_newPayloadV3`")]
    pub new_payload_v3: Histogram,

    #[metric(describe = "Latency for `engine_getPayloadV3`")]
    pub get_payload_v3: Histogram,

    #[metric(describe = "Latency for `engine_forkChoiceUpdatedV3`")]
    pub fork_choice_updated_v3: Histogram,

    #[metric(describe = "Count of blocks created by the builder")]
    pub blocks_created_by_builder: Counter,

    #[metric(describe = "Count of blocks created by the L2 builder")]
    pub blocks_created_by_l2: Counter,
}
