//! The supernode-level admin/test RPC: the namespaces served at `/`.
//!
//! Each chain answers under its own route with the method set a single-chain node has; these
//! methods answer for the *process*, so they are the ones served at the root of the supernode's
//! socket. It is the seam a test harness drives the supernode through, and the surface a harness
//! reaches first: the socket may be port 0 and the address it got is logged, which is what makes
//! an out-of-process launch a single handshake rather than N of them.
//!
//! The surface is deliberately small. It reports what the supernode was configured to run; it does
//! not report liveness, because a chain's own RPC answering is the liveness signal a caller
//! actually needs and this server would only be guessing at it. The controls that do need
//! process-wide reach — pausing and resuming interop, introspecting backfill — are served here for
//! the same reason: they are not questions about one chain. See
//! [`crate::interop::InteropTestHandle`].
//!
//! The supernode *query* API — `supernode_syncStatus` and `superroot_atTimestamp` — is served at
//! the same root, in its own two namespaces. It belongs here for the same reason as everything
//! else at the root: both methods are statements about the whole chain set, and neither has a
//! per-chain answer to be served from a chain's own route. It is also what makes lokahi reachable
//! by the existing consumers, which dial one supernode endpoint and call two methods on it.

use crate::{config::ResolvedChain, interop::InteropTestHandle, query::QueryHandle, version};
use anyhow::{Context, Result};
use jsonrpsee::{RpcModule, core::RpcResult, proc_macros::rpc};
use serde::{Deserialize, Serialize};
use std::{path::PathBuf, sync::Arc};

/// One chain the supernode was configured to host.
///
/// `rpc_path` rather than an address: every chain is served on the one socket the caller is
/// already talking to, and what distinguishes them is the path. So the answer to *where is chain
/// N* is a path to append, and a caller that has this list needs nothing else to reach any chain
/// in it.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct HostedChain {
    /// The L2 chain id.
    pub(crate) chain_id: u64,
    /// The path on the supernode's socket this chain's own RPC answers under.
    pub(crate) rpc_path: String,
    /// Whether this chain is sequenced or validated by this supernode.
    pub(crate) mode: String,
    /// The directory this chain's state lives under.
    pub(crate) datadir: PathBuf,
}

impl From<&ResolvedChain> for HostedChain {
    fn from(chain: &ResolvedChain) -> Self {
        Self {
            chain_id: chain.l2_chain_id,
            rpc_path: format!("/{}", chain.l2_chain_id),
            mode: chain.mode().to_string(),
            datadir: chain.datadir.clone(),
        }
    }
}

/// The `lokahi` namespace: questions about the supernode rather than about one of its chains.
#[rpc(server, namespace = "lokahi")]
pub(crate) trait LokahiAdminApi {
    /// The chains this supernode was configured to host, in configuration order.
    #[method(name = "chains")]
    async fn chains(&self) -> RpcResult<Vec<HostedChain>>;

    /// The supernode's version.
    #[method(name = "version")]
    async fn version(&self) -> RpcResult<String>;
}

/// The admin RPC's answers: the configuration, resolved, and nothing that can drift from it.
#[derive(Debug)]
struct AdminRpc {
    /// The chains, as configured.
    chains: Arc<[HostedChain]>,
}

#[async_trait::async_trait]
impl LokahiAdminApiServer for AdminRpc {
    async fn chains(&self) -> RpcResult<Vec<HostedChain>> {
        Ok(self.chains.to_vec())
    }

    async fn version(&self) -> RpcResult<String> {
        Ok(version::short_version().to_string())
    }
}

/// Builds the module set the supernode's own namespaces are served from.
///
/// Built rather than bound: the socket is the supernode's one socket, opened by
/// [`crate::rpc::SupernodeRpc`], and these methods are mounted at its root. That is also why this
/// can be called — and the address logged — before any chain exists.
///
/// `queries` is handed over empty and filled once the chains are composed. A query arriving in
/// that window is answered with an error saying the supernode is starting.
pub(crate) fn module(
    chains: &[ResolvedChain],
    queries: QueryHandle,
    interop_test: InteropTestHandle,
) -> Result<RpcModule<()>> {
    let chains: Arc<[HostedChain]> = chains.iter().map(HostedChain::from).collect();
    let mut module = RpcModule::new(());
    module
        .merge(AdminRpc { chains }.into_rpc())
        .context("failed to register the supernode admin API")?;
    module
        .merge(queries.into_rpc_module().context("failed to build the supernode query API")?)
        .context("failed to register the supernode query API")?;
    module
        .merge(
            interop_test
                .into_rpc_module()
                .context("failed to build the interop test-control API")?,
        )
        .context("failed to register the interop test-control API")?;
    Ok(module)
}
