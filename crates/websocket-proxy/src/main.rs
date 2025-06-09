mod auth;
mod client;
#[cfg(all(feature = "integration", test))]
mod integration;
mod metrics;
mod rate_limit;
mod registry;
mod server;
mod subscriber;

use crate::metrics::Metrics;
use crate::rate_limit::{InMemoryRateLimit, RateLimit};
use crate::registry::Registry;
use crate::server::Server;
use crate::subscriber::WebsocketSubscriber;
use axum::http::Uri;
use clap::Parser;
use dotenvy::dotenv;
use metrics_exporter_prometheus::PrometheusBuilder;
use rate_limit::RedisRateLimit;
use std::io::Write;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::signal::unix::{signal, SignalKind};
use tokio::sync::broadcast;
use tokio_util::sync::CancellationToken;
use tracing::{error, info, trace, warn, Level};
use tracing_subscriber::EnvFilter;

#[derive(Parser, Debug)]
#[command(author, version, about)]
struct Args {
    #[arg(
        long,
        env,
        default_value = "0.0.0.0:8545",
        help = "The address and port to listen on for incoming connections"
    )]
    listen_addr: SocketAddr,

    #[arg(
        long,
        env,
        value_delimiter = ',',
        help = "WebSocket URI of the upstream server to connect to"
    )]
    upstream_ws: Vec<Uri>,

    #[arg(
        long,
        env,
        default_value = "20",
        help = "Number of messages to buffer for lagging clients"
    )]
    message_buffer_size: usize,

    #[arg(
        long,
        env,
        default_value = "100",
        help = "Maximum number of concurrently connected clients"
    )]
    global_connections_limit: usize,

    #[arg(
        long,
        env,
        default_value = "10",
        help = "Maximum number of concurrently connected clients"
    )]
    per_ip_connections_limit: usize,
    #[arg(
        long,
        env,
        default_value = "false",
        help = "Enable brotli compression on messages to downstream clients"
    )]
    enable_compression: bool,

    #[arg(
        long,
        env,
        default_value = "X-Forwarded-For",
        help = "Header to use to determine the clients origin IP"
    )]
    ip_addr_http_header: String,

    #[arg(long, env, default_value = "info")]
    log_level: Level,

    /// Format for logs, can be json or text
    #[arg(long, env, default_value = "text")]
    log_format: String,

    /// Enable Prometheus metrics
    #[arg(long, env, default_value = "true")]
    metrics: bool,

    /// API Keys, if not provided will be an unauthenticated endpoint, should be in the format <app1>:<apiKey1>,<app2>:<apiKey2>,..
    #[arg(long, env, value_delimiter = ',', help = "API keys to allow")]
    api_keys: Vec<String>,

    /// Address to run the metrics server on
    #[arg(long, env, default_value = "0.0.0.0:9000")]
    metrics_addr: SocketAddr,

    /// Tags to add to every metrics emitted, should be in the format --metrics-global-labels label1=value1,label2=value2
    #[arg(long, env, default_value = "")]
    metrics_global_labels: String,

    /// Add the hostname as a label to all Prometheus metrics
    #[arg(long, env, default_value = "false")]
    metrics_host_label: bool,

    /// Maximum backoff allowed for upstream connections
    #[arg(long, env, default_value = "20")]
    subscriber_max_interval: u64,

    #[arg(
        long,
        env,
        help = "Redis URL for distributed rate limiting (e.g., redis://localhost:6379). If not provided, in-memory rate limiting will be used."
    )]
    redis_url: Option<String>,

    #[arg(
        long,
        env,
        default_value = "flashblocks",
        help = "Prefix for Redis keys"
    )]
    redis_key_prefix: String,
}

