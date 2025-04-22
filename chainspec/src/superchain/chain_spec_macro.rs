/// Create a chain spec for a given superchain and environment.
#[macro_export]
macro_rules! create_chain_spec {
    ($name:expr, $environment:expr) => {
        paste::paste! {
            /// The Optimism $name $environment spec
            pub static [<$name:upper _ $environment:upper>]: $crate::LazyLock<alloc::sync::Arc<$crate::OpChainSpec>> = $crate::LazyLock::new(|| {
                $crate::OpChainSpec::from_genesis($crate::superchain::configs::read_superchain_genesis($name, $environment)
                    .expect(&alloc::format!("Can't read {}-{} genesis", $name, $environment)))
                    .into()
            });
        }
    };
}
