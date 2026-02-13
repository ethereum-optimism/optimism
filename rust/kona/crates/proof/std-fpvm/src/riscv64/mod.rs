//! This module contains raw syscall bindings for the `riscv64imac` target architecture, as well as
//! a high-level implementation of the [`crate::BasicKernelInterface`] trait for the kernel.

pub(crate) mod io;
mod syscall;
