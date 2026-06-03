//! Local SDM `PostExec` opt-in admin RPC for op-reth's payload builder.
//!
//! The protocol gate (chain spec Interop activation) is a consensus rule shared by every node.
//! This module adds an orthogonal *operator* gate: even on an SDM-active chain, op-reth's
//! standard payload builder produces `PostExec` txs only when the operator has explicitly opted
//! in via `admin_setSdmPostExecOptIn`. Both gates must be true.
//!
//! State is in-memory and starts disabled on every process boot; persistence is deliberately
//! out of scope.

use alloy_hardforks::ForkCondition;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use reth_optimism_forks::{OpHardfork, OpHardforks};
use reth_optimism_payload_builder::config::SdmPostExecOptIn;
use serde::{Deserialize, Serialize};
use std::{
    sync::Arc,
    time::{SystemTime, UNIX_EPOCH},
};

/// Status snapshot returned by `admin_sdmStatus`. Shape matches op-rbuilder and op-node so a
/// single client view can render any participant's state.
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SdmStatus {
    /// Whether the operator has opted in via `admin_setSdmPostExecOptIn`.
    pub post_exec_opt_in: bool,
    /// Whether SDM is active per the chain spec at `query_timestamp`.
    pub protocol_active: bool,
    /// AND of the two gates.
    pub effective: bool,
    /// Activation timestamp of the protocol gate (Interop) if scheduled.
    pub activation_time: Option<u64>,
}

#[cfg_attr(not(test), rpc(server, namespace = "admin"))]
#[cfg_attr(test, rpc(server, client, namespace = "admin"))]
pub trait SdmAdminApi {
    /// Toggle local `PostExec` production. Starts disabled on process boot.
    #[method(name = "setSdmPostExecOptIn")]
    fn set_sdm_post_exec_opt_in(&self, enabled: bool) -> RpcResult<()>;

    /// Report the local opt-in flag, the chain-spec gate at `query_timestamp`, and the AND.
    /// If `query_timestamp` is omitted, uses the current wall-clock time.
    #[method(name = "sdmStatus")]
    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus>;
}

/// Concrete `admin_` SDM RPC server holding the shared opt-in flag and the chain spec used
/// to evaluate the protocol gate for status queries.
#[derive(Debug, Clone)]
pub struct OpSdmAdminApi<ChainSpec> {
    opt_in: SdmPostExecOptIn,
    chain_spec: Arc<ChainSpec>,
}

impl<ChainSpec> OpSdmAdminApi<ChainSpec> {
    /// Construct a handler that mutates `opt_in` and reads the protocol gate from `chain_spec`.
    /// The shared `SdmPostExecOptIn` should also be threaded into the payload builder's
    /// `OpBuilderConfig` so writes here take effect on the next produced block.
    pub const fn new(opt_in: SdmPostExecOptIn, chain_spec: Arc<ChainSpec>) -> Self {
        Self { opt_in, chain_spec }
    }
}

impl<ChainSpec> SdmAdminApiServer for OpSdmAdminApi<ChainSpec>
where
    ChainSpec: OpHardforks + Send + Sync + 'static,
{
    fn set_sdm_post_exec_opt_in(&self, enabled: bool) -> RpcResult<()> {
        self.opt_in.set(enabled);
        Ok(())
    }

    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus> {
        let timestamp = query_timestamp.unwrap_or_else(current_unix_timestamp);
        let post_exec_opt_in = self.opt_in.enabled();
        let protocol_active =
            reth_optimism_evm::is_sdm_active_at_timestamp(&self.chain_spec, timestamp);
        let activation_time = match self.chain_spec.op_fork_activation(OpHardfork::Lagoon) {
            ForkCondition::Timestamp(timestamp) => Some(timestamp),
            _ => None,
        };
        let effective = post_exec_opt_in && protocol_active;

        Ok(SdmStatus { post_exec_opt_in, protocol_active, effective, activation_time })
    }
}

fn current_unix_timestamp() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |duration| duration.as_secs())
}
