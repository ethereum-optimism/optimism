//! Defines the exact transaction variants that are allowed to be propagated over the eth p2p
//! protocol in op.

use crate::OpTxEnvelope;
use alloy_consensus::{
    Extended, SignableTransaction, Signed, TransactionEnvelope, TxEip7702, TxEnvelope,
    error::ValueError,
    transaction::{TxEip1559, TxEip2930, TxHashRef, TxLegacy},
};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{B256, Signature, TxHash, bytes};
use core::hash::Hash;

/// All possible transactions that can be included in a response to `GetPooledTransactions`.
/// A response to `GetPooledTransactions`. This can include a typed signed transaction, but cannot
/// include a deposit transaction or EIP-4844 transaction.
///
/// The difference between this and the [`OpTxEnvelope`] is that this type does not have the deposit
/// transaction variant, which is not expected to be pooled.
#[derive(Clone, Debug, TransactionEnvelope)]
#[envelope(tx_type_name = OpPooledTxType, serde_cfg(feature = "serde"))]
pub enum OpPooledTransaction {
    /// An untagged [`TxLegacy`].
    #[envelope(ty = 0)]
    Legacy(Signed<TxLegacy>),
    /// A [`TxEip2930`] transaction tagged with type 1.
    #[envelope(ty = 1)]
    Eip2930(Signed<TxEip2930>),
    /// A [`TxEip1559`] transaction tagged with type 2.
    #[envelope(ty = 2)]
    Eip1559(Signed<TxEip1559>),
    /// A [`TxEip7702`] transaction tagged with type 4.
    #[envelope(ty = 4)]
    Eip7702(Signed<TxEip7702>),
}

impl OpPooledTransaction {
    /// Heavy operation that returns the signature hash over rlp encoded transaction. It is only
    /// for signature signing or signer recovery.
    pub fn signature_hash(&self) -> B256 {
        match self {
            Self::Legacy(tx) => tx.signature_hash(),
            Self::Eip2930(tx) => tx.signature_hash(),
            Self::Eip1559(tx) => tx.signature_hash(),
            Self::Eip7702(tx) => tx.signature_hash(),
        }
    }

    /// Reference to transaction hash. Used to identify transaction.
    pub fn hash(&self) -> &TxHash {
        match self {
            Self::Legacy(tx) => tx.hash(),
            Self::Eip2930(tx) => tx.hash(),
            Self::Eip1559(tx) => tx.hash(),
            Self::Eip7702(tx) => tx.hash(),
        }
    }

    /// Returns the signature of the transaction.
    pub const fn signature(&self) -> &Signature {
        match self {
            Self::Legacy(tx) => tx.signature(),
            Self::Eip2930(tx) => tx.signature(),
            Self::Eip1559(tx) => tx.signature(),
            Self::Eip7702(tx) => tx.signature(),
        }
    }

    /// This encodes the transaction _without_ the signature, and is only suitable for creating a
    /// hash intended for signing.
    pub fn encode_for_signing(&self, out: &mut dyn bytes::BufMut) {
        match self {
            Self::Legacy(tx) => tx.tx().encode_for_signing(out),
            Self::Eip2930(tx) => tx.tx().encode_for_signing(out),
            Self::Eip1559(tx) => tx.tx().encode_for_signing(out),
            Self::Eip7702(tx) => tx.tx().encode_for_signing(out),
        }
    }

    /// Converts the transaction into the ethereum [`TxEnvelope`].
    pub fn into_envelope(self) -> TxEnvelope {
        match self {
            Self::Legacy(tx) => tx.into(),
            Self::Eip2930(tx) => tx.into(),
            Self::Eip1559(tx) => tx.into(),
            Self::Eip7702(tx) => tx.into(),
        }
    }

    /// Converts the transaction into the optimism [`OpTxEnvelope`].
    pub fn into_op_envelope(self) -> OpTxEnvelope {
        match self {
            Self::Legacy(tx) => tx.into(),
            Self::Eip2930(tx) => tx.into(),
            Self::Eip1559(tx) => tx.into(),
            Self::Eip7702(tx) => tx.into(),
        }
    }

    /// Returns the [`TxLegacy`] variant if the transaction is a legacy transaction.
    pub const fn as_legacy(&self) -> Option<&TxLegacy> {
        match self {
            Self::Legacy(tx) => Some(tx.tx()),
            _ => None,
        }
    }

    /// Returns the [`TxEip2930`] variant if the transaction is an EIP-2930 transaction.
    pub const fn as_eip2930(&self) -> Option<&TxEip2930> {
        match self {
            Self::Eip2930(tx) => Some(tx.tx()),
            _ => None,
        }
    }

