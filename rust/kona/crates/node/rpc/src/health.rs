use async_trait::async_trait;
use jsonrpsee::core::RpcResult;
use rollup_boost::Health;
use tokio::sync::oneshot;

use crate::jsonrpsee::HealthzApiServer;

/// Key for the rollup boost health status.
/// +----------------+-------------------------------+--------------------------------------+-------------------------------+
/// | Execution Mode | Healthy                       | `PartialContent`                       | Service Unavailable           |
/// +----------------+-------------------------------+--------------------------------------+-------------------------------+
/// | Enabled        | - Request-path: L2 succeeds   | - Request-path: builder fails/stale  | - Request-path: L2 fails      |
/// |                |   (get/new payload) → 200     |   while L2 succeeds → 206            |   (error from L2) → 503       |
/// |                | - Background: builder         | - Background: builder fetch fails or | - Background: never sets 503  |
/// |                |   latest-unsafe is fresh →    |   latest-unsafe is stale → 206       |                               |
/// |                |   200                         |                                      |                               |
/// +----------------+-------------------------------+--------------------------------------+-------------------------------+
/// | `DryRun`         | - Request-path: L2 succeeds   | - Never set in `DryRun`                | - Request-path: L2 fails      |
/// |                |   (always returns L2) → 200   |   (degrade only in Enabled)          |   (error from L2) → 503       |
/// |                | - Background: builder stale   |                                      | - Background: never sets 503  |
/// |                |   ignored (remains 200)       |                                      |                               |
/// +----------------+-------------------------------+--------------------------------------+-------------------------------+
/// | Disabled       | - Request-path: L2 succeeds   | - Never set in Disabled              | - Request-path: L2 fails      |
/// |                |   (builder skipped) → 200     |   (degrade only in Enabled)          |   (error from L2) → 503       |
/// |                | - Background: N/A             |                                      | - Background: never sets 503  |
/// +----------------+-------------------------------+--------------------------------------+-------------------------------+
///
/// This type is the same as [`Health`], but it implements `serde::Serialize`
/// and `serde::Deserialize`.
#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
pub enum RollupBoostHealth {
    /// Rollup boost is healthy.
    Healthy,
    /// Rollup boost is partially healthy.
    PartialContent,
    /// Rollup boost service is unavailable.
    ServiceUnavailable,
}

impl From<Health> for RollupBoostHealth {
    fn from(health: Health) -> Self {
        match health {
            Health::Healthy => Self::Healthy,
            Health::PartialContent => Self::PartialContent,
            Health::ServiceUnavailable => Self::ServiceUnavailable,
        }
    }
}

/// A healthcheck response for the RPC server.
#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
pub struct HealthzResponse {
    /// The application version.
    pub version: String,
}

/// A healthcheck response for the rollup boost health.
#[derive(Debug, Clone, serde::Deserialize, serde::Serialize)]
pub struct RollupBoostHealthzResponse {
    /// The rollup boost health.
    pub rollup_boost_health: RollupBoostHealth,
}

/// A query to get the health of the rollup boost server.
#[derive(Debug)]
pub struct RollupBoostHealthQuery {
    /// The sender to send the rollup boost health to.
    pub sender: oneshot::Sender<RollupBoostHealth>,
}

/// The healthz rpc server.
#[derive(Debug, Clone)]
pub struct HealthzRpc {}

#[async_trait]
impl HealthzApiServer for HealthzRpc {
    async fn healthz(&self) -> RpcResult<HealthzResponse> {
        Ok(HealthzResponse { version: env!("CARGO_PKG_VERSION").to_string() })
    }
}
