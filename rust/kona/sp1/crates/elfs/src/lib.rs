//! The zkvm ELF binaries.

pub const AGGREGATION_ELF: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/aggregation-elf"));
pub const RANGE_ELF: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/range-elf"));
pub const SUPER_AGGREGATION_ELF: &[u8] =
    include_bytes!(concat!(env!("OUT_DIR"), "/super-aggregation-elf"));
pub const SUPER_RANGE_ELF: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/super-range-elf"));
