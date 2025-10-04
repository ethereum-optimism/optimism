use std::{error, fmt};
use tower::BoxError;

/// This error is substitution for tower Elapsed error, because it's pub(super) and cannot be used
#[derive(Debug, Default)]
pub struct Elapsed;

impl fmt::Display for Elapsed {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.pad("request timed out")
    }
}

impl error::Error for Elapsed {}

#[derive(Debug)]
pub enum HyperError {
    Hyper(Box<hyper::Error>),
    Timeout,
}

impl fmt::Display for HyperError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Hyper(e) => e.fmt(f),
            Self::Timeout => Elapsed.fmt(f),
        }
    }
}

impl error::Error for HyperError {
    fn source(&self) -> Option<&(dyn error::Error + 'static)> {
        match self {
            Self::Hyper(e) => e.source(),
            Self::Timeout => Elapsed.source(),
        }
    }
}

impl From<BoxError> for HyperError {
    fn from(value: BoxError) -> Self {
        value
            .downcast::<hyper::Error>()
            .map(HyperError::Hyper)
            .unwrap_or_else(|_| HyperError::Timeout)
    }
}
