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

/// The error code of a head asked for before verification caught up with the sources. The error
/// data is the timestamp being verified.
const NOT_CAUGHT_UP: i32 = -32001;

/// Serves the `interop_*` heads from the verifier's state.
///
/// The heads are answered only once verification has caught up with the sources' local-safe
/// heads since the start: before that, a store that is empty or behind would answer heads far
/// below the ones the nodes were told before, and the nodes would drop their labels to them. An
/// error makes the nodes hold their labels instead. The denied blocks are answered at once: every
/// denial is in the store before it is answered.
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

impl AuthorityRpc {
    /// The head `head` reads from the state, once verification has caught up.
    fn head(
        &self,
        chain_id: u64,
        head: impl FnOnce(&Verified, u64) -> Option<VerifierHead>,
    ) -> RpcResult<VerifierHead> {
        let state = self.state.read().expect("state lock");
        let head = head(&state, chain_id).ok_or_else(|| unknown_chain(chain_id))?;
        if !state.is_caught_up() {
            let next_timestamp = state.next_timestamp();
            return Err(ErrorObject::owned(
                NOT_CAUGHT_UP,
                format!("verification has not caught up with the sources yet, at {next_timestamp}"),
                Some(next_timestamp),
            ));
        }
        Ok(head)
    }
}

#[async_trait]
impl InteropAuthorityApiServer for AuthorityRpc {
    async fn verified_head(&self, chain_id: u64) -> RpcResult<VerifierHead> {
        self.head(chain_id, Verified::verified_head)
    }

    async fn finalized_head(&self, chain_id: u64) -> RpcResult<VerifierHead> {
        self.head(chain_id, Verified::finalized_head)
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::tests::open_state;
    use std::collections::BTreeMap;

    const CHAIN: u64 = 10;

    /// Until verification caught up the heads are an error, which makes the nodes hold their
    /// labels, while the denied blocks are answered from the start.
    #[tokio::test]
    async fn heads_wait_for_the_catch_up_and_denials_do_not() {
        let dir = tempfile::tempdir().unwrap();
        let state = open_state(dir.path(), 100, BTreeMap::from([(CHAIN, 100)]));
        let rpc = AuthorityRpc { state: Arc::new(RwLock::new(state)) };

        let err = rpc.verified_head(CHAIN).await.unwrap_err();
        assert_eq!(err.code(), NOT_CAUGHT_UP);
        assert_eq!(err.data().map(|data| data.get()), Some("100"), "the timestamp being verified");
        assert_eq!(rpc.finalized_head(CHAIN).await.unwrap_err().code(), NOT_CAUGHT_UP);
        assert_eq!(rpc.denied_blocks(CHAIN, 0).await.unwrap(), vec![]);
        assert_eq!(
            rpc.verified_head(CHAIN + 1).await.unwrap_err().code(),
            ErrorCode::InvalidParams.code()
        );

        rpc.state.write().unwrap().set_caught_up();
        assert_eq!(
            rpc.verified_head(CHAIN).await.unwrap(),
            VerifierHead::Anchor { timestamp: 100 }
        );
    }
}