#[tokio::main]
async fn main() {
    dotenv().ok();
    let args = Args::parse();

    let log_format = args.log_format.to_lowercase();
    let log_level = args.log_level.to_string();

    if log_format == "json" {
        tracing_subscriber::fmt()
            .json()
            .with_env_filter(EnvFilter::new(log_level))
            .with_ansi(false)
            .init();
    } else {
        tracing_subscriber::fmt()
            .with_env_filter(EnvFilter::new(log_level))
            .with_ansi(false)
            .init();
    }

    let api_keys: Vec<String> = args
        .api_keys
        .into_iter()
        .filter(|s| !s.is_empty())
        .collect();
    let authentication = if api_keys.is_empty() {
        None
    } else {
        match auth::Authentication::try_from(api_keys) {
            Ok(auth) => Some(auth),
            Err(e) => {
                panic!("Failed to parse API Keys: {}", e)
            }
        }
    };

    if args.metrics {
        info!(
            message = "starting metrics server",
            address = args.metrics_addr.to_string()
        );

        let mut builder = PrometheusBuilder::new().with_http_listener(args.metrics_addr);

        if args.metrics_host_label {
            let hostname = hostname::get()
                .expect("could not find hostname")
                .into_string()
                .expect("could not convert hostname to string");
            builder = builder.add_global_label("hostname", hostname);
        }

        for (key, value) in parse_global_metrics(args.metrics_global_labels) {
            builder = builder.add_global_label(key, value);
        }

        builder
            .install()
            .expect("failed to setup Prometheus endpoint")
    }

    // Validate that we have at least one upstream URI
    if args.upstream_ws.is_empty() {
        error!(message = "no upstream URIs provided");
        panic!("No upstream URIs provided");
    }

    info!(message = "using upstream URIs", uris = ?args.upstream_ws);

    let metrics = Arc::new(Metrics::default());
    let metrics_clone = metrics.clone();

    let (send, _rec) = broadcast::channel(args.message_buffer_size);
    let sender = send.clone();

    let listener = move |data: String| {
        trace!(message = "received data", data = data);
        // Subtract one from receiver count, as we have to keep one receiver open at all times (see _rec)
        // to avoid the channel being closed. However this is not an active client connection.
        metrics_clone
            .active_connections
            .set((send.receiver_count() - 1) as f64);

        let message_data = if args.enable_compression {
            let data_bytes = data.as_bytes();
            let mut compressed_data_bytes = Vec::new();
            {
                let mut compressor =
                    brotli::CompressorWriter::new(&mut compressed_data_bytes, 4096, 5, 22);
                compressor.write_all(data_bytes).unwrap();
            }
            compressed_data_bytes
        } else {
            data.into_bytes()
        };

        match send.send(message_data) {
            Ok(_) => (),
            Err(e) => error!(message = "failed to send data", error = e.to_string()),
        }
    };

    let token = CancellationToken::new();
    let mut subscriber_tasks = Vec::new();

    // Start a subscriber for each upstream URI
    for (index, uri) in args.upstream_ws.iter().enumerate() {
        let uri_clone = uri.clone();
        let listener_clone = listener.clone();
        let token_clone = token.clone();
        let metrics_clone = metrics.clone();

        let mut subscriber = WebsocketSubscriber::new(
            uri_clone.clone(),
            listener_clone,
            args.subscriber_max_interval,
            metrics_clone,
        );

        let task = tokio::spawn(async move {
            info!(
                message = "starting subscriber",
                index = index,
                uri = uri_clone.to_string()
            );
            subscriber.run(token_clone).await;
        });

        subscriber_tasks.push(task);
    }

    let registry = Registry::new(sender, metrics.clone());

    let rate_limiter = match &args.redis_url {
        Some(redis_url) => {
            info!(message = "Using Redis rate limiter", redis_url = redis_url);
            match RedisRateLimit::new(
                redis_url,
                args.global_connections_limit,
                args.per_ip_connections_limit,
                &args.redis_key_prefix,
            ) {
                Ok(limiter) => {
                    info!(message = "Connected to Redis successfully");
                    Arc::new(limiter) as Arc<dyn RateLimit>
                }
                Err(e) => {
                    error!(
                        message =
                            "Failed to connect to Redis, falling back to in-memory rate limiting",
                        error = e.to_string()
                    );
                    Arc::new(InMemoryRateLimit::new(
                        args.global_connections_limit,
                        args.per_ip_connections_limit,
                    )) as Arc<dyn RateLimit>
                }
            }
        }
        None => {
            info!(message = "Using in-memory rate limiter");
            Arc::new(InMemoryRateLimit::new(
                args.global_connections_limit,
                args.per_ip_connections_limit,
            )) as Arc<dyn RateLimit>
        }
    };

    let server = Server::new(
        args.listen_addr,
        registry.clone(),
        metrics,
        rate_limiter,
        authentication,
        args.ip_addr_http_header,
    );
    let server_task = server.listen(token.clone());

    let mut interrupt = signal(SignalKind::interrupt()).unwrap();
    let mut terminate = signal(SignalKind::terminate()).unwrap();

    tokio::select! {
        _ = futures::future::join_all(subscriber_tasks) => {
            info!("all subscriber tasks terminated");
            token.cancel();
        },
        _ = server_task => {
            info!("server task terminated");
            token.cancel();
        }
        _ = interrupt.recv() => {
            info!("process interrupted, shutting down");
            token.cancel();
        }
        _ = terminate.recv() => {
            info!("process terminated, shutting down");
            token.cancel();
        }
    }
}

fn parse_global_metrics(metrics: String) -> Vec<(String, String)> {
    let mut result = Vec::new();

    for metric in metrics.split(',') {
        if metric.is_empty() {
            continue;
        }

        let parts = metric
            .splitn(2, '=')
            .map(|s| s.to_string())
            .collect::<Vec<String>>();

        if parts.len() != 2 {
            warn!(
                message = "malformed global metric: invalid count",
                metric = metric
            );
            continue;
        }

        let label = parts[0].to_string();
        let value = parts[1].to_string();

        if label.is_empty() || value.is_empty() {
            warn!(
                message = "malformed global metric: empty value",
                metric = metric
            );
            continue;
        }

        result.push((label, value));
    }

    result
}

#[cfg(test)]
mod test {
    use crate::parse_global_metrics;

    #[test]
    fn test_parse_global_metrics() {
        assert_eq!(
            parse_global_metrics("".into()),
            Vec::<(String, String)>::new(),
        );

        assert_eq!(
            parse_global_metrics("key=value".into()),
            vec![("key".into(), "value".into())]
        );

        assert_eq!(
            parse_global_metrics("key=value,key2=value2".into()),
            vec![
                ("key".into(), "value".into()),
                ("key2".into(), "value2".into())
            ],
        );

        assert_eq!(parse_global_metrics("gibberish".into()), Vec::new());

        assert_eq!(
            parse_global_metrics("key=value,key2=,".into()),
            vec![("key".into(), "value".into())],
        );
    }
}