    /// Returns the [`TxEip1559`] variant if the transaction is an EIP-1559 transaction.
    pub const fn as_eip1559(&self) -> Option<&TxEip1559> {
        match self {
            Self::Eip1559(tx) => Some(tx.tx()),
            _ => None,
        }
    }

    /// Returns the [`TxEip7702`] variant if the transaction is an EIP-7702 transaction.
    pub const fn as_eip7702(&self) -> Option<&TxEip7702> {
        match self {
            Self::Eip7702(tx) => Some(tx.tx()),
            _ => None,
        }
    }
}

impl From<Signed<TxLegacy>> for OpPooledTransaction {
    fn from(v: Signed<TxLegacy>) -> Self {
        Self::Legacy(v)
    }
}

impl From<Signed<TxEip2930>> for OpPooledTransaction {
    fn from(v: Signed<TxEip2930>) -> Self {
        Self::Eip2930(v)
    }
}

impl From<Signed<TxEip1559>> for OpPooledTransaction {
    fn from(v: Signed<TxEip1559>) -> Self {
        Self::Eip1559(v)
    }
}

impl From<Signed<TxEip7702>> for OpPooledTransaction {
    fn from(v: Signed<TxEip7702>) -> Self {
        Self::Eip7702(v)
    }
}

impl From<OpPooledTransaction> for alloy_consensus::transaction::PooledTransaction {
    fn from(value: OpPooledTransaction) -> Self {
        match value {
            OpPooledTransaction::Legacy(tx) => tx.into(),
            OpPooledTransaction::Eip2930(tx) => tx.into(),
            OpPooledTransaction::Eip1559(tx) => tx.into(),
            OpPooledTransaction::Eip7702(tx) => tx.into(),
        }
    }
}

impl TxHashRef for OpPooledTransaction {
    fn tx_hash(&self) -> &B256 {
        Self::hash(self)
    }
}

#[cfg(feature = "k256")]
impl alloy_consensus::transaction::SignerRecoverable for OpPooledTransaction {
    fn recover_signer(
        &self,
    ) -> Result<alloy_primitives::Address, alloy_consensus::crypto::RecoveryError> {
        let signature_hash = self.signature_hash();
        alloy_consensus::crypto::secp256k1::recover_signer(self.signature(), signature_hash)
    }

    fn recover_signer_unchecked(
        &self,
    ) -> Result<alloy_primitives::Address, alloy_consensus::crypto::RecoveryError> {
        let signature_hash = self.signature_hash();
        alloy_consensus::crypto::secp256k1::recover_signer_unchecked(
            self.signature(),
            signature_hash,
        )
    }

    fn recover_unchecked_with_buf(
        &self,
        buf: &mut alloc::vec::Vec<u8>,
    ) -> Result<alloy_primitives::Address, alloy_consensus::crypto::RecoveryError> {
        match self {
            Self::Legacy(tx) => {
                alloy_consensus::transaction::SignerRecoverable::recover_unchecked_with_buf(tx, buf)
            }
            Self::Eip2930(tx) => {
                alloy_consensus::transaction::SignerRecoverable::recover_unchecked_with_buf(tx, buf)
            }
            Self::Eip1559(tx) => {
                alloy_consensus::transaction::SignerRecoverable::recover_unchecked_with_buf(tx, buf)
            }
            Self::Eip7702(tx) => {
                alloy_consensus::transaction::SignerRecoverable::recover_unchecked_with_buf(tx, buf)
            }
        }
    }
}

impl From<OpPooledTransaction> for TxEnvelope {
    fn from(tx: OpPooledTransaction) -> Self {
        tx.into_envelope()
    }
}

impl From<OpPooledTransaction> for OpTxEnvelope {
    fn from(tx: OpPooledTransaction) -> Self {
        tx.into_op_envelope()
    }
}

impl TryFrom<OpTxEnvelope> for OpPooledTransaction {
    type Error = ValueError<OpTxEnvelope>;

    fn try_from(value: OpTxEnvelope) -> Result<Self, Self::Error> {
        value.try_into_pooled()
    }
}

impl<Tx> From<OpPooledTransaction> for Extended<OpTxEnvelope, Tx> {
    fn from(tx: OpPooledTransaction) -> Self {
        Self::BuiltIn(tx.into())
    }
}

impl<Tx> TryFrom<Extended<OpTxEnvelope, Tx>> for OpPooledTransaction {
    type Error = ();

