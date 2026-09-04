//! Reserved for Rust implementations of SQL user-defined functions.
//!
//! The chain view declares no UDFs today: every fact is decoded host-side before it is pushed
//! into the circuit. `sql-to-dbsp` still emits `pub mod udf;`, so this file must exist.
