//! Optimism constants used in the Optimism EVM.
use revm::primitives::{address, Address, U256};

/// The cost of a non-zero byte in the EVM.
pub const NON_ZERO_BYTE_COST: u64 = 16;

/// The two 4-byte Ecotone fee scalar values are packed into the same storage slot as the 8-byte sequence number.
/// Byte offset within the storage slot of the 4-byte baseFeeScalar attribute.
pub const BASE_FEE_SCALAR_OFFSET: usize = 16;
/// The two 4-byte Ecotone fee scalar values are packed into the same storage slot as the 8-byte sequence number.
/// Byte offset within the storage slot of the 4-byte blobBaseFeeScalar attribute.
pub const BLOB_BASE_FEE_SCALAR_OFFSET: usize = 20;

/// The Isthmus operator fee scalar values are similarly packed. Byte offset within
/// the storage slot of the 4-byte operatorFeeScalar attribute.
pub const OPERATOR_FEE_SCALAR_OFFSET: usize = 20;
/// The Isthmus operator fee scalar values are similarly packed. Byte offset within
/// the storage slot of the 8-byte operatorFeeConstant attribute.
pub const OPERATOR_FEE_CONSTANT_OFFSET: usize = 24;

/// The Jovian daFootprintGasScalar value is packed into a single storage slot. Byte offset within
/// the storage slot of the 16-byte daFootprintGasScalar attribute.
pub const DA_FOOTPRINT_GAS_SCALAR_OFFSET: usize = 18;

/// The fixed point decimal scaling factor associated with the operator fee scalar.
///
/// Allows users to use 6 decimal points of precision when specifying the operator_fee_scalar.
pub const OPERATOR_FEE_SCALAR_DECIMAL: u64 = 1_000_000;

/// The Jovian multiplier applied to the operator fee scalar component.
pub const OPERATOR_FEE_JOVIAN_MULTIPLIER: u64 = 100;

/// The L1 base fee slot.
pub const L1_BASE_FEE_SLOT: U256 = U256::from_limbs([1u64, 0, 0, 0]);
/// The L1 overhead slot.
pub const L1_OVERHEAD_SLOT: U256 = U256::from_limbs([5u64, 0, 0, 0]);
/// The L1 scalar slot.
pub const L1_SCALAR_SLOT: U256 = U256::from_limbs([6u64, 0, 0, 0]);

/// [ECOTONE_L1_BLOB_BASE_FEE_SLOT] was added in the Ecotone upgrade and stores the L1 blobBaseFee attribute.
pub const ECOTONE_L1_BLOB_BASE_FEE_SLOT: U256 = U256::from_limbs([7u64, 0, 0, 0]);

/// As of the ecotone upgrade, this storage slot stores the 32-bit basefeeScalar and blobBaseFeeScalar attributes at
/// offsets [BASE_FEE_SCALAR_OFFSET] and [BLOB_BASE_FEE_SCALAR_OFFSET] respectively.
pub const ECOTONE_L1_FEE_SCALARS_SLOT: U256 = U256::from_limbs([3u64, 0, 0, 0]);

/// This storage slot stores the 32-bit operatorFeeScalar and operatorFeeConstant attributes at
/// offsets [OPERATOR_FEE_SCALAR_OFFSET] and [OPERATOR_FEE_CONSTANT_OFFSET] respectively.
pub const OPERATOR_FEE_SCALARS_SLOT: U256 = U256::from_limbs([8u64, 0, 0, 0]);

/// As of the Jovian upgrade, this storage slot stores the 16-bit daFootprintGasScalar attribute at
/// offset [DA_FOOTPRINT_GAS_SCALAR_OFFSET].
pub const DA_FOOTPRINT_GAS_SCALAR_SLOT: U256 = U256::from_limbs([8u64, 0, 0, 0]);

/// An empty 64-bit set of scalar values.
pub const EMPTY_SCALARS: [u8; 8] = [0u8; 8];

/// The address of L1 fee recipient.
pub const L1_FEE_RECIPIENT: Address = address!("0x420000000000000000000000000000000000001A");

/// The address of the operator fee recipient.
pub const OPERATOR_FEE_RECIPIENT: Address = address!("0x420000000000000000000000000000000000001B");

/// The address of the base fee recipient.
pub const BASE_FEE_RECIPIENT: Address = address!("0x4200000000000000000000000000000000000019");

/// The address of the L1Block contract.
pub const L1_BLOCK_CONTRACT: Address = address!("0x4200000000000000000000000000000000000015");
