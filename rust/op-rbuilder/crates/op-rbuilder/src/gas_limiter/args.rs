use clap::Args;

#[derive(Debug, Clone, Default, PartialEq, Eq, Args)]
pub struct GasLimiterArgs {
    /// Enable address-based gas rate limiting
    #[arg(long = "gas-limiter.enabled", env)]
    pub gas_limiter_enabled: bool,

    /// Maximum gas per address in token bucket. Defaults to 10 million gas.
    #[arg(
        long = "gas-limiter.max-gas-per-address",
        env,
        default_value = "10000000"
    )]
    pub max_gas_per_address: u64,

    /// Gas refill rate per block. Defaults to 1 million gas per block.
    #[arg(
        long = "gas-limiter.refill-rate-per-block",
        env,
        default_value = "1000000"
    )]
    pub refill_rate_per_block: u64,

    /// How many blocks to wait before cleaning up stale buckets for addresses.
    #[arg(long = "gas-limiter.cleanup-interval", env, default_value = "100")]
    pub cleanup_interval: u64,
}
