//! A deny list for tests, answering from a fixed set or failing every read.

use crate::{DenyList, DenyListReadError};
use alloy_primitives::B256;
use std::sync::{Arc, RwLock};

/// A [`DenyList`] over a fixed set of `(number, hash)` entries, recording every query.
#[derive(Debug, Default)]
pub struct StaticDenyList {
    /// The denied blocks.
    denied: RwLock<Vec<(u64, B256)>>,
    /// When set, every read fails with this message.
    unreadable: RwLock<Option<String>>,
    /// Every `(number, hash)` asked of [`DenyList::is_denied`], in order.
    queries: RwLock<Vec<(u64, B256)>>,
}

impl StaticDenyList {
    /// A deny list denying exactly `entries`.
    pub fn denying(entries: impl IntoIterator<Item = (u64, B256)>) -> Arc<Self> {
        Arc::new(Self {
            denied: RwLock::new(entries.into_iter().collect()),
            unreadable: RwLock::new(None),
            queries: RwLock::new(Vec::new()),
        })
    }

    /// A deny list whose every read fails with `message`.
    pub fn unreadable(message: &str) -> Arc<Self> {
        Arc::new(Self {
            denied: RwLock::new(Vec::new()),
            unreadable: RwLock::new(Some(message.to_string())),
            queries: RwLock::new(Vec::new()),
        })
    }

    /// Every `(number, hash)` asked so far.
    pub fn queries(&self) -> Vec<(u64, B256)> {
        self.queries.read().unwrap().clone()
    }
}

impl DenyList for StaticDenyList {
    fn is_denied(&self, number: u64, hash: B256) -> Result<bool, DenyListReadError> {
        self.queries.write().unwrap().push((number, hash));
        if let Some(message) = self.unreadable.read().unwrap().as_ref() {
            return Err(DenyListReadError(message.clone()));
        }
        Ok(self.denied.read().unwrap().contains(&(number, hash)))
    }

    fn max_denied_height(&self) -> Result<Option<u64>, DenyListReadError> {
        if let Some(message) = self.unreadable.read().unwrap().as_ref() {
            return Err(DenyListReadError(message.clone()));
        }
        Ok(self.denied.read().unwrap().iter().map(|(number, _)| *number).max())
    }
}
