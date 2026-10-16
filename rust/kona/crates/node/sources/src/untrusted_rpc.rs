//! Reads blocks from an RPC the node does not trust.

use alloy_consensus::{Header, Sealable, Sealed};
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::Block;
use alloy_transport::{RpcError, TransportError};
use std::time::Duration;
use thiserror::Error;
use tracing::warn;

/// Reads block headers from an RPC, and checks that each is the block asked for.
///
/// An RPC reports each block's hash beside its header, and could report any hash. Each header is
/// hashed here instead, and is an error if its hash differs from the one the RPC reported, or if it
/// is not the block asked for. Callers get the header sealed with the hash computed here.
///
/// It reads through a [`RootProvider`], so it shares the provider's HTTP client.
#[derive(Debug, Clone)]
pub struct UntrustedRpc {
    provider: RootProvider,
}

/// A block an [`UntrustedRpc`] read could not be trusted, or could not be read.
#[derive(Debug, Error)]
pub enum UntrustedRpcError {
    /// The RPC request failed.
    #[error(transparent)]
    Transport(#[from] TransportError),
    /// The RPC reported a hash other than the hash of the header it returned.
    #[error("the RPC reported hash {reported} for a header that hashes to {computed}")]
    ReportedHash {
        /// The hash the RPC reported.
        reported: B256,
        /// The hash of the header the RPC returned.
        computed: B256,
    },
    /// The RPC returned a block other than the one with the requested hash.
    #[error("asked for block {requested}, the RPC returned block {returned}")]
    WrongHash {
        /// The hash asked for.
        requested: B256,
        /// The hash of the header the RPC returned.
        returned: B256,
    },
    /// The RPC returned a block other than the one at the requested height.
    #[error("asked for block {requested}, the RPC returned block {returned}")]
    WrongNumber {
        /// The number asked for.
        requested: u64,
        /// The number of the header the RPC returned.
        returned: u64,
    },
}

impl UntrustedRpc {
    /// Creates a reader that reads through `provider`.
    pub const fn new(provider: RootProvider) -> Self {
        Self { provider }
    }

    /// Returns a reader that retries every failed read, after `delay`, until it succeeds.
    pub const fn retrying(self, delay: Duration) -> RetryingRpc {
        RetryingRpc { rpc: self, delay }
    }

    /// Reads the header of the block with the given number or tag, or `None` if the RPC has no
    /// such block.
    pub async fn header_by_number(
        &self,
        number: BlockNumberOrTag,
    ) -> Result<Option<Sealed<Header>>, UntrustedRpcError> {
        let block = match self.provider.get_block_by_number(number).await {
            Ok(block) => block,
            // Geth answers with an error, rather than no block, until it has a safe or finalized
            // block.
            Err(RpcError::ErrorResp(error))
                if matches!(
                    (number, error.message.as_ref()),
                    (BlockNumberOrTag::Safe, "safe block not found") |
                        (BlockNumberOrTag::Finalized, "finalized block not found")
                ) =>
            {
                None
            }
            Err(error) => return Err(error.into()),
        };
        let Some(header) = checked(block)? else { return Ok(None) };
        match number {
            BlockNumberOrTag::Number(requested) if header.number != requested => {
                Err(UntrustedRpcError::WrongNumber { requested, returned: header.number })
            }
            _ => Ok(Some(header)),
        }
    }

    /// Reads the header of the block with the given hash, or `None` if the RPC has no such block.
    pub async fn header_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<Sealed<Header>>, UntrustedRpcError> {
        let Some(header) = checked(self.provider.get_block_by_hash(hash).await?)? else {
            return Ok(None);
        };
        if header.hash() != hash {
            return Err(UntrustedRpcError::WrongHash { requested: hash, returned: header.hash() });
        }
        Ok(Some(header))
    }
}

/// Seals the header of an RPC block, checking that it hashes to the hash the RPC reported.
fn checked(block: Option<Block>) -> Result<Option<Sealed<Header>>, UntrustedRpcError> {
    let Some(block) = block else { return Ok(None) };
    let reported = block.header.hash;
    let header = block.header.inner.seal_slow();
    if header.hash() != reported {
        return Err(UntrustedRpcError::ReportedHash { reported, computed: header.hash() });
    }
    Ok(Some(header))
}

/// An [`UntrustedRpc`] that retries every failed read, after a fixed delay, until it succeeds.
///
/// A read never fails: it waits for as long as the RPC is unreachable or returns blocks that
/// cannot be trusted. `None`, the RPC having no such block, is an answer, and is not retried.
#[derive(Debug, Clone)]
pub struct RetryingRpc {
    rpc: UntrustedRpc,
    delay: Duration,
}

impl RetryingRpc {
    /// Returns the reader without retries.
    pub const fn untrusted(&self) -> &UntrustedRpc {
        &self.rpc
    }

    /// Reads the header of the block with the given number or tag, or `None` if the RPC has no
    /// such block.
    pub async fn header_by_number(&self, number: BlockNumberOrTag) -> Option<Sealed<Header>> {
        self.retry(|| self.rpc.header_by_number(number)).await
    }

