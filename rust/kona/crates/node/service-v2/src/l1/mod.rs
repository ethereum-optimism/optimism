//! Shared L1 data access.
//!
//! Unsafe and safe chain processing consume the same canonical L1 source while maintaining
//! independent origins and progress cursors.
