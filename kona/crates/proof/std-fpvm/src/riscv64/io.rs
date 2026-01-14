use crate::{BasicKernelInterface, FileDescriptor, errors::IOResult, riscv64::syscall};

/// Concrete implementation of the [`KernelIO`] trait for the `riscv64` target architecture.
#[derive(Debug)]
pub(crate) struct RiscV64IO;

/// Relevant system call numbers for the `riscv64` target architecture.
///
/// See https://jborza.com/post/2021-05-11-riscv-linux-syscalls/
///
/// **Note**: This is not an exhaustive list of system calls available to the `client` program,
/// only the ones necessary for the [BasicKernelInterface] trait implementation. If an extension
/// trait for the [BasicKernelInterface] trait is created for the linux kernel, this list
/// should be extended accordingly.
#[repr(usize)]
pub(crate) enum SyscallNumber {
    /// Sets the Exited and ExitCode states to true and $a0 respectively.
    Exit = 93,
    /// Similar behavior as Linux with support for unaligned reads.
    Read = 63,
    /// Similar behavior as Linux with support for unaligned writes.
    Write = 64,
    /// Similar behavior as Linux for mapping memory on the host machine.
    Mmap = 222,
}

impl BasicKernelInterface for RiscV64IO {
    fn write(fd: FileDescriptor, buf: &[u8]) -> IOResult<usize> {
        unsafe {
            crate::linux::from_ret(syscall::syscall3(
                SyscallNumber::Write as usize,
                fd.into(),
                buf.as_ptr() as usize,
                buf.len(),
            ))
        }
    }

    fn read(fd: FileDescriptor, buf: &mut [u8]) -> IOResult<usize> {
        unsafe {
            crate::linux::from_ret(syscall::syscall3(
                SyscallNumber::Read as usize,
                fd.into(),
                buf.as_ptr() as usize,
                buf.len(),
            ))
        }
    }

    fn mmap(size: usize) -> IOResult<usize> {
        // https://github.com/ethereum-optimism/asterisc/blob/master/rvgo/fast/vm.go#L360-L398
        unsafe {
            crate::linux::from_ret(syscall::syscall6(
                SyscallNumber::Mmap as usize,
                0usize,            // address hint - 0 for anonymous maps
                size,              // block size
                0usize,            // prot, ignored.
                0x20,              // flags - set MAP_ANONYMOUS
                u64::MAX as usize, // fd = -1, anonymous memory maps only.
                0usize,            // offset - ignored, anonymous memory maps only.
            ))
        }
    }

    fn exit(code: usize) -> ! {
        unsafe {
            let _ = syscall::syscall1(SyscallNumber::Exit as usize, code);
            panic!()
        }
    }
}
