use clap::Parser;
use url::Url;

#[derive(Parser, Clone, Debug)]
pub struct FlashblocksArgs {
    /// Enable Flashblocks client
    #[arg(long, env, default_value = "false")]
    pub flashblocks: bool,

    /// Flashblocks Builder WebSocket URL
    #[arg(long, env, default_value = "ws://127.0.0.1:1111")]
    pub flashblocks_builder_url: Url,

    /// Flashblocks WebSocket host for outbound connections
    #[arg(long, env, default_value = "127.0.0.1")]
    pub flashblocks_host: String,

    /// Flashblocks WebSocket port for outbound connections
    #[arg(long, env, default_value = "1112")]
    pub flashblocks_port: u16,

    /// Time used for timeout if builder disconnected
    #[arg(long, env, default_value = "5000")]
    pub flashblock_builder_ws_reconnect_ms: u64,
}
