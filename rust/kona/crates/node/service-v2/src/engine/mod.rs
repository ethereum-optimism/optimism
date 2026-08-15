//! Semantic execution-engine access and forkchoice reconciliation.
//!
//! This module is the exclusive owner of raw Engine API mutations. Unsafe and safe chain
//! workflows interact with it through narrow semantic operations rather than Engine API calls.
