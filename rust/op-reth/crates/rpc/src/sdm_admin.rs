//! Local SDM `PostExec` opt-in admin RPC for op-reth's payload builder.
//!
//! The protocol gate (hardfork activation) is a consensus rule shared by every node. This module
//! adds an orthogonal *operator* gate: even on an SDM-active chain, op-reth's standard payload
//! builder produces `PostExec` txs only when the operator has opted in via
//! `admin_setOperatorSdmOptIn`. Both must be true in order for SDM to be active.
//!
//! State is in-memory and starts disabled on every process boot; persistence is deliberately
//! out of scope.

use alloy_hardforks::ForkCondition;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use reth_optimism_forks::{OpHardfork, OpHardforks};
use reth_optimism_payload_builder::config::OperatorSdmOptIn;
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
    /// Whether the operator has opted in via `admin_setOperatorSdmOptIn`.
    pub operator_sdm_opt_in: bool,
    /// Whether SDM is active per the chain spec at `query_timestamp`.
    pub protocol_active: bool,
    /// AND of the two gates.
    pub effective: bool,
    /// Activation timestamp of the protocol gate if scheduled.
    pub activation_time: Option<u64>,
}

#[cfg_attr(not(test), rpc(server, namespace = "admin"))]
#[cfg_attr(test, rpc(server, client, namespace = "admin"))]
pub trait SdmAdminApi {
    /// Toggle local `PostExec` production. Starts disabled on process boot.
    #[method(name = "setOperatorSdmOptIn")]
    fn set_operator_sdm_opt_in(&self, enabled: bool) -> RpcResult<()>;

    /// Report the local opt-in flag, the chain-spec gate at `query_timestamp`, and the AND.
    /// If `query_timestamp` is omitted, uses the current wall-clock time.
    #[method(name = "sdmStatus")]
    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus>;
}

/// Concrete `admin_` SDM RPC server holding the shared opt-in flag and the chain spec used
/// to evaluate the protocol gate for status queries.
#[derive(Debug, Clone)]
pub struct OpSdmAdminApi<ChainSpec> {
    opt_in: OperatorSdmOptIn,
    chain_spec: Arc<ChainSpec>,
}

impl<ChainSpec> OpSdmAdminApi<ChainSpec> {
    /// Construct a handler that mutates `opt_in` and reads the protocol gate from `chain_spec`.
    /// The shared `OperatorSdmOptIn` should also be threaded into the payload builder's
    /// `OpBuilderConfig` so writes here take effect on the next produced block.
    pub const fn new(opt_in: OperatorSdmOptIn, chain_spec: Arc<ChainSpec>) -> Self {
        Self { opt_in, chain_spec }
    }
}

impl<ChainSpec> SdmAdminApiServer for OpSdmAdminApi<ChainSpec>
where
    ChainSpec: OpHardforks + Send + Sync + 'static,
{
    fn set_operator_sdm_opt_in(&self, enabled: bool) -> RpcResult<()> {
        self.opt_in.set(enabled);
        Ok(())
    }

    fn sdm_status(&self, query_timestamp: Option<u64>) -> RpcResult<SdmStatus> {
        let timestamp = query_timestamp.unwrap_or_else(current_unix_timestamp);
        let operator_sdm_opt_in = self.opt_in.enabled();
        let protocol_active =
            reth_optimism_evm::is_sdm_active_at_timestamp(&self.chain_spec, timestamp);
        let activation_time = match self.chain_spec.op_fork_activation(OpHardfork::Lagoon) {
            ForkCondition::Timestamp(timestamp) => Some(timestamp),
            _ => None,
        };
        let effective = operator_sdm_opt_in && protocol_active;

        Ok(SdmStatus { operator_sdm_opt_in, protocol_active, effective, activation_time })
    }
}

fn current_unix_timestamp() -> u64 {
    SystemTime::now().duration_since(UNIX_EPOCH).map_or(0, |duration| duration.as_secs())
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
    use rstest::rstest;

    // Any timestamp works for the gate checks below: SDM's gating hardfork (Lagoon) is activated
    // at genesis on the active spec, and never on the inactive one.
    const QUERY_TIMESTAMP: u64 = 2_000_000_000;

    fn sdm_active_chain_spec() -> Arc<OpChainSpec> {
        Arc::new(OpChainSpecBuilder::optimism_mainnet().lagoon_activated().build())
    }

    fn sdm_inactive_chain_spec() -> Arc<OpChainSpec> {
        Arc::new(OpChainSpecBuilder::optimism_mainnet().regolith_activated().build())
    }

    fn admin_api(protocol_active: bool) -> OpSdmAdminApi<OpChainSpec> {
        let chain_spec =
            if protocol_active { sdm_active_chain_spec() } else { sdm_inactive_chain_spec() };
        OpSdmAdminApi::new(OperatorSdmOptIn::default(), chain_spec)
    }

    /// The protocol gate (chain spec) and the operator opt-in are independent; SDM is effective
    /// only when both are true.
    #[rstest]
    #[case::neither(false, false, false)]
    #[case::operator_only(false, true, false)]
    #[case::protocol_only(true, false, false)]
    #[case::both(true, true, true)]
    fn sdm_status_effective_is_and_of_both_gates(
        #[case] protocol_active: bool,
        #[case] operator_sdm_opt_in: bool,
        #[case] expected_effective: bool,
    ) {
        let api = admin_api(protocol_active);
        api.set_operator_sdm_opt_in(operator_sdm_opt_in).unwrap();

        let status = api.sdm_status(Some(QUERY_TIMESTAMP)).unwrap();
        assert_eq!(status.operator_sdm_opt_in, operator_sdm_opt_in);
        assert_eq!(status.protocol_active, protocol_active);
        assert_eq!(status.effective, expected_effective);
    }

    /// The operator opt-in starts disabled and can be hot-toggled on -> off -> on at runtime.
    #[test]
    fn set_operator_sdm_opt_in_toggles_on_off_on() {
        let api = admin_api(true);
        let ts = Some(QUERY_TIMESTAMP);

        assert!(!api.sdm_status(ts).unwrap().operator_sdm_opt_in, "boots disabled");
        api.set_operator_sdm_opt_in(true).unwrap();
        assert!(api.sdm_status(ts).unwrap().effective, "enabled on an SDM-active chain");
        api.set_operator_sdm_opt_in(false).unwrap();
        assert!(!api.sdm_status(ts).unwrap().effective, "disabling clears effective");
        api.set_operator_sdm_opt_in(true).unwrap();
        assert!(api.sdm_status(ts).unwrap().effective, "re-enabling restores effective");
    }

    /// The status serializes with camelCase JSON keys.
    #[test]
    fn sdm_status_serializes_with_camel_case_keys() {
        let json = serde_json::to_value(SdmStatus {
            operator_sdm_opt_in: true,
            protocol_active: false,
            effective: false,
            activation_time: Some(123),
        })
        .unwrap();

        assert_eq!(json["operatorSdmOptIn"], serde_json::json!(true));
        assert_eq!(json["protocolActive"], serde_json::json!(false));
        assert_eq!(json["effective"], serde_json::json!(false));
        assert_eq!(json["activationTime"], serde_json::json!(123));
    }
}
