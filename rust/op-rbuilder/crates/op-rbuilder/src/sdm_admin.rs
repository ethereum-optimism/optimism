//! Local SDM PostExec opt-in for op-rbuilder.
//!
//! The protocol gate (chain spec Interop activation) controls *whether* a block may
//! carry a PostExec tx — that's a consensus rule shared by every node. This module
//! adds an orthogonal *operator* gate: even on an SDM-active chain, the local
//! builder produces PostExec txs only when the operator has explicitly opted in
//! via [`admin_setSdmPostExecOptIn`]. Both gates must be true.
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
/// handler (writer) and every payload-builder ctx (reader).
pub type SdmPostExecOptInFlag = Arc<AtomicBool>;

/// Status snapshot returned by `admin_sdmStatus`.
///
/// Mirrors the op-node side so a single client surface can render either node's
/// state without translation.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SdmStatus {
    /// Whether the operator has opted in via `admin_setSdmPostExecOptIn`.
    pub post_exec_opt_in: bool,
    /// Whether SDM is active per the chain spec at `query_timestamp`.
    pub protocol_active: bool,
    /// AND of the above — the actual decision the builder will make for a block
    /// at `query_timestamp`.
    pub effective: bool,
    /// Activation timestamp of the protocol gate (Interop) if scheduled.
    pub activation_time: Option<u64>,
}

#[cfg_attr(not(test), rpc(server, namespace = "admin"))]
#[cfg_attr(test, rpc(server, client, namespace = "admin"))]
pub trait SdmAdminApi {
    /// Toggle local PostExec production. Starts disabled on process boot.
    #[method(name = "setSdmPostExecOptIn")]
    fn set_sdm_post_exec_opt_in(&self, enabled: bool) -> RpcResult<()>;

    /// Report the local opt-in flag, the chain-spec gate at `query_timestamp`,
    /// and the AND. If `query_timestamp` is omitted, uses the current wall-clock
    /// time, which is good enough for "is this builder configured to produce now?".
    #[method(name = "sdmStatus")]
    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus>;
}

#[derive(Clone)]
pub struct SdmAdminExt {
    opt_in: SdmPostExecOptInFlag,
    chain_spec: Arc<OpChainSpec>,
}

impl SdmAdminExt {
    pub fn new(opt_in: SdmPostExecOptInFlag, chain_spec: Arc<OpChainSpec>) -> Self {
        Self { opt_in, chain_spec }
    }
}

impl SdmAdminApiServer for SdmAdminExt {
    fn set_sdm_post_exec_opt_in(&self, enabled: bool) -> RpcResult<()> {
        self.opt_in.store(enabled, Ordering::Release);
        gauge!("op_rbuilder_flags_sdm_enabled").set(enabled as i32);
        Ok(())
    }

    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus> {
        let timestamp = query_timestamp.unwrap_or_else(current_unix_timestamp);
        let opt_in = self.opt_in.load(Ordering::Acquire);
        let protocol_active = is_sdm_active_at_timestamp(&*self.chain_spec, timestamp);
        let activation_time = match self.chain_spec.op_fork_activation(OpHardfork::Lagoon) {
            ForkCondition::Timestamp(t) => Some(t),
            _ => None,
        };
        Ok(SdmStatus {
            post_exec_opt_in: opt_in,
            protocol_active,
            effective: opt_in && protocol_active,
            activation_time,
        })
    }
}

fn current_unix_timestamp() -> u64 {
    UNIX_EPOCH.elapsed().map(|d| d.as_secs()).unwrap_or(0)
}
