//! The supernode-level admin/test RPC.
//!
//! Each chain answers on its own socket with the method set a single-chain node has; this server
//! answers for the *process*. It is the seam a test harness drives the supernode through, and the
//! only endpoint whose address a harness can learn without knowing the configuration: it may bind
//! port 0 and logs the address it got, which is what makes an out-of-process launch a single
//! handshake rather than N of them.
//!
//! The surface is deliberately small. It reports what the supernode was configured to run; it does
//! not report liveness, because a chain's own RPC answering is the liveness signal a caller
//! actually needs and this server would only be guessing at it. Later phases add the controls that
//! do need process-wide reach — pausing and resuming interop, introspecting backfill — and they
//! belong here for the same reason: they are not questions about one chain.

use crate::{config::ResolvedChain, version};
use anyhow::{Context, Result};
use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    server::{Server, ServerHandle},
};
use serde::{Deserialize, Serialize};
use std::{net::SocketAddr, path::PathBuf, sync::Arc};
use tracing::info;

/// One chain the supernode was configured to host.
///
/// `rpc_addr` is the socket from the configuration rather than one the RPC server reported back:
/// kona binds each chain's server itself and does not hand the address up, so a chain configured
/// on port 0 would be unaddressable. Requiring a concrete port per chain is what keeps this field
/// honest, and [`crate::config`] already rejects two chains sharing one.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct HostedChain {
    /// The L2 chain id.
    pub(crate) chain_id: u64,
    /// The socket this chain's own RPC server listens on.
    pub(crate) rpc_addr: SocketAddr,
    /// Whether this chain is sequenced or validated by this supernode.
    pub(crate) mode: String,
    /// The directory this chain's state lives under.
    pub(crate) datadir: PathBuf,
}

impl From<&ResolvedChain> for HostedChain {
    fn from(chain: &ResolvedChain) -> Self {
        Self {
            chain_id: chain.l2_chain_id,
            rpc_addr: chain.rpc_socket,
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

/// Binds the admin RPC and logs the address it got.
///
/// Returning the handle rather than spawning and forgetting is what lets the caller stop the
/// server when the chains stop: a process that has torn its chains down should not still be
/// answering questions about them.
pub(crate) async fn serve(socket: SocketAddr, chains: &[ResolvedChain]) -> Result<ServerHandle> {
    let server = Server::builder()
        .build(socket)
        .await
        .with_context(|| format!("failed to bind the admin rpc to {socket}"))?;

    // The bound address, not the requested one: a harness that asked for port 0 learns its port
    // from this line, and this is the line an out-of-process launch waits for.
    let addr = server.local_addr().context("the admin rpc server has no local address")?;
    info!(target: "lokahi", %addr, "Admin RPC server bound to address");

    let chains: Arc<[HostedChain]> = chains.iter().map(HostedChain::from).collect();
    Ok(server.start(AdminRpc { chains }.into_rpc()))
}