    fn try_from(_tx: Extended<OpTxEnvelope, Tx>) -> Result<Self, Self::Error> {
        match _tx {
            Extended::BuiltIn(inner) => inner.try_into().map_err(|_| ()),
            Extended::Other(_tx) => Err(()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Transaction;
    use alloy_primitives::{address, hex};
    use alloy_rlp::Decodable;
    use bytes::Bytes;

    #[test]
    fn invalid_legacy_pooled_decoding_input_too_short() {
        let input_too_short = [
            // this should fail because the payload length is longer than expected
            &hex!("d90b0280808bc5cd028083c5cdfd9e407c56565656")[..],
            // these should fail decoding
            //
            // The `c1` at the beginning is a list header, and the rest is a valid legacy
            // transaction, BUT the payload length of the list header is 1, and the payload is
            // obviously longer than one byte.
            &hex!("c10b02808083c5cd028883c5cdfd9e407c56565656"),
            &hex!("c10b0280808bc5cd028083c5cdfd9e407c56565656"),
            // this one is 19 bytes, and the buf is long enough, but the transaction will not
            // consume that many bytes.
            &hex!("d40b02808083c5cdeb8783c5acfd9e407c5656565656"),
            &hex!("d30102808083c5cd02887dc5cdfd9e64fd9e407c56"),
        ];

        for hex_data in &input_too_short {
            let input_rlp = &mut &hex_data[..];
            let res = OpPooledTransaction::decode(input_rlp);

            assert!(
                res.is_err(),
                "expected err after decoding rlp input: {:x?}",
                Bytes::copy_from_slice(hex_data)
            );

            // this is a legacy tx so we can attempt the same test with decode_enveloped
            let input_rlp = &mut &hex_data[..];
            let res = OpPooledTransaction::decode_2718(input_rlp);

            assert!(
                res.is_err(),
                "expected err after decoding enveloped rlp input: {:x?}",
                Bytes::copy_from_slice(hex_data)
            );
        }
    }

    // <https://holesky.etherscan.io/tx/0x7f60faf8a410a80d95f7ffda301d5ab983545913d3d789615df3346579f6c849>
    #[test]
    fn decode_eip1559_enveloped() {
        let data = hex!(
            "02f903d382426882ba09832dc6c0848674742682ed9694714b6a4ea9b94a8a7d9fd362ed72630688c8898c80b90364492d24749189822d8512430d3f3ff7a2ede675ac08265c08e2c56ff6fdaa66dae1cdbe4a5d1d7809f3e99272d067364e597542ac0c369d69e22a6399c3e9bee5da4b07e3f3fdc34c32c3d88aa2268785f3e3f8086df0934b10ef92cfffc2e7f3d90f5e83302e31382e302d64657600000000000000000000000000000000000000000000569e75fc77c1a856f6daaf9e69d8a9566ca34aa47f9133711ce065a571af0cfd000000000000000000000000e1e210594771824dad216568b91c9cb4ceed361c00000000000000000000000000000000000000000000000000000000000546e00000000000000000000000000000000000000000000000000000000000e4e1c00000000000000000000000000000000000000000000000000000000065d6750c00000000000000000000000000000000000000000000000000000000000f288000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002cf600000000000000000000000000000000000000000000000000000000000000640000000000000000000000000000000000000000000000000000000000000000f1628e56fa6d8c50e5b984a58c0df14de31c7b857ce7ba499945b99252976a93d06dcda6776fc42167fbe71cb59f978f5ef5b12577a90b132d14d9c6efa528076f0161d7bf03643cfc5490ec5084f4a041db7f06c50bd97efa08907ba79ddcac8b890f24d12d8db31abbaaf18985d54f400449ee0559a4452afe53de5853ce090000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000028000000000000000000000000000000000000000000000000000000000000003e800000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000064ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000000000000000000000000000000000000000000000000000c080a01428023fc54a27544abc421d5d017b9a7c5936ad501cbdecd0d9d12d04c1a033a0753104bbf1c87634d6ff3f0ffa0982710612306003eb022363b57994bdef445a"
        );

        let res = OpPooledTransaction::decode_2718(&mut &data[..]).unwrap();
        assert_eq!(res.to(), Some(address!("714b6a4ea9b94a8a7d9fd362ed72630688c8898c")));
    }

    #[test]
    fn legacy_valid_pooled_decoding() {
        // d3 <- payload length, d3 - c0 = 0x13 = 19
        // 0b <- nonce
        // 02 <- gas_price
        // 80 <- gas_limit
        // 80 <- to (Create)
        // 83 c5cdeb <- value
        // 87 83c5acfd9e407c <- input
        // 56 <- v (eip155, so modified with a chain id)
        // 56 <- r
        // 56 <- s
        let data = &hex!("d30b02808083c5cdeb8783c5acfd9e407c565656")[..];

        let input_rlp = &mut &data[..];
        let res = OpPooledTransaction::decode(input_rlp);
        assert!(res.is_ok());
        assert!(input_rlp.is_empty());

        // we can also decode_enveloped
        let res = OpPooledTransaction::decode_2718(&mut &data[..]);
        assert!(res.is_ok());
    }
}
