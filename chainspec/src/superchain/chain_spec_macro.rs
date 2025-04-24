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

/// Generates the key string for a given name and environment pair.
#[macro_export]
macro_rules! key_for {
    ($name:expr, "mainnet") => {
        $name
    };
    ($name:expr, $env:expr) => {
        concat!($name, "-", $env)
    };
}

/// Create chain specs and an enum of every superchain (name, environment) pair.
#[macro_export]
macro_rules! create_superchain_specs {
    ( $( ($name:expr, $env:expr) ),+ $(,)? ) => {
        $(
            $crate::create_chain_spec!($name, $env);
        )+

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

            /// All supported superchains, including both older and newer naming,
            /// for backwards compatibility
            pub const SUPPORTED_CHAINS: &'static [&'static str] = &[
                "dev",
                "optimism",
                "optimism_sepolia",
                "optimism-sepolia",
                "base",
                "base_sepolia",
                "base-sepolia",
                $(
                    $crate::key_for!($name, $env),
                )+
            ];

            /// Parses the chain into an [`$crate::OpChainSpec`], if recognized.
            pub fn generated_chain_value_parser(s: &str) -> Option<alloc::sync::Arc<$crate::OpChainSpec>> {
                match s {
                    "dev"                                   => Some($crate::OP_DEV.clone()),
                    "optimism"                              => Some($crate::OP_MAINNET.clone()),
                    "optimism_sepolia" | "optimism-sepolia" => Some($crate::OP_SEPOLIA.clone()),
                    "base"                                  => Some($crate::BASE_MAINNET.clone()),
                    "base_sepolia" | "base-sepolia"         => Some($crate::BASE_SEPOLIA.clone()),
                    $(
                        $crate::key_for!($name, $env)        => Some($crate::[<$name:upper _ $env:upper>].clone()),
                    )+
                    _                                       => None,
                }
            }
        }
    };
}
