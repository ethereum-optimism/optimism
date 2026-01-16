//! Derived from the syscalls crate <https://github.com/jasonwhite/syscalls>
//!
//! Unsafe system call interface for the `riscv64` target architecture.
//!
//! List of RISC-V system calls: <https://jborza.com/post/2021-05-11-riscv-linux-syscalls/>
//!
//! **Registers used for system calls**
//! | Register Number |    Description     |
//! |=================|====================|
//! | %a0             | arg1, return value |
//! | %a1             | arg2               |
//! | %a2             | arg3               |
//! | %a3             | arg4               |
//! | %a4             | arg5               |
//! | %a5             | arg6               |
//! | %a7             | syscall number     |

use core::arch::asm;

/// Issues a raw system call with 1 argument. (e.g. exit)
#[inline]
pub(crate) unsafe fn syscall1(syscall_number: usize, arg1: usize) -> usize {
    let mut ret: usize;
    unsafe {
        asm!(
            "ecall",
            in("a7") syscall_number,
            inlateout("a0") arg1 => ret,
            options(nostack, preserves_flags)
        );
    }
    ret
}

/// Issues a raw system call with 3 arguments. (e.g. read, write)
#[inline]
pub(crate) unsafe fn syscall3(
    syscall_number: usize,
    arg1: usize,
    arg2: usize,
    arg3: usize,
) -> usize {
    let mut ret: usize;
    unsafe {
        asm!(
            "ecall",
            in("a7") syscall_number,
            inlateout("a0") arg1 => ret,
            in("a1") arg2,
            in("a2") arg3,
            options(nostack, preserves_flags)
        );
    }
    ret
}

/// Issues a raw system call with 6 arguments. (e.g. mmap)
///
/// # Safety
///
/// Running a system call is inherently unsafe. It is the caller's
/// responsibility to ensure safety.
#[inline]
pub(crate) unsafe fn syscall6(
    syscall_number: usize,
    arg1: usize,
    arg2: usize,
    arg3: usize,
    arg4: usize,
    arg5: usize,
    arg6: usize,
) -> usize {
    let mut ret: usize;
    unsafe {
        asm!(
            "ecall",
            in("a7") syscall_number,
            inlateout("a0") arg1 => ret,
            in("a1") arg2,
            in("a2") arg3,
            in("a3") arg4,
            in("a4") arg5,
            in("a5") arg6,
            options(nostack, preserves_flags)
        );
    }
    ret
}
