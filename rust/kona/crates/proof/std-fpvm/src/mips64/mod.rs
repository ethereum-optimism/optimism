//! This module contains raw syscall bindings for the `MIPS64r2` target architecture, as well as a
//! high-level implementation of the [`crate::BasicKernelInterface`] trait for the `Cannon` kernel.

pub(crate) mod io;
mod syscall;
