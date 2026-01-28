#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod traits;
pub use traits::Hardfork;

mod forks;
pub use forks::Hardforks;

mod fjord;
pub use fjord::Fjord;

mod ecotone;
pub use ecotone::Ecotone;

mod isthmus;
pub use isthmus::Isthmus;

mod interop;
pub use interop::Interop;

mod jovian;
pub use jovian::Jovian;

mod utils;
pub(crate) use utils::upgrade_to_calldata;

#[cfg(test)]
mod test_utils;
