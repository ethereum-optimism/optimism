//! Macros used across test utilities.

/// A shorthand syntax for constructing [kona_protocol::Frame]s.
#[macro_export]
macro_rules! frame {
    ($id:expr, $number:expr, $data:expr, $is_last:expr) => {
        kona_protocol::Frame { id: [$id; 16], number: $number, data: $data, is_last: $is_last }
    };
}

/// A shorthand syntax for constructing a list of [kona_protocol::Frame]s.
#[macro_export]
macro_rules! frames {
    ($id:expr, $number:expr, $data:expr, $count:expr) => {{
        let mut frames = vec![$crate::frame!($id, $number, $data, false); $count];
        frames[$count - 1].is_last = true;
        frames
    }};
}
