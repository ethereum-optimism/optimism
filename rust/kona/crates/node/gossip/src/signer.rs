//! Signature verification with grace for the previous unsafe block signer.

use crate::{BlockInvalidError, Metrics, payload::SignedGossipPayload};
use alloy_primitives::Address;
use std::time::Duration;
use tokio::time::Instant;

/// Maximum time to accept the previous signer, matching op-node.
pub const SIGNER_GRACE_PERIOD: Duration = Duration::from_secs(3 * 60 * 60);

/// Verifies gossip signatures against the current signer and its immediate predecessor.
/// Owned by the block handler; a verified current signature ends grace early.
/// Clearing the current signer disables acceptance; a subsequent nonzero signer does not
/// restore grace for the signer that preceded the zero address.
#[derive(Debug, Clone)]
pub struct UnsafeBlockSigner {
    current: Address,
    previous: Address,
    changed_at: Instant,
}

impl UnsafeBlockSigner {
    /// Initializes the current signer without a previous key or grace period.
    pub fn new(current: Address) -> Self {
        Self { current, previous: Address::ZERO, changed_at: Instant::now() }
    }

    /// Observes a signer update. Re-reading the same address does not restart grace.
    pub fn new_signer(&mut self, current: Address) {
        if self.current == current {
            return;
        }
        self.previous = self.current;
        self.current = current;
        self.changed_at = Instant::now();
        info!(
            target: "gossip",
            previous = %self.previous,
            current = %self.current,
            grace_period = ?SIGNER_GRACE_PERIOD,
            "Unsafe block signer rotated"
        );
    }

    /// Authenticates the exact envelope bytes and applies the rotation grace period.
    pub(crate) fn accepts(
        &mut self,
        payload: &SignedGossipPayload,
        chain_id: u64,
    ) -> Result<(), BlockInvalidError> {
        let message = payload.payload_hash().signature_message(chain_id);
        let received = payload.recover_signer(&message).map_err(|_| {
            metrics::counter!(Metrics::BLOCK_VALIDATION_FAILED, "reason" => "invalid_signature")
                .increment(1);
            BlockInvalidError::Signature
        })?;
        if self.current != Address::ZERO {
            if received == self.current {
                if self.previous != Address::ZERO {
                    info!(
                        target: "gossip",
                        previous = %self.previous,
                        current = %self.current,
                        "Current unsafe block signer confirmed, ending grace for previous signer"
                    );
                    self.previous = Address::ZERO;
                }
                return Ok(());
            }
            if self.previous != Address::ZERO &&
                received == self.previous &&
                self.changed_at.elapsed() <= SIGNER_GRACE_PERIOD
            {
                info!(
                    target: "gossip",
                    previous = %self.previous,
                    current = %self.current,
                    hash = ?payload.payload_hash(),
                    "Accepted block from previous unsafe block signer during grace period"
                );
                return Ok(());
            }
        }
        metrics::counter!(Metrics::BLOCK_VALIDATION_FAILED, "reason" => "invalid_signer")
            .increment(1);
        Err(BlockInvalidError::Signer { expected: self.current, received })
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use crate::payload::GossipPayloadVersion;
    use alloy_primitives::Signature;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use k256::ecdsa::SigningKey;
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;

    const CHAIN_ID: u64 = 10;

    fn key(byte: u8) -> SigningKey {
        SigningKey::from_bytes(&[byte; 32].into()).unwrap()
    }

    pub(crate) fn address(byte: u8) -> Address {
        Address::from_public_key(key(byte).verifying_key())
    }

    pub(crate) fn payload(byte: u8) -> SignedGossipPayload {
        let envelope = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1::from_block_slow(
            &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
        ));
        let message = envelope.payload_hash().signature_message(CHAIN_ID);
        let signature: Signature =
            key(byte).sign_prehash_recoverable(message.as_slice()).unwrap().into();
        SignedGossipPayload::from_envelope(&envelope, signature, GossipPayloadVersion::V1).unwrap()
    }

    #[test]
    fn initial_signer_has_no_grace() {
        let mut signer = UnsafeBlockSigner::new(Address::ZERO);
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        signer.new_signer(address(1));
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
    }

    #[tokio::test(start_paused = true)]
    async fn previous_signer_expires_three_hours_after_update_without_gossip() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        tokio::time::advance(Duration::from_secs(25 * 60)).await;
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
        tokio::time::advance(SIGNER_GRACE_PERIOD - Duration::from_secs(25 * 60)).await;
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
        tokio::time::advance(Duration::from_nanos(1)).await;
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
    }

    #[test]
    fn current_signature_ends_grace() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
    }

    #[test]
    fn invalid_signature_does_not_end_grace() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        let mut bytes =
            snap::raw::Decoder::new().decompress_vec(&payload(2).encode_snappy().unwrap()).unwrap();
        bytes[..64].fill(0);
        let invalid = SignedGossipPayload::decode_snappy(
            &snap::raw::Encoder::new().compress_vec(&bytes).unwrap(),
        )
        .unwrap();
        assert!(matches!(signer.accepts(&invalid, CHAIN_ID), Err(BlockInvalidError::Signature)));
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
    }

    #[test]
    fn unrelated_signer_or_wrong_chain_does_not_end_grace() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        assert!(signer.accepts(&payload(3), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(2), CHAIN_ID + 1).is_err());
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
    }

    #[tokio::test(start_paused = true)]
    async fn unchanged_update_does_not_extend_or_reopen_grace() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        tokio::time::advance(SIGNER_GRACE_PERIOD).await;
        signer.new_signer(address(2));
        tokio::time::advance(Duration::from_nanos(1)).await;
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
        signer.new_signer(address(2));
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
    }

    #[test]
    fn consecutive_updates_keep_only_the_immediately_previous_signer() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        signer.new_signer(address(3));
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(3), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_err());
    }

    #[test]
    fn clearing_the_signer_disables_grace() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(Address::ZERO);
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        signer.new_signer(address(2));
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_err());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
    }

    #[test]
    fn reorg_back_to_previous_signer_starts_a_new_rotation() {
        let mut signer = UnsafeBlockSigner::new(address(1));
        signer.new_signer(address(2));
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
        signer.new_signer(address(1));
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(1), CHAIN_ID).is_ok());
        assert!(signer.accepts(&payload(2), CHAIN_ID).is_err());
    }
}
