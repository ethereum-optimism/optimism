//! Message safety level for interoperability.
use alloc::string::String;
use thiserror::Error;

/// Error when parsing SafetyLevel from string.
#[derive(Error, Debug)]
#[error("Invalid SafetyLevel, error: {0}")]
pub struct SafetyLevelParseError(pub String);

#[cfg(test)]
mod tests {
    use core::str::FromStr;
    use op_alloy_consensus::interop::SafetyLevel;

    #[test]
    #[cfg(feature = "serde")]
    fn test_safety_level_serde() {
        let level = SafetyLevel::Finalized;
        let json = serde_json::to_string(&level).unwrap();
        assert_eq!(json, r#""finalized""#);

        let level: SafetyLevel = serde_json::from_str(&json).unwrap();
        assert_eq!(level, SafetyLevel::Finalized);
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_serde_safety_level_fails() {
        let json = r#""failed""#;
        let level: Result<SafetyLevel, _> = serde_json::from_str(json);
        assert!(level.is_err());
    }

    #[test]
    fn test_safety_level_from_str_valid() {
        assert_eq!(SafetyLevel::from_str("finalized").unwrap(), SafetyLevel::Finalized);
        assert_eq!(SafetyLevel::from_str("safe").unwrap(), SafetyLevel::CrossSafe);
        assert_eq!(SafetyLevel::from_str("local-safe").unwrap(), SafetyLevel::LocalSafe);
        assert_eq!(SafetyLevel::from_str("localsafe").unwrap(), SafetyLevel::LocalSafe);
        assert_eq!(SafetyLevel::from_str("cross-unsafe").unwrap(), SafetyLevel::CrossUnsafe);
        assert_eq!(SafetyLevel::from_str("crossunsafe").unwrap(), SafetyLevel::CrossUnsafe);
        assert_eq!(SafetyLevel::from_str("unsafe").unwrap(), SafetyLevel::LocalUnsafe);
        assert_eq!(SafetyLevel::from_str("invalid").unwrap(), SafetyLevel::Invalid);
    }

    #[test]
    fn test_safety_level_from_str_invalid() {
        assert!(SafetyLevel::from_str("unknown").is_err());
        assert!(SafetyLevel::from_str("123").is_err());
        assert!(SafetyLevel::from_str("").is_err());
        assert!(SafetyLevel::from_str("safe ").is_err());
    }
}
