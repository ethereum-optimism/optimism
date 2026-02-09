use crate::{
    OpPooledTransaction, TxDeposit,
    transaction::{OpDepositInfo, OpTransactionInfo},
};
use alloy_consensus::{
    EthereumTxEnvelope, Extended, Sealable, Sealed, SignableTransaction, Signed,
    TransactionEnvelope, TxEip1559, TxEip2930, TxEip7702, TxEnvelope, TxLegacy,
    error::ValueError,
    transaction::{TransactionInfo, TxHashRef},
};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{B256, Bytes, Signature, TxHash};

/// The Ethereum [EIP-2718] Transaction Envelope, modified for OP Stack chains.
///
/// # Note:
///
/// This enum distinguishes between tagged and untagged legacy transactions, as
/// the in-protocol merkle tree may commit to EITHER 0-prefixed or raw.
/// Therefore we must ensure that encoding returns the precise byte-array that
/// was decoded, preserving the presence or absence of the `TransactionType`
/// flag.
///
/// [EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718
#[derive(Debug, Clone, TransactionEnvelope)]
#[envelope(tx_type_name = OpTxType, typed = OpTypedTransaction, serde_cfg(feature = "serde"))]
pub enum OpTxEnvelope {
    /// An untagged [`TxLegacy`].
    #[envelope(ty = 0)]
    Legacy(Signed<TxLegacy>),
    /// A [`TxEip2930`] tagged with type 1.
    #[envelope(ty = 1)]
    Eip2930(Signed<TxEip2930>),
    /// A [`TxEip1559`] tagged with type 2.
    #[envelope(ty = 2)]
    Eip1559(Signed<TxEip1559>),
    /// A [`TxEip7702`] tagged with type 4.
    #[envelope(ty = 4)]
    Eip7702(Signed<TxEip7702>),
    /// A [`TxDeposit`] tagged with type 0x7E.
    #[envelope(ty = 126)]
    #[serde(serialize_with = "crate::serde_deposit_tx_rpc")]
    Deposit(Sealed<TxDeposit>),
}

/// Represents an Optimism transaction envelope.
///
/// Compared to Ethereum it can tell whether the transaction is a deposit.
pub trait OpTransaction {
    /// Returns `true` if the transaction is a deposit.
    fn is_deposit(&self) -> bool;

    /// Returns `Some` if the transaction is a deposit.
    fn as_deposit(&self) -> Option<&Sealed<TxDeposit>>;
}

impl OpTransaction for OpTxEnvelope {
    fn is_deposit(&self) -> bool {
        self.is_deposit()
    }

    fn as_deposit(&self) -> Option<&Sealed<TxDeposit>> {
        self.as_deposit()
    }
}

impl<B, T> OpTransaction for Extended<B, T>
where
    B: OpTransaction,
    T: OpTransaction,
{
    fn is_deposit(&self) -> bool {
        match self {
            Self::BuiltIn(b) => b.is_deposit(),
            Self::Other(t) => t.is_deposit(),
        }
    }

    fn as_deposit(&self) -> Option<&Sealed<TxDeposit>> {
        match self {
            Self::BuiltIn(b) => b.as_deposit(),
            Self::Other(t) => t.as_deposit(),
        }
    }
}

impl AsRef<Self> for OpTxEnvelope {
    fn as_ref(&self) -> &Self {
        self
    }
}

impl From<Signed<TxLegacy>> for OpTxEnvelope {
    fn from(v: Signed<TxLegacy>) -> Self {
        Self::Legacy(v)
    }
}

impl From<Signed<TxEip2930>> for OpTxEnvelope {
    fn from(v: Signed<TxEip2930>) -> Self {
        Self::Eip2930(v)
    }
}

impl From<Signed<TxEip1559>> for OpTxEnvelope {
    fn from(v: Signed<TxEip1559>) -> Self {
        Self::Eip1559(v)
    }
}

impl From<Signed<TxEip7702>> for OpTxEnvelope {
    fn from(v: Signed<TxEip7702>) -> Self {
        Self::Eip7702(v)
    }
}

impl From<TxDeposit> for OpTxEnvelope {
    fn from(v: TxDeposit) -> Self {
        v.seal_slow().into()
    }
}

