use clap::{Parser, Subcommand};
use tracing::Level;

use crate::{
    client::rpc::{BuilderArgs, L2ClientArgs},
    server::ExecutionMode,
};

#[derive(Parser, Debug)]
#[clap(author, version, about)]
pub struct Args {
    #[command(subcommand)]
    pub command: Option<Commands>,

    #[clap(flatten)]
    pub builder: BuilderArgs,

    #[clap(flatten)]
    pub l2_client: L2ClientArgs,

    /// Disable using the proposer to sync the builder node
    #[arg(long, env, default_value = "false")]
    pub no_boost_sync: bool,

    /// Host to run the server on
    #[arg(long, env, default_value = "0.0.0.0")]
    pub rpc_host: String,

    /// Port to run the server on
    #[arg(long, env, default_value = "8081")]
    pub rpc_port: u16,

    // Enable tracing
    #[arg(long, env, default_value = "false")]
    pub tracing: bool,

    // Enable Prometheus metrics
    #[arg(long, env, default_value = "false")]
    pub metrics: bool,

    /// Host to run the metrics server on
    #[arg(long, env, default_value = "0.0.0.0")]
    pub metrics_host: String,

    /// Port to run the metrics server on
    #[arg(long, env, default_value = "9090")]
    pub metrics_port: u16,

    /// OTLP endpoint
    #[arg(long, env, default_value = "http://localhost:4317")]
    pub otlp_endpoint: String,

    /// Log level
    #[arg(long, env, default_value = "info")]
    pub log_level: Level,

    /// Log format
    #[arg(long, env, default_value = "text")]
    pub log_format: LogFormat,

    /// Host to run the debug server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub debug_host: String,

    /// Debug server port
    #[arg(long, env, default_value = "5555")]
    pub debug_server_port: u16,

    /// Execution mode to start rollup boost with
    #[arg(long, env, default_value = "enabled")]
    pub execution_mode: ExecutionMode,
}

#[derive(Clone, Debug)]
pub enum LogFormat {
    Json,
    Text,
}

impl std::str::FromStr for LogFormat {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "json" => Ok(LogFormat::Json),
            "text" => Ok(LogFormat::Text),
            _ => Err("Invalid log format".into()),
        }
    }
}

#[derive(Subcommand, Debug)]
pub enum Commands {
    /// Debug commands
    Debug {
        #[command(subcommand)]
        command: DebugCommands,
    },
}

#[derive(Subcommand, Debug)]
pub enum DebugCommands {
    /// Set the execution mode
    SetExecutionMode { execution_mode: ExecutionMode },

    /// Get the execution mode
    ExecutionMode {},
}
