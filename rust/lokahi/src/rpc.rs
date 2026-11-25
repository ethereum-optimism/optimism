//! The authority RPC the nodes poll.

use crate::state::Verified;
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::VerifierHead;
use kona_rpc::InteropAuthorityApiServer;
use std::sync::{Arc, RwLock};

/// Serves the `interop_*` heads from the verifier's state.
#[derive(Debug)]
pub(crate) struct AuthorityRpc {
    pub state: Arc<RwLock<Verified>>,
}

fn unknown_chain(chain_id: u64) -> ErrorObject<'static> {
    ErrorObject::owned(
        ErrorCode::InvalidParams.code(),
        format!("chain {chain_id} is not verified here"),
        None::<()>,
    )
}

#[async_trait]
impl InteropAuthorityApiServer for AuthorityRpc {
    async fn verified_head(&self, chain_id: u64) -> RpcResult<VerifierHead> {
        self.state
            .read()
            .expect("state lock")
            .verified_head(chain_id)
            .ok_or_else(|| unknown_chain(chain_id))
    }

    async fn finalized_head(&self, chain_id: u64) -> RpcResult<VerifierHead> {
        self.state
            .read()
            .expect("state lock")
            .finalized_head(chain_id)
            .ok_or_else(|| unknown_chain(chain_id))
    }

    async fn denied_blocks(&self, chain_id: u64, from: u64) -> RpcResult<Vec<BlockNumHash>> {
        let state = self.state.read().expect("state lock");
        state.verified_head(chain_id).ok_or_else(|| unknown_chain(chain_id))?;
        let denied = state.denied_blocks(chain_id, from).map_err(|err| {
            ErrorObject::owned(ErrorCode::InternalError.code(), err.to_string(), None::<()>)
        })?;
        Ok(denied.into_iter().map(|record| record.block).collect())
    }
}