impl From<Signed<OpTypedTransaction>> for OpTxEnvelope {
    fn from(value: Signed<OpTypedTransaction>) -> Self {
        let (tx, sig, hash) = value.into_parts();
        match tx {
            OpTypedTransaction::Legacy(tx_legacy) => {
                let tx = Signed::new_unchecked(tx_legacy, sig, hash);
                Self::Legacy(tx)
            }
            OpTypedTransaction::Eip2930(tx_eip2930) => {
                let tx = Signed::new_unchecked(tx_eip2930, sig, hash);
                Self::Eip2930(tx)
            }
            OpTypedTransaction::Eip1559(tx_eip1559) => {
                let tx = Signed::new_unchecked(tx_eip1559, sig, hash);
                Self::Eip1559(tx)
            }
            OpTypedTransaction::Eip7702(tx_eip7702) => {
                let tx = Signed::new_unchecked(tx_eip7702, sig, hash);
                Self::Eip7702(tx)
            }
            OpTypedTransaction::Deposit(tx) => Self::Deposit(Sealed::new_unchecked(tx, hash)),
        }
    }
}

impl From<(OpTypedTransaction, Signature)> for OpTxEnvelope {
    fn from(value: (OpTypedTransaction, Signature)) -> Self {
        Self::new_unhashed(value.0, value.1)
    }
}

impl From<Sealed<TxDeposit>> for OpTxEnvelope {
    fn from(v: Sealed<TxDeposit>) -> Self {
        Self::Deposit(v)
    }
}

impl<Tx> From<OpTxEnvelope> for Extended<OpTxEnvelope, Tx> {
    fn from(value: OpTxEnvelope) -> Self {
        Self::BuiltIn(value)
    }
}

impl<T> TryFrom<EthereumTxEnvelope<T>> for OpTxEnvelope {
    type Error = EthereumTxEnvelope<T>;

    fn try_from(value: EthereumTxEnvelope<T>) -> Result<Self, Self::Error> {
        Self::try_from_eth_envelope(value)
    }
}

impl TryFrom<OpTxEnvelope> for TxEnvelope {
    type Error = ValueError<OpTxEnvelope>;

    fn try_from(value: OpTxEnvelope) -> Result<Self, Self::Error> {
        value.try_into_eth_envelope()
    }
}

#[cfg(feature = "alloy-compat")]
impl From<OpTxEnvelope> for alloy_rpc_types_eth::TransactionRequest {
    fn from(value: OpTxEnvelope) -> Self {
        match value {
            OpTxEnvelope::Eip2930(tx) => tx.into_parts().0.into(),
            OpTxEnvelope::Eip1559(tx) => tx.into_parts().0.into(),
            OpTxEnvelope::Eip7702(tx) => tx.into_parts().0.into(),
            OpTxEnvelope::Deposit(tx) => tx.into_inner().into(),
            OpTxEnvelope::Legacy(tx) => tx.into_parts().0.into(),
        }
    }
}

impl OpTxEnvelope {
    /// Creates a new enveloped transaction from the given transaction, signature and hash.
    ///
    /// Caution: This assumes the given hash is the correct transaction hash.
    pub fn new_unchecked(
        transaction: OpTypedTransaction,
        signature: Signature,
        hash: B256,
    ) -> Self {
        Signed::new_unchecked(transaction, signature, hash).into()
    }

    /// Creates a new signed transaction from the given typed transaction and signature without the
    /// hash.
    ///
    /// Note: this only calculates the hash on the first [`OpTxEnvelope::hash`] call.
    pub fn new_unhashed(transaction: OpTypedTransaction, signature: Signature) -> Self {
        transaction.into_signed(signature).into()
    }

    /// Returns true if the transaction is a legacy transaction.
    #[inline]
    pub const fn is_legacy(&self) -> bool {
        matches!(self, Self::Legacy(_))
    }

    /// Returns true if the transaction is an EIP-2930 transaction.
    #[inline]
    pub const fn is_eip2930(&self) -> bool {
        matches!(self, Self::Eip2930(_))
    }

    /// Returns true if the transaction is an EIP-1559 transaction.
    #[inline]
    pub const fn is_eip1559(&self) -> bool {
        matches!(self, Self::Eip1559(_))
    }

