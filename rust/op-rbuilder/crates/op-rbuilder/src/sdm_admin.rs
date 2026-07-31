//! Local SDM PostExec opt-in for op-rbuilder.
//!
//! The protocol gate (hardfork activation) is a consensus rule shared by every node. This module
//! adds an orthogonal *operator* gate: even on an SDM-active chain, the local builder produces
//! PostExec txs only when the operator has opted in via [`admin_setOperatorSdmOptIn`]. Both must
//! be true in order for SDM to be active.
//!
//! State is in-memory and starts disabled on every process boot; persistence is
//! deliberately out of scope.

use alloy_hardforks::ForkCondition;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use metrics::gauge;
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_evm::is_sdm_active_at_timestamp;
use reth_optimism_forks::{OpHardfork, OpHardforks};
use serde::{Deserialize, Serialize};
use std::{
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    time::UNIX_EPOCH,
};

/// Shared "operator wants to produce PostExec" flag. Cloned into both the RPC
/// handler (writer) and every payload-builder ctx (reader); all clones observe the
/// same atomic. Access goes through [`OperatorSdmOptIn::enabled`]/[`OperatorSdmOptIn::set`]
/// so the memory ordering lives in one place rather than at each call site.
#[derive(Debug, Clone, Default)]
pub struct OperatorSdmOptIn {
    inner: Arc<AtomicBool>,
}

impl OperatorSdmOptIn {
    /// Creates an opt-in flag initialized to `enabled`.
    pub fn new(enabled: bool) -> Self {
        let this = Self::default();
        this.set(enabled);
        this
    }

    /// Returns the current opt-in state.
    pub fn enabled(&self) -> bool {
        self.inner.load(Ordering::Acquire)
    }

    /// Sets the opt-in state.
    pub fn set(&self, enabled: bool) {
        self.inner.store(enabled, Ordering::Release);
    }
}

/// Status snapshot returned by `admin_sdmStatus`.
///
/// Mirrors the op-reth side so a single client surface can render either builder's
/// state without translation.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SdmStatus {
    /// Whether the operator has opted in via `admin_setOperatorSdmOptIn`.
    pub operator_sdm_opt_in: bool,
    /// Whether SDM is active per the chain spec at `query_timestamp`.
    pub protocol_active: bool,
    /// AND of the above — the actual decision the builder will make for a block
    /// at `query_timestamp`.
    pub effective: bool,
    /// Activation timestamp of the protocol gate if scheduled.
    pub activation_time: Option<u64>,
}

#[cfg_attr(not(test), rpc(server, namespace = "admin"))]
#[cfg_attr(test, rpc(server, client, namespace = "admin"))]
pub trait SdmAdminApi {
    /// Toggle local PostExec production. Starts disabled on process boot.
    #[method(name = "setOperatorSdmOptIn")]
    fn set_operator_sdm_opt_in(&self, enabled: bool) -> RpcResult<()>;

    /// Report the local opt-in flag, the chain-spec gate at `query_timestamp`,
    /// and the AND. If `query_timestamp` is omitted, uses the current wall-clock
    /// time, which is good enough for "is this builder configured to produce now?".
    #[method(name = "sdmStatus")]
    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus>;
}

#[derive(Clone)]
pub struct SdmAdminExt {
    opt_in: OperatorSdmOptIn,
    chain_spec: Arc<OpChainSpec>,
}

impl SdmAdminExt {
    pub fn new(opt_in: OperatorSdmOptIn, chain_spec: Arc<OpChainSpec>) -> Self {
        Self { opt_in, chain_spec }
    }
}

impl SdmAdminApiServer for SdmAdminExt {
    fn set_operator_sdm_opt_in(&self, enabled: bool) -> RpcResult<()> {
        self.opt_in.set(enabled);
        gauge!("op_rbuilder_flags_sdm_enabled").set(enabled as i32);
        Ok(())
    }

    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus> {
        let timestamp = query_timestamp.unwrap_or_else(current_unix_timestamp);
        let opt_in = self.opt_in.enabled();
        let protocol_active = is_sdm_active_at_timestamp(&*self.chain_spec, timestamp);
        let activation_time = match self.chain_spec.op_fork_activation(OpHardfork::Lagoon) {
            ForkCondition::Timestamp(t) => Some(t),
            _ => None,
        };
        Ok(SdmStatus {
            operator_sdm_opt_in: opt_in,
            protocol_active,
            effective: opt_in && protocol_active,
            activation_time,
        })
    }
}

fn current_unix_timestamp() -> u64 {
    UNIX_EPOCH.elapsed().map(|d| d.as_secs()).unwrap_or(0)
}
