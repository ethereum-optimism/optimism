/// A wrapper around `u64` that supports hex string (e.g. `"0x1"`) or numeric deserialization
/// for RPC inputs.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct HexStringU64(pub u64);

impl serde::Serialize for HexStringU64 {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        alloy_serde::quantity::serialize(&self.0, serializer)
    }
}

impl<'de> serde::Deserialize<'de> for HexStringU64 {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let inner = alloy_serde::quantity::deserialize(deserializer)?;
        Ok(Self(inner))
    }
}

impl From<HexStringU64> for u64 {
    fn from(value: HexStringU64) -> Self {
        value.0
    }
}

impl From<u64> for HexStringU64 {
    fn from(value: u64) -> Self {
        Self(value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_deserialize_from_hex_string() {
        let json = r#""0x1a""#;
        let parsed: HexStringU64 = serde_json::from_str(json).expect("should parse hex string");
        let chain_id: u64 = parsed.0;
        assert_eq!(chain_id, 0x1a);
    }

    #[test]
    fn test_serialize_to_hex() {
        let value = HexStringU64(26);
        let json = serde_json::to_string(&value).expect("should serialize");
        assert_eq!(json, r#""0x1a""#);
    }

    #[test]
    fn test_round_trip() {
        let original = HexStringU64(12345);
        let json = serde_json::to_string(&original).unwrap();
        let parsed: HexStringU64 = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.0, original.0);
    }
}