    /// Returns true if the transaction is a system transaction.
    #[inline]
    pub const fn is_system_transaction(&self) -> bool {
        match self {
            Self::Deposit(tx) => tx.inner().is_system_transaction,
            _ => false,
        }
    }

    /// Attempts to convert the envelope into the pooled variant.
    ///
    /// Returns an error if the envelope's variant is incompatible with the pooled format:
    /// [`TxDeposit`].
    pub fn try_into_pooled(self) -> Result<OpPooledTransaction, ValueError<Self>> {
        match self {
            Self::Legacy(tx) => Ok(tx.into()),
            Self::Eip2930(tx) => Ok(tx.into()),
            Self::Eip1559(tx) => Ok(tx.into()),
            Self::Eip7702(tx) => Ok(tx.into()),
            Self::Deposit(tx) => {
                Err(ValueError::new(tx.into(), "Deposit transactions cannot be pooled"))
            }
        }
    }

    /// Attempts to convert the envelope into the ethereum pooled variant.
    ///
    /// Returns an error if the envelope's variant is incompatible with the pooled format:
    /// [`TxDeposit`].
    pub fn try_into_eth_pooled(
        self,
    ) -> Result<alloy_consensus::transaction::PooledTransaction, ValueError<Self>> {
        self.try_into_pooled().map(Into::into)
    }

    /// Attempts to convert the optimism variant into an ethereum [`TxEnvelope`].
    ///
    /// Returns the envelope as error if it is a variant unsupported on ethereum: [`TxDeposit`]
    pub fn try_into_eth_envelope(self) -> Result<TxEnvelope, ValueError<Self>> {
        match self {
            Self::Legacy(tx) => Ok(tx.into()),
            Self::Eip2930(tx) => Ok(tx.into()),
            Self::Eip1559(tx) => Ok(tx.into()),
            Self::Eip7702(tx) => Ok(tx.into()),
            tx @ Self::Deposit(_) => Err(ValueError::new(
                tx,
                "Deposit transactions cannot be converted to ethereum transaction",
            )),
        }
    }

    /// Helper that creates [`OpTransactionInfo`] by adding [`OpDepositInfo`] obtained from the
    /// given closure if this transaction is a deposit and return the [`OpTransactionInfo`].
    pub fn try_to_tx_info<F, E>(
        &self,
        tx_info: TransactionInfo,
        f: F,
    ) -> Result<OpTransactionInfo, E>
    where
        F: FnOnce(TxHash) -> Result<Option<OpDepositInfo>, E>,
    {
        let deposit_meta =
            if self.is_deposit() { f(self.tx_hash())? } else { None }.unwrap_or_default();

        Ok(OpTransactionInfo::new(tx_info, deposit_meta))
    }

    /// Attempts to convert an ethereum [`TxEnvelope`] into the optimism variant.
    ///
    /// Returns the given envelope as error if [`OpTxEnvelope`] doesn't support the variant
    /// (EIP-4844)
    pub fn try_from_eth_envelope<T>(
        tx: EthereumTxEnvelope<T>,
    ) -> Result<Self, EthereumTxEnvelope<T>> {
        match tx {
            EthereumTxEnvelope::Legacy(tx) => Ok(tx.into()),
            EthereumTxEnvelope::Eip2930(tx) => Ok(tx.into()),
            EthereumTxEnvelope::Eip1559(tx) => Ok(tx.into()),
            tx @ EthereumTxEnvelope::<T>::Eip4844(_) => Err(tx),
            EthereumTxEnvelope::Eip7702(tx) => Ok(tx.into()),
        }
    }

    /// Returns mutable access to the input bytes.
    ///
    /// Caution: modifying this will cause side-effects on the hash.
    #[doc(hidden)]
    pub const fn input_mut(&mut self) -> &mut Bytes {
        match self {
            Self::Eip1559(tx) => &mut tx.tx_mut().input,
            Self::Eip2930(tx) => &mut tx.tx_mut().input,
            Self::Legacy(tx) => &mut tx.tx_mut().input,
            Self::Eip7702(tx) => &mut tx.tx_mut().input,
            Self::Deposit(tx) => &mut tx.inner_mut().input,
        }
    }

