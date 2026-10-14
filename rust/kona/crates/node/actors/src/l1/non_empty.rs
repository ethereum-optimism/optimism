//! A double-ended queue that is never empty.

use std::{
    collections::{VecDeque, vec_deque},
    ops::Index,
};

/// A [`VecDeque`] that always holds at least one element.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) struct NonEmptyVecDeque<T>(VecDeque<T>);

impl<T> NonEmptyVecDeque<T> {
    /// Creates a queue holding only `first`.
    pub(super) fn new(first: T) -> Self {
        Self(VecDeque::from([first]))
    }

    /// Returns the first element.
    pub(super) fn front(&self) -> &T {
        self.0.front().expect("the queue is never empty")
    }

    /// Returns the last element.
    pub(super) fn back(&self) -> &T {
        self.0.back().expect("the queue is never empty")
    }

    /// Returns the element at `index`, if there is one.
    pub(super) fn get(&self, index: usize) -> Option<&T> {
        self.0.get(index)
    }

    /// Returns the elements, front to back.
    pub(super) fn iter(&self) -> vec_deque::Iter<'_, T> {
        self.0.iter()
    }

    /// Appends an element.
    pub(super) fn push_back(&mut self, value: T) {
        self.0.push_back(value);
    }

    /// Removes the first element, unless it is the only one.
    pub(super) fn pop_front(&mut self) -> Option<T> {
        if self.0.len() > 1 { self.0.pop_front() } else { None }
    }

    /// Keeps the elements up to and including `index`, and returns the rest, front to back.
    pub(super) fn split_after(&mut self, index: usize) -> VecDeque<T> {
        let at = index.saturating_add(1).min(self.0.len());
        self.0.split_off(at)
    }
}

impl<T> Index<usize> for NonEmptyVecDeque<T> {
    type Output = T;

    fn index(&self, index: usize) -> &T {
        &self.0[index]
    }
}

impl<T> IntoIterator for NonEmptyVecDeque<T> {
    type Item = T;
    type IntoIter = vec_deque::IntoIter<T>;

    fn into_iter(self) -> Self::IntoIter {
        self.0.into_iter()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn never_removes_the_last_element() {
        let mut queue = NonEmptyVecDeque::new(1);
        queue.push_back(2);
        assert_eq!(queue.pop_front(), Some(1));
        assert_eq!(queue.pop_front(), None);
        assert_eq!((*queue.front(), *queue.back()), (2, 2));
    }

    #[test]
    fn splitting_keeps_the_front() {
        let mut queue = NonEmptyVecDeque::new(1);
        queue.push_back(2);
        queue.push_back(3);
        assert_eq!(queue.split_after(0), VecDeque::from([2, 3]));
        assert_eq!(queue.into_iter().collect::<Vec<_>>(), [1]);

        let mut queue = NonEmptyVecDeque::new(1);
        assert!(queue.split_after(5).is_empty());
        assert_eq!(*queue.back(), 1);
    }
}
