//! Constants for the OP Stack interop protocol.

/// Re-export of the interop message expiry window. Now defined in `kona-genesis`.
pub use kona_genesis::MESSAGE_EXPIRY_WINDOW;

/// The current version of the [`SuperRoot`](crate::SuperRoot) encoding format.
pub const SUPER_ROOT_VERSION: u8 = 1;