    /// Attempts to convert an ethereum [`TxEnvelope`] into the optimism variant.
    ///
    /// Returns the given envelope as error if [`OpTxEnvelope`] doesn't support the variant
    /// (EIP-4844)
    #[cfg(feature = "alloy-compat")]
    pub fn try_from_any_envelope(
        tx: alloy_network::AnyTxEnvelope,
    ) -> Result<Self, alloy_network::AnyTxEnvelope> {
        match tx.try_into_envelope() {
            Ok(eth) => {
                Self::try_from_eth_envelope(eth).map_err(alloy_network::AnyTxEnvelope::Ethereum)
            }
            Err(err) => match err.into_value() {
                alloy_network::AnyTxEnvelope::Unknown(unknown) => {
                    let Ok(deposit) = unknown.inner.clone().try_into() else {
                        return Err(alloy_network::AnyTxEnvelope::Unknown(unknown));
                    };
                    Ok(Self::Deposit(Sealed::new_unchecked(deposit, unknown.hash)))
                }
                unsupported => Err(unsupported),
            },
        }
    }

    /// Returns true if the transaction is a deposit transaction.
    #[inline]
    pub const fn is_deposit(&self) -> bool {
        matches!(self, Self::Deposit(_))
    }

    /// Returns the [`TxLegacy`] variant if the transaction is a legacy transaction.
    pub const fn as_legacy(&self) -> Option<&Signed<TxLegacy>> {
        match self {
            Self::Legacy(tx) => Some(tx),
            _ => None,
        }
    }

    /// Returns the [`TxEip2930`] variant if the transaction is an EIP-2930 transaction.
    pub const fn as_eip2930(&self) -> Option<&Signed<TxEip2930>> {
        match self {
            Self::Eip2930(tx) => Some(tx),
            _ => None,
        }
    }

    /// Returns the [`TxEip1559`] variant if the transaction is an EIP-1559 transaction.
    pub const fn as_eip1559(&self) -> Option<&Signed<TxEip1559>> {
        match self {
            Self::Eip1559(tx) => Some(tx),
            _ => None,
        }
    }

    /// Returns the [`TxEip1559`] variant if the transaction is an EIP-1559 transaction.
    pub const fn as_deposit(&self) -> Option<&Sealed<TxDeposit>> {
        match self {
            Self::Deposit(tx) => Some(tx),
            _ => None,
        }
    }

    /// Return the reference to signature.
    ///
    /// Returns `None` if this is a deposit variant.
    pub const fn signature(&self) -> Option<&Signature> {
        match self {
            Self::Legacy(tx) => Some(tx.signature()),
            Self::Eip2930(tx) => Some(tx.signature()),
            Self::Eip1559(tx) => Some(tx.signature()),
            Self::Eip7702(tx) => Some(tx.signature()),
            Self::Deposit(_) => None,
        }
    }

    /// Return the [`OpTxType`] of the inner txn.
    pub const fn tx_type(&self) -> OpTxType {
        match self {
            Self::Legacy(_) => OpTxType::Legacy,
            Self::Eip2930(_) => OpTxType::Eip2930,
            Self::Eip1559(_) => OpTxType::Eip1559,
            Self::Eip7702(_) => OpTxType::Eip7702,
            Self::Deposit(_) => OpTxType::Deposit,
        }
    }

    /// Returns the inner transaction hash.
    pub fn hash(&self) -> &B256 {
        match self {
            Self::Legacy(tx) => tx.hash(),
            Self::Eip1559(tx) => tx.hash(),
            Self::Eip2930(tx) => tx.hash(),
            Self::Eip7702(tx) => tx.hash(),
            Self::Deposit(tx) => tx.hash_ref(),
        }
    }

    /// Returns the inner transaction hash.
    pub fn tx_hash(&self) -> B256 {
        *self.hash()
    }

    /// Return the length of the inner txn, including type byte length
    pub fn eip2718_encoded_length(&self) -> usize {
        match self {
            Self::Legacy(t) => t.eip2718_encoded_length(),
            Self::Eip2930(t) => t.eip2718_encoded_length(),
            Self::Eip1559(t) => t.eip2718_encoded_length(),
            Self::Eip7702(t) => t.eip2718_encoded_length(),
            Self::Deposit(t) => t.eip2718_encoded_length(),
        }
    }
}

impl TxHashRef for OpTxEnvelope {
    fn tx_hash(&self) -> &B256 {
        Self::hash(self)
    }
}

