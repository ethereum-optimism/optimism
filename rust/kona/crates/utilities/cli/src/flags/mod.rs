//! Common CLI Flags
//!
//! These are cli flags that are shared across binaries to standardize kona's services CLI UX.

mod log;
pub use log::{LogArgs, LogLevel};

/// Returns the environment variable a flag reads: `prefix`, then the flag's long name in upper
/// case with `.` and `-` replaced by `_`. For example, with the prefix `KONA_NODE_`,
/// `--l1.eth-url` reads `KONA_NODE_L1_ETH_URL`.
///
/// Each flag names its variable explicitly, so that it can be found in the source; binaries test
/// that the names follow this rule.
pub fn env_var_for_flag(prefix: &str, long: &str) -> String {
    format!("{prefix}{}", long.to_uppercase().replace(['.', '-'], "_"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{Args, Command};

    #[test]
    fn shared_flags_read_the_environment_variables_named_after_them() {
        let command = LogArgs::augment_args(Command::new("test"));
        let mismatches: Vec<_> = command
            .get_arguments()
            .filter_map(|arg| {
                let long = arg.get_long()?;
                let expected = env_var_for_flag("KONA_", long);
                let actual = arg.get_env().map(|env| env.to_string_lossy().into_owned());
                (actual.as_deref() != Some(expected.as_str()))
                    .then(|| format!("--{long}: {actual:?}, expected {expected}"))
            })
            .collect();
        assert!(mismatches.is_empty(), "{mismatches:#?}");
    }
}
