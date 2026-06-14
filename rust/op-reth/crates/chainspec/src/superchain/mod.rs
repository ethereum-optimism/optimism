//! Support for superchain registry.

mod chain_metadata;
mod chain_spec_macro;
mod chain_specs;
mod configs;

pub use chain_specs::*;
/// Re-exported so the well-known OP chain specs ([`crate::OP_MAINNET`], [`crate::OP_SEPOLIA`]) can be
/// built from the embedded registry instead of a hand-coded fork list.
pub(crate) use configs::read_superchain_genesis;

#[cfg(test)]
mod tests {
    use super::Superchain;

    #[test]
    fn round_trip_superchain_enum_name_and_env() {
        for &chain in Superchain::ALL {
            let name = chain.name();
            let env = chain.environment();

            assert!(!name.is_empty(), "name() must not be empty");
            assert!(!env.is_empty(), "environment() must not be empty");
        }
    }

    #[test]
    fn superchain_enum_has_funki_mainnet() {
        assert!(
            Superchain::ALL.iter().any(|&c| c.name() == "funki" && c.environment() == "mainnet"),
            "Expected funki/mainnet in ALL"
        );
    }
}
