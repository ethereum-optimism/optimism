//! Constants for OP Stack interop primitives shared with the registry.

/// The expiry window for relaying an initiating message (in seconds).
/// <https://specs.optimism.io/interop/messaging.html#message-expiry-invariant>
pub const MESSAGE_EXPIRY_WINDOW: u64 = 7 * 24 * 60 * 60;
