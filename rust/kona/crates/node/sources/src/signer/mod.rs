//! Signer utilities for the Kona node.
//!
//! We currently support two types of block signers:
//!
//! 1. A local block signer that is used to sign blocks with a locally available private key.
//! 2. A remote block signer that is used to sign blocks with a remote private key.

use alloy_primitives::{Address, ChainId};
use alloy_signer::{Signature, SignerSync};
use derive_more::From;
use op_alloy_rpc_types_engine::PayloadHash;
use std::fmt::Debug;

mod remote;
pub use remote::{
    CertificateError, ClientCert, RemoteSigner, RemoteSignerError, RemoteSignerHandler,
    RemoteSignerStartError,
};

/// A builder for a block signer.
#[derive(Debug, Clone, From)]
pub enum BlockSigner {
    /// A local block signer that is used to sign blocks with a locally available private key.
    Local(#[from] alloy_signer_local::PrivateKeySigner),
    /// A remote block signer that is used to sign blocks with a remote private key.
    Remote(#[from] RemoteSigner),
}

/// A handler for a block signer.
#[derive(Debug)]
pub enum BlockSignerHandler {
    /// A local block signer that is used to sign blocks with a locally available private key.
    Local(alloy_signer_local::PrivateKeySigner),
    /// A remote block signer that is used to sign blocks with a remote private key.
    Remote(RemoteSignerHandler),
}

/// Errors that can occur when starting a block signer.
#[derive(Debug, thiserror::Error)]
pub enum BlockSignerStartError {
    /// An error that can occur when signing a block with a local signer.
    #[error(transparent)]
    Local(#[from] alloy_signer::Error),
    /// An error that can occur when signing a block with a remote signer.
    #[error(transparent)]
    Remote(#[from] RemoteSignerStartError),
}

/// Errors that can occur when signing a block.
#[derive(Debug, thiserror::Error)]
pub enum BlockSignerError {
    /// An error that can occur when signing a block with a local signer.
    #[error(transparent)]
    Local(#[from] alloy_signer::Error),
    /// An error that can occur when signing a block with a remote signer.
    #[error(transparent)]
    Remote(#[from] RemoteSignerError),
}

impl BlockSigner {
    /// Starts a block signer.
    pub async fn start(self) -> Result<BlockSignerHandler, BlockSignerStartError> {
        match self {
            Self::Local(signer) => Ok(BlockSignerHandler::Local(signer)),
            Self::Remote(signer) => Ok(BlockSignerHandler::Remote(signer.start().await?)),
        }
    }
}

impl BlockSignerHandler {
    /// Signs a payload with the signer.
    pub async fn sign_block(
        &self,
        payload_hash: PayloadHash,
        chain_id: ChainId,
        sender_address: Address,
    ) -> Result<Signature, BlockSignerError> {
        let signature = match self {
            Self::Local(signer) => {
                signer.sign_hash_sync(&payload_hash.signature_message(chain_id))?
            }
            Self::Remote(signer) => {
                signer.sign_block_v1(payload_hash, chain_id, sender_address).await?
            }
        };

        Ok(signature)
    }
}
