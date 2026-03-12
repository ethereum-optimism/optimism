//! Transaction creation for batch submission.

use alloy_consensus::{TxEnvelope, TxLegacy};
use alloy_primitives::{Address, Bytes, TxKind, U256};
use kona_protocol::{Frame, DERIVATION_VERSION_0};

/// Creates a calldata transaction containing the given frame, sent to the batch inbox.
pub fn create_calldata_tx(
    frame: Frame,
    batch_inbox: Address,
    _batcher_addr: Address,
) -> TxEnvelope {
    // Encode frame with derivation version prefix
    let mut calldata = vec![DERIVATION_VERSION_0];
    calldata.extend_from_slice(&frame.encode());

    let tx = TxLegacy {
        chain_id: Some(1), // L1 chain ID
        nonce: 0,
        gas_price: 1_000_000_000, // 1 gwei
        gas_limit: 1_000_000,
        to: TxKind::Call(batch_inbox),
        value: U256::ZERO,
        input: Bytes::from(calldata),
    };

    // Sign with a dummy signature - the derivation pipeline only checks
    // the `from` address which we derive from the signature.
    let sig = alloy_primitives::Signature::new(U256::from(1), U256::from(1), false);

    TxEnvelope::Legacy(alloy_consensus::Signed::new_unchecked(tx, sig, Default::default()))
}