#[cfg(feature = "k256")]
impl alloy_consensus::transaction::SignerRecoverable for OpTxEnvelope {
    fn recover_signer(
        &self,
    ) -> Result<alloy_primitives::Address, alloy_consensus::crypto::RecoveryError> {
        let signature_hash = match self {
            Self::Legacy(tx) => tx.signature_hash(),
            Self::Eip2930(tx) => tx.signature_hash(),
            Self::Eip1559(tx) => tx.signature_hash(),
            Self::Eip7702(tx) => tx.signature_hash(),
            // Optimism's Deposit transaction does not have a signature. Directly return the
            // `from` address.
            Self::Deposit(tx) => return Ok(tx.from),
        };
        let signature = match self {
            Self::Legacy(tx) => tx.signature(),
            Self::Eip2930(tx) => tx.signature(),
            Self::Eip1559(tx) => tx.signature(),
            Self::Eip7702(tx) => tx.signature(),
            Self::Deposit(_) => unreachable!("Deposit transactions should not be handled here"),
        };
        alloy_consensus::crypto::secp256k1::recover_signer(signature, signature_hash)
    }

    fn recover_signer_unchecked(
        &self,
    ) -> Result<alloy_primitives::Address, alloy_consensus::crypto::RecoveryError> {
        let signature_hash = match self {
            Self::Legacy(tx) => tx.signature_hash(),
            Self::Eip2930(tx) => tx.signature_hash(),
            Self::Eip1559(tx) => tx.signature_hash(),
            Self::Eip7702(tx) => tx.signature_hash(),
            // Optimism's Deposit transaction does not have a signature. Directly return the
            // `from` address.
            Self::Deposit(tx) => return Ok(tx.from),
        };
        let signature = match self {
            Self::Legacy(tx) => tx.signature(),
            Self::Eip2930(tx) => tx.signature(),
            Self::Eip1559(tx) => tx.signature(),
            Self::Eip7702(tx) => tx.signature(),
            Self::Deposit(_) => unreachable!("Deposit transactions should not be handled here"),
        };
        alloy_consensus::crypto::secp256k1::recover_signer_unchecked(signature, signature_hash)
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
            Self::Deposit(tx) => Ok(tx.from),
        }
    }
}

