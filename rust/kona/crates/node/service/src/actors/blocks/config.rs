use url::Url;

/// Configuration for consuming canonical unsafe blocks from a sequencer blocks server.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct BlocksClientConfig {
    /// Base URL of the sequencer blocks server.
    pub endpoint: Url,
}

impl BlocksClientConfig {
    /// Creates a blocks client configuration for the provided endpoint.
    pub const fn new(endpoint: Url) -> Self {
        Self { endpoint }
    }
}
