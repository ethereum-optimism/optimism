//! Flags that enable a feature.

/// Parses the value of a flag that enables a feature, such as `--sequencer.enabled`, from its
/// environment variable.
///
/// Declare such a flag with `action = ArgAction::Set`, `num_args = 0`,
/// `default_missing_value = "true"` and this parser, on an `Option<bool>` field. Then the flag
/// takes no value on the command line, and its environment variable must be `true` or unset:
/// clap counts any set variable as given, so a flag set to `false` would still satisfy the
/// `requires` of the flags it enables.
pub fn parse_enabled(value: &str) -> Result<bool, String> {
    match value {
        "true" => Ok(true),
        _ => Err("set the environment variable to `true`, or leave it unset".to_owned()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_only_true() {
        assert_eq!(parse_enabled("true"), Ok(true));
        for value in ["false", "1", "0", "TRUE", "yes", ""] {
            assert!(parse_enabled(value).is_err(), "{value:?}");
        }
    }
}
