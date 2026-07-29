//! Contains the [`SystemConfigUpdate`].

use crate::{
    BatcherUpdate, Eip1559Update, GasConfigUpdate, GasLimitUpdate, OperatorFeeUpdate, SystemConfig,
    SystemConfigUpdateKind, UnsafeBlockSignerUpdate,
    updates::{DaFootprintGasScalarUpdate, MinBaseFeeUpdate},
};

/// The system config update is an update
/// of type [`SystemConfigUpdateKind`].
#[derive(Debug, Clone, Hash, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum SystemConfigUpdate {
    /// The batcher update.
    Batcher(BatcherUpdate),
    /// The gas config update.
    GasConfig(GasConfigUpdate),
    /// The gas limit update.
    GasLimit(GasLimitUpdate),
    /// The unsafe block signer update.
    UnsafeBlockSigner(UnsafeBlockSignerUpdate),
    /// The EIP-1559 parameters update.
    Eip1559(Eip1559Update),
    /// The operator fee parameter update.
    OperatorFee(OperatorFeeUpdate),
    /// Min base fee parameter update.
    MinBaseFee(MinBaseFeeUpdate),
    /// DA footprint gas scalar update.
    DaFootprintGasScalar(DaFootprintGasScalarUpdate),
}

impl SystemConfigUpdate {
    /// Applies the update to the [`SystemConfig`].
    pub const fn apply(&self, config: &mut SystemConfig) {
        match self {
            Self::Batcher(update) => update.apply(config),
            Self::GasConfig(update) => update.apply(config),
            Self::GasLimit(update) => update.apply(config),
            Self::UnsafeBlockSigner(_) => { /* Ignored in derivation */ }
            Self::Eip1559(update) => update.apply(config),
            Self::OperatorFee(update) => update.apply(config),
            Self::MinBaseFee(update) => update.apply(config),
            Self::DaFootprintGasScalar(update) => update.apply(config),
        }
    }

    /// Returns the update kind.
    pub const fn kind(&self) -> SystemConfigUpdateKind {
        match self {
            Self::Batcher(_) => SystemConfigUpdateKind::Batcher,
            Self::GasConfig(_) => SystemConfigUpdateKind::GasConfig,
            Self::GasLimit(_) => SystemConfigUpdateKind::GasLimit,
            Self::UnsafeBlockSigner(_) => SystemConfigUpdateKind::UnsafeBlockSigner,
            Self::Eip1559(_) => SystemConfigUpdateKind::Eip1559,
            Self::OperatorFee(_) => SystemConfigUpdateKind::OperatorFee,
            Self::MinBaseFee(_) => SystemConfigUpdateKind::MinBaseFee,
            Self::DaFootprintGasScalar(_) => SystemConfigUpdateKind::DaFootprintGasScalar,
        }
    }
}
