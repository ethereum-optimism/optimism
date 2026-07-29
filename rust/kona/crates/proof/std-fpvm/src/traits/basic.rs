//! Defines the [`BasicKernelInterface`] trait, which describes the functionality of several system
//! calls inside of the kernel.

use crate::{FileDescriptor, errors::IOResult};

/// The [`BasicKernelInterface`] trait describes the functionality of several core system calls
/// inside of the kernel.
///
/// Commonly, embedded proving environments delegate IO operations to custom file descriptors.
/// This trait is a safe wrapper around the raw system calls available to the `client` program
/// for host<->client communication.
///
/// In cases where the set of system calls defined in this trait need to be extended, an additional
/// trait should be created that extends this trait.
pub trait BasicKernelInterface {
    /// Write the given buffer to the given file descriptor.
    fn write(fd: FileDescriptor, buf: &[u8]) -> IOResult<usize>;

    /// Read from the given file descriptor into the passed buffer.
    fn read(fd: FileDescriptor, buf: &mut [u8]) -> IOResult<usize>;

    /// Map new memory with block size `size`. Returns the new heap pointer.
    fn mmap(size: usize) -> IOResult<usize>;

    /// Exit the process with the given exit code. The implementation of this function
    /// should always panic after invoking the `EXIT` syscall.
    fn exit(code: usize) -> !;
}
