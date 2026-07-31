use metrics::{Counter, Gauge, Histogram};
use metrics_derive::Metrics;

#[derive(Metrics, Clone)]
#[metrics(scope = "reth_flashblocks")]
pub struct Metrics {
    #[metric(describe = "Count of times upstream receiver was closed/errored")]
    pub upstream_errors: Counter,

    #[metric(describe = "Count of messages received from the upstream source")]
    pub upstream_messages: Gauge,

    #[metric(describe = "Time taken to process a message")]
    pub block_processing_duration: Histogram,

    #[metric(describe = "Time taken to process a websocket message")]
    pub websocket_processing_duration: Histogram,

    #[metric(describe = "Count of times flashblocks get_transaction_count is called")]
    pub get_transaction_count: Counter,

    #[metric(describe = "Count of times flashblocks get_transaction_receipt is called")]
    pub get_transaction_receipt: Counter,

    #[metric(describe = "Count of times flashblocks get_balance is called")]
    pub get_balance: Counter,

    #[metric(describe = "Count of times flashblocks get_block_by_number is called")]
    pub get_block_by_number: Counter,

    #[metric(describe = "Number of flashblocks in a block")]
    pub flashblocks_in_block: Histogram,
}
