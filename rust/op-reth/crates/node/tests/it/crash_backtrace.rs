//! Prints a backtrace when the test process dies on a fatal signal.
//!
//! A binary killed by a signal produces no Rust backtrace — `RUST_BACKTRACE` only
//! covers panics — so a crash that happens *after* a test body has passed leaves
//! nothing to diagnose beyond the signal number. Installing a handler recovers the
//! faulting thread's name and stack, which is the evidence needed to tell apart the
//! candidate causes of a teardown crash (a database's `atexit` destructor racing its
//! own live threads, background threads outliving process exit, and so on).
//!
//! The handler re-raises with the default disposition, so the process still dies with
//! the original signal and the test runner still records a crash rather than a pass.
//!
//! Installing this replaces Rust std's own SIGSEGV/SIGBUS handler, so a genuine stack
//! overflow in this binary prints the output below rather than the usual
//! `thread '...' has overflowed its stack` — still diagnosable (the fault lands by the
//! guard page and the frames show the recursion), just different.

use std::sync::Once;

static INIT: Once = Once::new();

/// Installs the fatal-signal handler once per process.
///
/// Call at the start of a test that launches a full node. The handler stays installed
/// for the lifetime of the process, so it also covers teardown after the test body has
/// returned — which is when such crashes typically occur.
pub(crate) fn install() {
    INIT.call_once(install_inner);
}

/// Seconds allowed for backtrace capture before the watchdog kills the process.
const CAPTURE_WATCHDOG_SECS: libc::c_uint = 10;

/// Frames captured on a fault. Deep enough for a teardown stack through a runtime.
const MAX_FRAMES: usize = 64;

fn install_inner() {
    unsafe {
        // The first `backtrace()` call loads libgcc's unwinder and may allocate. Doing it
        // here means the call inside the handler does not have to.
        let mut warmup = [std::ptr::null_mut::<libc::c_void>(); 1];
        libc::backtrace(warmup.as_mut_ptr(), warmup.len() as libc::c_int);

        let mut action: libc::sigaction = std::mem::zeroed();
        action.sa_sigaction = handler as *const () as usize;
        action.sa_flags = libc::SA_SIGINFO | libc::SA_ONSTACK;
        libc::sigemptyset(&raw mut action.sa_mask);

        // SIGABRT is included because heap corruption in a teardown thread — a suspected
        // cause here — often surfaces as glibc `free(): invalid pointer` -> abort() rather
        // than a segfault, and dies just as stackless. The re-raise path handles it the same.
        for sig in [libc::SIGSEGV, libc::SIGBUS, libc::SIGILL, libc::SIGFPE, libc::SIGABRT] {
            libc::sigaction(sig, &raw const action, std::ptr::null_mut());
        }
    }
}

/// Writes directly to fd 2. `println!`/`eprintln!` take a lock and allocate, neither of
/// which is legal in a signal handler; `write(2)` is async-signal-safe.
fn write_stderr(bytes: &[u8]) {
    unsafe {
        libc::write(libc::STDERR_FILENO, bytes.as_ptr().cast(), bytes.len());
    }
}

fn write_u64(mut n: u64) {
    let mut buf = [0u8; 20];
    let mut i = buf.len();
    loop {
        i -= 1;
        buf[i] = b'0' + (n % 10) as u8;
        n /= 10;
        if n == 0 {
            break;
        }
    }
    write_stderr(&buf[i..]);
}

extern "C" fn handler(sig: libc::c_int, info: *mut libc::siginfo_t, _ctx: *mut libc::c_void) {
    // Everything up to the watchdog is async-signal-safe, so this much is emitted even
    // if the richer capture below wedges.
    write_stderr(b"\n=== fatal signal ");
    write_u64(sig as u64);
    write_stderr(b" in thread '");
    write_thread_name();
    write_stderr(b"'");

    if !info.is_null() {
        write_stderr(b" faulting address 0x");
        let addr = unsafe { (*info).si_addr() } as usize;
        write_hex(addr as u64);
    }
    write_stderr(b" ===\n");

    // Belt and suspenders: `backtrace()` is warmed up at install time so it should not
    // allocate here, but a wedge would otherwise hang the job until the CI timeout.
    unsafe {
        libc::signal(libc::SIGALRM, libc::SIG_DFL);
        libc::alarm(CAPTURE_WATCHDOG_SECS);
    }

    // `backtrace_symbols_fd` writes straight to the fd and is documented not to call
    // malloc, unlike `std::backtrace::Backtrace`, whose symbolization allocates heavily
    // and faults a second time when called from here — killing the process before it can
    // print anything, since SIGSEGV is blocked inside its own handler.
    //
    // The tradeoff is resolution: this yields function names and addresses but no line
    // numbers or inlined frames. Addresses can be resolved offline with addr2line against
    // the same binary when more detail is needed.
    unsafe {
        let mut frames = [std::ptr::null_mut::<libc::c_void>(); MAX_FRAMES];
        let n = libc::backtrace(frames.as_mut_ptr(), frames.len() as libc::c_int);
        libc::backtrace_symbols_fd(frames.as_ptr(), n, libc::STDERR_FILENO);
    }

    // Now that the reliable frames are already on the wire, try for names and line
    // numbers as a bonus. This is the step that can fault; if it does, the process dies
    // with the original signal anyway and nothing above is lost.
    write_stderr(b"--- symbolized (best effort; empty here means symbolization faulted) ---\n");
    write_stderr(std::backtrace::Backtrace::force_capture().to_string().as_bytes());
    write_stderr(b"\n");

    // Die with the original signal so the runner still sees a crash. Restoring the
    // default disposition first prevents recursing back into this handler.
    unsafe {
        libc::alarm(0);
        libc::signal(sig, libc::SIG_DFL);
        libc::raise(sig);
    }
}

fn write_hex(n: u64) {
    const DIGITS: &[u8; 16] = b"0123456789abcdef";
    let mut buf = [0u8; 16];
    for (i, slot) in buf.iter_mut().enumerate() {
        *slot = DIGITS[((n >> (60 - i * 4)) & 0xf) as usize];
    }
    write_stderr(&buf);
}

/// The faulting thread's name is the single most useful field here: it distinguishes a
/// crash in, say, a database transaction-manager thread from one in a runtime worker.
///
/// Writes straight from a stack buffer — building a `String` would allocate, which is
/// not legal before the watchdog below is armed.
fn write_thread_name() {
    // `libc::c_char` (not `i8`): its signedness is target-dependent — `u8` on
    // aarch64-linux — and hardcoding `i8` fails to compile there.
    let mut buf: [libc::c_char; 32] = [0; 32];
    let rc = unsafe { libc::pthread_getname_np(libc::pthread_self(), buf.as_mut_ptr(), buf.len()) };
    if rc != 0 {
        write_stderr(b"<unknown>");
        return;
    }
    let len = buf.iter().position(|&c| c == 0).unwrap_or(buf.len());
    let bytes = unsafe { std::slice::from_raw_parts(buf.as_ptr().cast::<u8>(), len) };
    write_stderr(bytes);
}
