//! The zkvm ELF binaries.

pub const AGGREGATION_ELF: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/aggregation-elf"));
pub const RANGE_ELF: &[u8] = include_bytes!(concat!(env!("OUT_DIR"), "/range-elf"));
