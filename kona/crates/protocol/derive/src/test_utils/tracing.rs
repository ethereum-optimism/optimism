//! This module contains a subscriber layer for `tracing-subscriber` that collects traces and their
//! log levels.

use alloc::{format, string::String, sync::Arc, vec::Vec};
use spin::Mutex;
use tracing::{Event, Level, Subscriber};
use tracing_subscriber::{Layer, layer::Context};

/// The storage for the collected traces.
#[derive(Debug, Default, Clone)]
pub struct TraceStorage(pub Arc<Mutex<Vec<(Level, String)>>>);

impl TraceStorage {
    /// Returns the items in the storage that match the specified level.
    pub fn get_by_level(&self, level: Level) -> Vec<String> {
        self.0
            .lock()
            .iter()
            .filter_map(|(l, message)| if *l == level { Some(message.clone()) } else { None })
            .collect()
    }

    /// Locks the storage and returns the items.
    pub fn lock(&self) -> spin::MutexGuard<'_, Vec<(Level, String)>> {
        self.0.lock()
    }

    /// Returns if the storage is empty.
    pub fn is_empty(&self) -> bool {
        self.0.lock().is_empty()
    }
}

/// A subscriber layer that collects traces and their log levels.
#[derive(Debug, Default)]
pub struct CollectingLayer {
    /// The storage for the collected traces.
    pub storage: TraceStorage,
}

impl CollectingLayer {
    /// Creates a new collecting layer with the specified storage.
    pub const fn new(storage: TraceStorage) -> Self {
        Self { storage }
    }
}

impl<S: Subscriber> Layer<S> for CollectingLayer {
    fn on_event(&self, event: &Event<'_>, _ctx: Context<'_, S>) {
        let metadata = event.metadata();
        let level = *metadata.level();
        let message = format!("{event:?}");

        let mut storage = self.storage.0.lock();
        storage.push((level, message));
    }
}
