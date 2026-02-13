//! Derived from the syscalls crate <https://github.com/jasonwhite/syscalls>
//!
//! MIPS has the following registers:
//!
//! | Symbolic Name | Number          | Usage                          |
//! | ============= | =============== | ============================== |
//! | zero          | 0               | Constant 0.                    |
//! | at            | 1               | Reserved for the assembler.    |
//! | v0 - v1       | 2 - 3           | Result Registers.              |
//! | a0 - a3       | 4 - 7           | Argument Registers 1 ·· · 4.   |
//! | t0 - t9       | 8 - 15, 24 - 25 | Temporary Registers 0 · · · 9. |
//! | s0 - s7       | 16 - 23         | Saved Registers 0 ·· · 7.      |
//! | k0 - k1       | 26 - 27         | Kernel Registers 0 ·· · 1.     |
//! | gp            | 28              | Global Data Pointer.           |
//! | sp            | 29              | Stack Pointer.                 |
//! | fp            | 30              | Frame Pointer.                 |
//! | ra            | 31              | Return Address.                |
//!
//! The following registers are used for args 1-6:
//!
//! arg1: %a0 ($4)
//! arg2: %a1 ($5)
//! arg3: %a2 ($6)
//! arg4: %a3 ($7)
//! arg5: (Passed via user stack)
//! arg6: (Passed via user stack)
//! arg7: (Passed via user stack)
//!
//! %v0 is the syscall number.
//! %v0 is the return value.
//! %v1 is the error code
//! %a3 is a boolean indicating that an error occurred.
//!
//!
//! All temporary registers are clobbered (8-15, 24-25).

use core::arch::asm;

/// Issues a raw system call with 1 argument. (e.g. exit)
#[inline]
pub(crate) unsafe fn syscall1(n: usize, arg1: usize) -> usize {
    let mut err: usize;
    let mut ret: usize;
    unsafe {
        asm!(
            "syscall",
            inlateout("$2") n => ret,
            lateout("$7") err,
            in("$4") arg1,
            // Clobber all temporary registers
            lateout("$8") _,
            lateout("$9") _,
            lateout("$10") _,
            lateout("$11") _,
            lateout("$12") _,
            lateout("$13") _,
            lateout("$14") _,
            lateout("$15") _,
            lateout("$24") _,
            lateout("$25") _,
            options(nostack, preserves_flags)
        );
    }

    if err == 0 { ret } else { ret.wrapping_neg() }
}

/// Issues a raw system call with 2 arguments. (e.g. cannon's flavor of mmap)
///
/// # Safety
///
/// Running a system call is inherently unsafe. It is the caller's
/// responsibility to ensure safety.
#[inline]
pub(crate) unsafe fn syscall2(n: usize, arg1: usize, arg2: usize) -> usize {
    let mut err: usize;
    let mut ret: usize;
    unsafe {
        asm!(
            "syscall",
            inlateout("$2") n => ret,
            lateout("$7") err,
            in("$4") arg1,
            in("$5") arg2,
            // All temporary registers are always clobbered
            lateout("$8") _,
            lateout("$9") _,
            lateout("$10") _,
            lateout("$11") _,
            lateout("$12") _,
            lateout("$13") _,
            lateout("$14") _,
            lateout("$15") _,
            lateout("$24") _,
            lateout("$25") _,
            options(nostack, preserves_flags)
        );
    }
    if err == 0 { ret } else { ret.wrapping_neg() }
}

/// Issues a raw system call with 3 arguments. (e.g. read, write)
#[inline]
pub(crate) unsafe fn syscall3(n: usize, arg1: usize, arg2: usize, arg3: usize) -> usize {
    let mut err: usize;
    let mut ret: usize;
    unsafe {
        asm!(
            "syscall",
            inlateout("$2") n => ret,
            lateout("$7") err,
            in("$4") arg1,
            in("$5") arg2,
            in("$6") arg3,
            // Clobber all temporary registers
            lateout("$8") _,
            lateout("$9") _,
            lateout("$10") _,
            lateout("$11") _,
            lateout("$12") _,
            lateout("$13") _,
            lateout("$14") _,
            lateout("$15") _,
            lateout("$24") _,
            lateout("$25") _,
            options(nostack, preserves_flags)
        );
    }

    if err == 0 { ret } else { ret.wrapping_neg() }
}
