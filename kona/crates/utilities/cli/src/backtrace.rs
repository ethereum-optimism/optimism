//! Helper to set the backtrace env var.

/// Sets the RUST_BACKTRACE environment variable to 1 if it is not already set.
pub fn enable() {
    // Enable backtraces unless a RUST_BACKTRACE value has already been explicitly provided.
    if std::env::var_os("RUST_BACKTRACE").is_none() {
        // We accept the risk that another process may set RUST_BACKTRACE at the same time.
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }
}
