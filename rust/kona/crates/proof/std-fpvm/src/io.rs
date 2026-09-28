//! This module contains the `ClientIO` struct, which is a system call interface for the kernel.

use crate::{BasicKernelInterface, FileDescriptor, errors::IOResult};
use cfg_if::cfg_if;

cfg_if! {
    if #[cfg(target_arch = "mips64")] {
        #[doc = "Concrete implementation of the [`BasicKernelInterface`] trait for the `MIPS64r2` target architecture."]
        pub(crate) type ClientIO = crate::mips64::io::Mips64IO;
    } else if #[cfg(target_arch = "riscv64")] {
        #[doc = "Concrete implementation of the [`BasicKernelInterface`] trait for the `riscv64` target architecture."]
        pub(crate) type ClientIO = crate::riscv64::io::RiscV64IO;
    } else {
        use std::{fs::File, mem::ManuallyDrop, os::fd::{FromRawFd, RawFd}, io::{Read, Write}};
        use crate::errors::IOError;

        #[doc = "Native implementation of the [`BasicKernelInterface`] trait."]
        pub(crate) struct NativeClientIO;

        impl NativeClientIO {
            /// Wraps `fd` in a [`File`] that never closes it, on success and error paths alike.
            /// The kernel interface only borrows its descriptors, and dropping a `File` for an
            /// invalid one aborts the process.
            fn borrow_file(fd: RawFd) -> ManuallyDrop<File> {
                // SAFETY: the `File` only issues read/write syscalls on `fd` for the duration of
                // the call and is never dropped, so it never takes ownership of or closes `fd`;
                // an invalid `fd` surfaces as an `EBADF` error from the syscall.
                ManuallyDrop::new(unsafe { File::from_raw_fd(fd) })
            }

            fn write_raw(fd: RawFd, buf: &[u8]) -> IOResult<usize> {
                Self::borrow_file(fd).write_all(buf).map_err(|_| IOError(-9))?;
                Ok(buf.len())
            }

            fn read_raw(fd: RawFd, buf: &mut [u8]) -> IOResult<usize> {
                Self::borrow_file(fd).read_exact(buf).map_err(|_| IOError(-9))?;
                Ok(buf.len())
            }
        }

        impl BasicKernelInterface for NativeClientIO {
            fn write(fd: FileDescriptor, buf: &[u8]) -> IOResult<usize> {
                Self::write_raw(fd.into(), buf)
            }

            fn read(fd: FileDescriptor, buf: &mut [u8]) -> IOResult<usize> {
                Self::read_raw(fd.into(), buf)
            }

            fn mmap(_size: usize) -> IOResult<usize> {
                unimplemented!("mmap is unimplemented for the native target; The default global allocator is favored.");
            }

            fn exit(code: usize) -> ! {
                std::process::exit(code as i32)
            }
        }

        #[doc = "Native implementation of the [`BasicKernelInterface`] trait."]
        pub(crate) type ClientIO = NativeClientIO;
    }
}

/// Print the passed string to the standard output [`FileDescriptor`].
///
/// # Panics
/// Panics if the write operation fails.
#[inline]
pub fn print(s: &str) {
    ClientIO::write(FileDescriptor::StdOut, s.as_bytes()).expect("Error writing to stdout.");
}

/// Print the passed string to the standard error [`FileDescriptor`].
///
/// # Panics
/// Panics if the write operation fails.
#[inline]
pub fn print_err(s: &str) {
    ClientIO::write(FileDescriptor::StdErr, s.as_bytes()).expect("Error writing to stderr.");
}

/// Write the passed buffer to the given [`FileDescriptor`].
#[inline]
pub fn write(fd: FileDescriptor, buf: &[u8]) -> IOResult<usize> {
    ClientIO::write(fd, buf)
}

/// Write the passed buffer to the given [`FileDescriptor`].
#[inline]
pub fn read(fd: FileDescriptor, buf: &mut [u8]) -> IOResult<usize> {
    ClientIO::read(fd, buf)
}

/// Map new memory of block size `size`. Returns the new heap pointer.
#[inline]
pub fn mmap(size: usize) -> IOResult<usize> {
    ClientIO::mmap(size)
}

/// Exit the process with the given exit code.
#[inline]
pub fn exit(code: usize) -> ! {
    ClientIO::exit(code)
}

#[cfg(all(test, not(any(target_arch = "mips64", target_arch = "riscv64"))))]
mod tests {
    use super::NativeClientIO;
    use crate::errors::IOError;
    use std::{
        io::{Read, Write},
        os::{fd::AsRawFd, unix::net::UnixStream},
    };

    const UNOPENED_FD: i32 = 1_000_000;

    #[test]
    fn write_to_invalid_fd_returns_error() {
        assert_eq!(NativeClientIO::write_raw(UNOPENED_FD, b"hello"), Err(IOError(-9)));
    }

    #[test]
    fn read_from_invalid_fd_returns_error() {
        let mut buf = [0u8; 5];
        assert_eq!(NativeClientIO::read_raw(UNOPENED_FD, &mut buf), Err(IOError(-9)));
    }

    #[test]
    fn write_leaves_fd_open() {
        let (a, mut b) = UnixStream::pair().unwrap();
        assert_eq!(NativeClientIO::write_raw(a.as_raw_fd(), b"hello"), Ok(5));
        assert_eq!(NativeClientIO::write_raw(a.as_raw_fd(), b"world"), Ok(5));

        let mut buf = [0u8; 10];
        b.read_exact(&mut buf).unwrap();
        assert_eq!(&buf, b"helloworld");
    }

    /// The fd must survive the error path too: after a failed read the same descriptor is
    /// still usable.
    #[test]
    fn read_error_leaves_fd_open() {
        let (mut a, b) = UnixStream::pair().unwrap();
        b.set_read_timeout(Some(std::time::Duration::from_millis(50))).unwrap();

        let mut buf = [0u8; 5];
        assert_eq!(NativeClientIO::read_raw(b.as_raw_fd(), &mut buf), Err(IOError(-9)));

        a.write_all(b"hello").unwrap();
        assert_eq!(NativeClientIO::read_raw(b.as_raw_fd(), &mut buf), Ok(5));
        assert_eq!(&buf, b"hello");
    }

    #[test]
    fn read_leaves_fd_open() {
        let (mut a, b) = UnixStream::pair().unwrap();
        a.write_all(b"helloworld").unwrap();

        let mut buf = [0u8; 5];
        assert_eq!(NativeClientIO::read_raw(b.as_raw_fd(), &mut buf), Ok(5));
        assert_eq!(&buf, b"hello");
        assert_eq!(NativeClientIO::read_raw(b.as_raw_fd(), &mut buf), Ok(5));
        assert_eq!(&buf, b"world");
    }
}
