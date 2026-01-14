//! Contains the kind of system config update.

/// Represents type of update to the system config.
#[derive(Debug, Clone, Copy, Hash, PartialEq, Eq, derive_more::TryFrom)]
#[try_from(repr)]
#[repr(u64)]
pub enum SystemConfigUpdateKind {
    /// Batcher update type
    Batcher = 0,
    /// Gas config update type
    GasConfig = 1,
    /// Gas limit update type
    GasLimit = 2,
    /// Unsafe block signer update type
    UnsafeBlockSigner = 3,
    /// EIP-1559 parameters update type
    Eip1559 = 4,
    /// Operator fee parameter update
    OperatorFee = 5,
    /// Min base fee parameter update
    MinBaseFee = 6,
    /// DA footprint gas scalar update type
    DaFootprintGasScalar = 7,
}
