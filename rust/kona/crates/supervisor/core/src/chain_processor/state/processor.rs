use kona_interop::DerivedRefPair;

/// This module contains the state management for the chain processor.
/// It provides a way to track the invalidated blocks and manage the state of the chain processor
#[derive(Debug, Default)]
pub struct ProcessorState {
    invalidated_block: Option<DerivedRefPair>,
}

impl ProcessorState {
    /// Creates a new instance of [`ProcessorState`].
    pub fn new() -> Self {
        Self::default()
    }

    /// Returns `true` if the state is invalidated, otherwise `false`.
    pub const fn is_invalidated(&self) -> bool {
        self.invalidated_block.is_some()
    }

    /// Returns the invalidated block if it exists.
    pub const fn get_invalidated(&self) -> Option<DerivedRefPair> {
        self.invalidated_block
    }

    /// Sets the invalidated block to the given pair if it is not already set.
    pub const fn set_invalidated(&mut self, pair: DerivedRefPair) -> bool {
        if self.invalidated_block.is_some() {
            return false; // Already set
        }
        // Set the invalidated block
        self.invalidated_block = Some(pair);
        true
    }

    /// Clears the invalidated block.
    pub const fn clear_invalidated(&mut self) {
        self.invalidated_block = None;
    }
}
