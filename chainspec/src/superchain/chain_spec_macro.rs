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

/// Create an enum of every superchain (name, environment) pair.
#[macro_export]
macro_rules! create_superchain_enum {
    ( $( ($name:expr, $env:expr) ),+ $(,)? ) => {
        paste::paste! {
            /// All available superchains as an enum
            #[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
            #[allow(non_camel_case_types)]
            pub enum Superchain {
                $(
                    #[doc = concat!("Superchain variant for `", $name, "-", $env, "`.")]
                    [<$name:camel _ $env:camel>],
                )+
            }

            impl Superchain {
                /// A slice of every superchain enum variant
                pub const ALL: &'static [Self] = &[
                    $(
                        Self::[<$name:camel _ $env:camel>],
                    )+
                ];

                /// Returns the original name
                pub const fn name(self) -> &'static str {
                    match self {
                        $(
                            Self::[<$name:camel _ $env:camel>] => $name,
                        )+
                    }
                }

                /// Returns the original environment
                pub const fn environment(self) -> &'static str {
                    match self {
                        $(
                            Self::[<$name:camel _ $env:camel>] => $env,
                        )+
                    }
                }
            }
        }
    };
}
