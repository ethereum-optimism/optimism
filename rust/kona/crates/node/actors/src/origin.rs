//! Keeps the sequencer's next L1 origin ready.
//!
//! The sequencer builds each L2 block on an L1 origin: the origin of its parent, or the next L1
//! block once the L2 block's timestamp reaches it. Finding that origin reads L1, which the
//! sequencer, a low-latency component, should not wait on. The origin actor keeps a window of the
//! L1 blocks the sequencer may build on next: the L1 origin of the engine's unsafe head and the
//! canonical, confirmed L1 blocks after it. It follows the engine's unsafe head and the L1 actor,
//! and does all of the L1 reads. [`Handle::next`] answers from the window, and only waits when the
//! parent's origin is not in it, such as right after an engine reset.

mod actor;
pub use actor::ActorError;

mod builder;
pub use builder::Builder;

mod handle;
pub use handle::{Handle, OriginError};

mod window;
