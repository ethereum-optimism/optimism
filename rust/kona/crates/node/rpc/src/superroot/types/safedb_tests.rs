use super::*;
use rstest::rstest;

#[rstest]
#[case::decimal("\"901\"", 901)]
#[case::hex("\"0x385\"", 901)]
fn a_chain_id_is_read_as_decimal_or_hex(#[case] json: &str, #[case] expected: u64) {
    let chain_id: DecimalChainId = serde_json::from_str(json).unwrap();
    assert_eq!(chain_id, DecimalChainId(expected));
}

#[test]
fn a_chain_id_is_written_as_a_decimal_string() {
    assert_eq!(serde_json::to_string(&DecimalChainId(901)).unwrap(), "\"901\"");
}

#[test]
fn a_malformed_chain_id_is_rejected() {
    assert!(serde_json::from_str::<DecimalChainId>("\"0xzz\"").is_err());
}
