//! Contains common traits for the `client` role.
//!
//! When developing a new `client` program, these traits are implemented on an
//! architecture-specific type that provides the concrete implementation of the
//! kernel interfaces. The `client` program then uses these traits to perform operations
//! without needing to know the underlying implementation, which allows the same `client`
//! program to be compiled and ran on different target architectures.

mod basic;
pub use basic::BasicKernelInterface;