/// Bincode-compatible serde implementation for OpTxEnvelope.
#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub mod serde_bincode_compat {
    use crate::serde_bincode_compat::TxDeposit;
    use alloy_consensus::{
        Sealed, Signed,
        transaction::serde_bincode_compat::{TxEip1559, TxEip2930, TxEip7702, TxLegacy},
    };
    use alloy_primitives::{B256, Signature};
    use serde::{Deserialize, Deserializer, Serialize, Serializer};
    use serde_with::{DeserializeAs, SerializeAs};

    /// Bincode-compatible representation of an OpTxEnvelope.
    #[derive(Debug, Serialize, Deserialize)]
    pub enum OpTxEnvelope<'a> {
        /// Legacy variant.
        Legacy {
            /// Transaction signature.
            signature: Signature,
            /// Borrowed legacy transaction data.
            transaction: TxLegacy<'a>,
        },
        /// EIP-2930 variant.
        Eip2930 {
            /// Transaction signature.
            signature: Signature,
            /// Borrowed EIP-2930 transaction data.
            transaction: TxEip2930<'a>,
        },
        /// EIP-1559 variant.
        Eip1559 {
            /// Transaction signature.
            signature: Signature,
            /// Borrowed EIP-1559 transaction data.
            transaction: TxEip1559<'a>,
        },
        /// EIP-7702 variant.
        Eip7702 {
            /// Transaction signature.
            signature: Signature,
            /// Borrowed EIP-7702 transaction data.
            transaction: TxEip7702<'a>,
        },
        /// Deposit variant.
        Deposit {
            /// Precomputed hash.
            hash: B256,
            /// Borrowed deposit transaction data.
            transaction: TxDeposit<'a>,
        },
    }

    impl<'a> From<&'a super::OpTxEnvelope> for OpTxEnvelope<'a> {
        fn from(value: &'a super::OpTxEnvelope) -> Self {
            match value {
                super::OpTxEnvelope::Legacy(signed_legacy) => Self::Legacy {
                    signature: *signed_legacy.signature(),
                    transaction: signed_legacy.tx().into(),
                },
                super::OpTxEnvelope::Eip2930(signed_2930) => Self::Eip2930 {
                    signature: *signed_2930.signature(),
                    transaction: signed_2930.tx().into(),
                },
                super::OpTxEnvelope::Eip1559(signed_1559) => Self::Eip1559 {
                    signature: *signed_1559.signature(),
                    transaction: signed_1559.tx().into(),
                },
                super::OpTxEnvelope::Eip7702(signed_7702) => Self::Eip7702 {
                    signature: *signed_7702.signature(),
                    transaction: signed_7702.tx().into(),
                },
                super::OpTxEnvelope::Deposit(sealed_deposit) => Self::Deposit {
                    hash: sealed_deposit.seal(),
                    transaction: sealed_deposit.inner().into(),
                },
            }
        }
    }

    impl<'a> From<OpTxEnvelope<'a>> for super::OpTxEnvelope {
        fn from(value: OpTxEnvelope<'a>) -> Self {
            match value {
                OpTxEnvelope::Legacy { signature, transaction } => {
                    Self::Legacy(Signed::new_unhashed(transaction.into(), signature))
                }
                OpTxEnvelope::Eip2930 { signature, transaction } => {
                    Self::Eip2930(Signed::new_unhashed(transaction.into(), signature))
                }
                OpTxEnvelope::Eip1559 { signature, transaction } => {
                    Self::Eip1559(Signed::new_unhashed(transaction.into(), signature))
                }
                OpTxEnvelope::Eip7702 { signature, transaction } => {
                    Self::Eip7702(Signed::new_unhashed(transaction.into(), signature))
                }
                OpTxEnvelope::Deposit { hash, transaction } => {
                    Self::Deposit(Sealed::new_unchecked(transaction.into(), hash))
                }
            }
        }
    }

    impl SerializeAs<super::OpTxEnvelope> for OpTxEnvelope<'_> {
        fn serialize_as<S>(source: &super::OpTxEnvelope, serializer: S) -> Result<S::Ok, S::Error>
        where
            S: Serializer,
        {
            let borrowed = OpTxEnvelope::from(source);
            borrowed.serialize(serializer)
        }
    }

    impl<'de> DeserializeAs<'de, super::OpTxEnvelope> for OpTxEnvelope<'de> {
        fn deserialize_as<D>(deserializer: D) -> Result<super::OpTxEnvelope, D::Error>
        where
            D: Deserializer<'de>,
        {
            let borrowed = OpTxEnvelope::deserialize(deserializer)?;
            Ok(borrowed.into())
        }
    }

    #[cfg(test)]
    mod tests {
        use super::*;
        use arbitrary::Arbitrary;
        use rand::Rng;
        use serde::{Deserialize, Serialize};
        use serde_with::serde_as;

        /// Tests a bincode round-trip for OpTxEnvelope using an arbitrary instance.
        #[test]
        fn test_op_tx_envelope_bincode_roundtrip_arbitrary() {
            #[serde_as]
            #[derive(Debug, PartialEq, Eq, Serialize, Deserialize)]
            struct Data {
                // Use the bincode-compatible representation defined in this module.
                #[serde_as(as = "OpTxEnvelope<'_>")]
                envelope: super::super::OpTxEnvelope,
            }

            let mut bytes = [0u8; 1024];
            rand::rng().fill(bytes.as_mut_slice());
            let data = Data {
                envelope: super::super::OpTxEnvelope::arbitrary(&mut arbitrary::Unstructured::new(
                    &bytes,
                ))
                .unwrap(),
            };

            let encoded = bincode::serde::encode_to_vec(&data, bincode::config::legacy()).unwrap();
            let (decoded, _) =
                bincode::serde::decode_from_slice::<Data, _>(&encoded, bincode::config::legacy())
                    .unwrap();
            assert_eq!(decoded, data);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_consensus::{SignableTransaction, Transaction};
    use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256, hex};

    #[test]
    fn test_tx_gas_limit() {
        let tx = TxDeposit { gas_limit: 1, ..Default::default() };
        let tx_envelope = OpTxEnvelope::Deposit(tx.seal_slow());
        assert_eq!(tx_envelope.gas_limit(), 1);
    }

    #[test]
    fn test_deposit() {
        let tx = TxDeposit { is_system_transaction: true, ..Default::default() };
        let tx_envelope = OpTxEnvelope::Deposit(tx.seal_slow());
        assert!(tx_envelope.is_deposit());

        let tx = TxEip1559::default();
        let sig = Signature::test_signature();
        let tx_envelope = OpTxEnvelope::Eip1559(tx.into_signed(sig));
        assert!(!tx_envelope.is_system_transaction());
    }

    #[test]
    fn test_system_transaction() {
        let mut tx = TxDeposit { is_system_transaction: true, ..Default::default() };
        let tx_envelope = OpTxEnvelope::Deposit(tx.clone().seal_slow());
        assert!(tx_envelope.is_system_transaction());

        tx.is_system_transaction = false;
        let tx_envelope = OpTxEnvelope::Deposit(tx.seal_slow());
        assert!(!tx_envelope.is_system_transaction());
    }

    #[test]
    fn test_encode_decode_deposit() {
        let tx = TxDeposit {
            source_hash: B256::left_padding_from(&[0xde, 0xad]),
            from: Address::left_padding_from(&[0xbe, 0xef]),
            mint: 1,
            gas_limit: 2,
            to: TxKind::Call(Address::left_padding_from(&[3])),
            value: U256::from(4_u64),
            input: Bytes::from(vec![5]),
            is_system_transaction: false,
        };
        let tx_envelope = OpTxEnvelope::Deposit(tx.seal_slow());
        let encoded = tx_envelope.encoded_2718();
        let decoded = OpTxEnvelope::decode_2718(&mut encoded.as_ref()).unwrap();
        assert_eq!(encoded.len(), tx_envelope.encode_2718_len());
        assert_eq!(decoded, tx_envelope);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_serde_roundtrip_deposit() {
        let tx = TxDeposit {
            gas_limit: u64::MAX,
            to: TxKind::Call(Address::random()),
            value: U256::MAX,
            input: Bytes::new(),
            source_hash: U256::MAX.into(),
            from: Address::random(),
            mint: u128::MAX,
            is_system_transaction: false,
        };
        let tx_envelope = OpTxEnvelope::Deposit(tx.seal_slow());

        let serialized = serde_json::to_string(&tx_envelope).unwrap();
        let deserialized: OpTxEnvelope = serde_json::from_str(&serialized).unwrap();

        assert_eq!(tx_envelope, deserialized);
    }

    #[test]
    fn eip2718_deposit_decode() {
        // <https://basescan.org/tx/0xc468b38a20375922828c8126912740105125143b9856936085474b2590bbca91>
        let b = hex!(
            "7ef8f8a0417d134467f4737fcdf2475f0ecdd2a0ed6d87ecffc888ba9f60ee7e3b8ac26a94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000008dd00101c1200000000000000040000000066c352bb000000000139c4f500000000000000000000000000000000000000000000000000000000c0cff1460000000000000000000000000000000000000000000000000000000000000001d4c88f4065ac9671e8b1329b90773e89b5ddff9cf8675b2b5e9c1b28320609930000000000000000000000005050f69a9786f081509234f1a7f4684b5e5b76c9"
        );

        let tx = OpTxEnvelope::decode_2718(&mut b[..].as_ref()).unwrap();
        let deposit = tx.as_deposit().unwrap();
        assert_eq!(deposit.mint, 0);
    }

    #[test]
    fn eip1559_decode() {
        let tx = TxEip1559 {
            chain_id: 1u64,
            nonce: 2,
            max_fee_per_gas: 3,
            max_priority_fee_per_gas: 4,
            gas_limit: 5,
            to: Address::left_padding_from(&[6]).into(),
            value: U256::from(7_u64),
            input: vec![8].into(),
            access_list: Default::default(),
        };
        let sig = Signature::test_signature();
        let tx_signed = tx.into_signed(sig);
        let envelope: OpTxEnvelope = tx_signed.into();
        let encoded = envelope.encoded_2718();
        let mut slice = encoded.as_slice();
        let decoded = OpTxEnvelope::decode_2718(&mut slice).unwrap();
        assert!(matches!(decoded, OpTxEnvelope::Eip1559(_)));
    }
}
