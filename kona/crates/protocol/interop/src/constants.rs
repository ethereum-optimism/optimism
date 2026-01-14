//! Constants for the OP Stack interop protocol.

/// The expiry window for relaying an initiating message (in seconds).
/// <https://specs.optimism.io/interop/messaging.html#message-expiry-invariant>
pub const MESSAGE_EXPIRY_WINDOW: u64 = 7 * 24 * 60 * 60;

/// The current version of the [SuperRoot] encoding format.
///
/// [SuperRoot]: crate::SuperRoot
pub const SUPER_ROOT_VERSION: u8 = 1;
