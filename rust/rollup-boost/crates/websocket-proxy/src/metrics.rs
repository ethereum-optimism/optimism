use axum::extract::ws::Message;
use metrics::{counter, Counter, Gauge};
use metrics_derive::Metrics;
#[derive(Metrics)]
#[metrics(scope = "websocket_proxy")]
pub struct Metrics {
    #[metric(describe = "Messages sent to clients")]
    pub sent_messages: Counter,

    #[metric(describe = "Count of messages that were unable to be sent")]
    pub failed_messages: Counter,

    #[metric(describe = "Count of new connections opened")]
    pub new_connections: Counter,

    #[metric(describe = "Count of number of connections closed")]
    pub closed_connections: Counter,

    #[metric(describe = "Count the number of connections which lagged and then disconnected")]
    pub lagged_connections: Counter,

    #[metric(describe = "Number of client connections currently open")]
    pub active_connections: Gauge,

    #[metric(describe = "Count of rate limited request")]
    pub rate_limited_requests: Counter,

    #[metric(describe = "Count of unauthorized requests with invalid API keys")]
    pub unauthorized_requests: Counter,

    #[metric(describe = "Count of times upstream receiver was closed/errored")]
    pub upstream_errors: Counter,

    #[metric(describe = "Number of active upstream connections")]
    pub upstream_connections: Gauge,

    #[metric(describe = "Number of upstream connection attempts")]
    pub upstream_connection_attempts: Counter,

    #[metric(describe = "Number of successful upstream connections")]
    pub upstream_connection_successes: Counter,

    #[metric(describe = "Number of failed upstream connection attempts")]
    pub upstream_connection_failures: Counter,

    #[metric(describe = "Total bytes broadcasted to clients")]
    pub bytes_broadcasted: Counter,

    #[metric(describe = "Count of clients disconnected due to pong timeout")]
    pub client_pong_disconnects: Counter,
}

fn get_message_size(msg: &Message) -> u64 {
    match msg {
        Message::Text(text) => text.len() as u64,
        Message::Binary(data) => data.len() as u64,
        Message::Ping(data) => data.len() as u64,
        Message::Pong(data) => data.len() as u64,
        Message::Close(_) => 0,
    }
}

impl Metrics {
    pub fn proxy_connections_by_app(&self, app: &str) {
        counter!("websocket_proxy.connections_by_app", "app" => app.to_owned()).increment(1);
    }

    pub fn message_received_from_upstream(&self, upstream: &str) {
        counter!("websocket_proxy.upstream_messages", "upstream" => upstream.to_owned())
            .increment(1);
    }

    pub fn record_message_sent(&self, msg: &Message) {
        self.sent_messages.increment(1);
        self.bytes_broadcasted.increment(get_message_size(msg));
    }
}
