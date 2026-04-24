//! Linux utilities

// Reason: syscall return conversion uses the same usize<->i32 cast discipline as musl
// (see https://git.musl-libc.org/cgit/musl/tree/src/internal/syscall_ret.c); casts are
// bounded by the `value > -4096isize as usize` check and intentional.
#![allow(clippy::redundant_pub_crate)]

use crate::errors::{IOError, IOResult};

/// Converts a return value from a syscall into a [`IOResult`] type.
// SAFETY: see module-level reason; casts are bounded by the musl-style check.
#[allow(
    unused,
    clippy::inline_always,
    clippy::cast_sign_loss,
    clippy::cast_possible_truncation,
    clippy::cast_possible_wrap
)]
#[inline(always)]
pub(crate) const fn from_ret(value: usize) -> IOResult<usize> {
    if value > -4096isize as usize {
        // Truncation of the error value is guaranteed to never occur due to
        // the above check. This is the same check that musl uses:
        // https://git.musl-libc.org/cgit/musl/tree/src/internal/syscall_ret.c?h=v1.1.15
        Err(IOError(-(value as i32)))
    } else {
        Ok(value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    #[allow(clippy::cast_sign_loss)]
    fn test_from_ret_io_error() {
        assert_eq!(from_ret(-4095isize as usize), Err(IOError(4095)));
    }

    #[test]
    fn test_from_ret_ok() {
        assert_eq!(from_ret(1), Ok(1));
    }
}