    /// Reads the header of the block with the given hash, or `None` if the RPC has no such block.
    pub async fn header_by_hash(&self, hash: B256) -> Option<Sealed<Header>> {
        self.retry(|| self.rpc.header_by_hash(hash)).await
    }

    /// Reads the header of block `number`, a block the RPC must have, such as one before the
    /// finalized block. An RPC without it, such as one behind the rest of a load-balanced pool, is
    /// retried too.
    pub async fn header_at(&self, number: u64) -> Sealed<Header> {
        loop {
            if let Some(header) = self.header_by_number(number.into()).await {
                return header;
            }
            warn!(target: "untrusted_rpc", number, "The RPC has no block it must have; retrying");
            tokio::time::sleep(self.delay).await;
        }
    }

    async fn retry<T, F: Future<Output = Result<T, UntrustedRpcError>>>(
        &self,
        read: impl Fn() -> F,
    ) -> T {
        loop {
            match read().await {
                Ok(value) => return value,
                Err(error) => {
                    warn!(target: "untrusted_rpc", %error, "Failed to read a block; retrying");
                    tokio::time::sleep(self.delay).await;
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::Asserter;

    /// Returns an RPC block with the given header, reporting the header's hash.
    fn rpc_block(header: Header) -> Block {
        let mut block: Block = Block::default();
        block.header.hash = header.hash_slow();
        block.header.inner = header;
        block
    }

    fn header(number: u64) -> Header {
        Header { number, timestamp: number * 12, ..Default::default() }
    }

    fn rpc(responses: &Asserter) -> UntrustedRpc {
        UntrustedRpc::new(RootProvider::new(RpcClient::mocked(responses.clone())))
    }

    #[tokio::test]
    async fn returns_headers_sealed_with_their_computed_hash() {
        let responses = Asserter::new();
        let rpc = rpc(&responses);
        let expected = header(7).seal_slow();
        responses.push_success(&rpc_block(header(7)));
        assert_eq!(rpc.header_by_number(7.into()).await.unwrap(), Some(expected.clone()));
        responses.push_success(&rpc_block(header(7)));
        assert_eq!(rpc.header_by_hash(expected.hash()).await.unwrap(), Some(expected.clone()));
        responses.push_success(&rpc_block(header(7)));
        assert_eq!(rpc.header_by_number(BlockNumberOrTag::Latest).await.unwrap(), Some(expected));
        responses.push_success(&Option::<Block>::None);
        assert_eq!(rpc.header_by_hash(B256::ZERO).await.unwrap(), None);
        // Geth's answer before it finalizes a block.
        responses.push_failure_msg("finalized block not found");
        assert_eq!(rpc.header_by_number(BlockNumberOrTag::Finalized).await.unwrap(), None);
        responses.push_failure_msg("finalized block not found");
        assert!(rpc.header_by_number(BlockNumberOrTag::Latest).await.is_err());
    }

    #[tokio::test]
    async fn rejects_blocks_other_than_the_one_reported_or_asked_for() {
        let responses = Asserter::new();
        let rpc = rpc(&responses);

        // A hash that is not the header's.
        let mut block = rpc_block(header(7));
        block.header.hash = B256::repeat_byte(1);
        responses.push_success(&block);
        assert!(matches!(
            rpc.header_by_number(7.into()).await,
            Err(UntrustedRpcError::ReportedHash { .. })
        ));

        // A block other than the one asked for, reported with its own hash.
        responses.push_success(&rpc_block(header(8)));
        assert!(matches!(
            rpc.header_by_number(7.into()).await,
            Err(UntrustedRpcError::WrongNumber { requested: 7, returned: 8 })
        ));
        responses.push_success(&rpc_block(header(8)));
        assert!(matches!(
            rpc.header_by_hash(header(7).hash_slow()).await,
            Err(UntrustedRpcError::WrongHash { .. })
        ));

        responses.push_failure_msg("L1 unavailable");
        assert!(matches!(
            rpc.header_by_hash(B256::ZERO).await,
            Err(UntrustedRpcError::Transport(_))
        ));
    }

    #[tokio::test(start_paused = true)]
    async fn a_retrying_reader_retries_failures_but_not_missing_blocks() {
        let responses = Asserter::new();
        let rpc = rpc(&responses).retrying(Duration::from_secs(1));
        let mut lying = rpc_block(header(7));
        lying.header.hash = B256::repeat_byte(1);
        responses.push_failure_msg("L1 unavailable");
        responses.push_success(&lying);
        responses.push_success(&rpc_block(header(7)));
        assert_eq!(rpc.header_by_number(7.into()).await, Some(header(7).seal_slow()));

        responses.push_success(&Option::<Block>::None);
        assert_eq!(rpc.header_by_number(8.into()).await, None);

        // A block the RPC must have is retried until it has it.
        responses.push_success(&Option::<Block>::None);
        responses.push_success(&rpc_block(header(8)));
        assert_eq!(rpc.header_at(8).await, header(8).seal_slow());
    }
}
