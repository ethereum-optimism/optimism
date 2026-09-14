//! Unchanged inbox checksum and access-list encoding (uint64 chain IDs).
use super::{Error, abi::Identifier};
use revm::{
    Database,
    primitives::{B256, Log, U256, keccak256},
};

pub(super) fn access<DB: Database>(id: &Identifier, payload: B256) -> Result<[B256; 2], Error<DB>> {
    let chain: u64 =
        id.chainId.try_into().map_err(|_| Error::Invalid("chain ID exceeds uint64"))?;
    let block: u64 =
        id.blockNumber.try_into().map_err(|_| Error::Invalid("block number exceeds uint64"))?;
    let time: u64 =
        id.timestamp.try_into().map_err(|_| Error::Invalid("timestamp exceeds uint64"))?;
    let index: u32 =
        id.logIndex.try_into().map_err(|_| Error::Invalid("log index exceeds uint32"))?;
    let mut lookup = [0; 32];
    lookup[0] = 1;
    lookup[4..12].copy_from_slice(&chain.to_be_bytes());
    lookup[12..20].copy_from_slice(&block.to_be_bytes());
    lookup[20..28].copy_from_slice(&time.to_be_bytes());
    lookup[28..].copy_from_slice(&index.to_be_bytes());
    let log_hash = keccak256([id.origin.as_slice(), payload.as_slice()].concat());
    let mut packed = [0; 32];
    packed[12..20].copy_from_slice(&block.to_be_bytes());
    packed[20..28].copy_from_slice(&time.to_be_bytes());
    packed[28..].copy_from_slice(&index.to_be_bytes());
    let inner = keccak256([log_hash.as_slice(), &packed].concat());
    let mut checksum = keccak256([inner.as_slice(), &id.chainId.to_be_bytes::<32>()].concat());
    checksum[0] = 3;
    Ok([B256::from(lookup), checksum])
}
pub(super) fn payload(log: &Log) -> B256 {
    keccak256(
        [log.data.topics().iter().flat_map(|t| t.0).collect::<Vec<_>>(), log.data.data.to_vec()]
            .concat(),
    )
}
pub(super) const fn word(value: B256) -> U256 {
    U256::from_be_bytes(value.0)
}

#[cfg(test)]
mod tests {
    use super::*;
    use revm::{
        database::InMemoryDB,
        primitives::{address, b256},
    };
    #[test]
    fn matches_go_message_and_access_list_golden_vector() {
        // Shared values from op-core/interop/messages/messages_test.go,
        // TestMessage/TestEncodeAccessList.
        let id = Identifier {
            origin: address!("e0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9"),
            blockNumber: U256::from(0xa1a2_a3a4_a5a6_a7a8u64),
            logIndex: U256::from(0xb1b2_b3b4u32),
            timestamp: U256::from(0xc1c2_c3c4_c5c6_c7c8u64),
            chainId: U256::from(0xd1d2_d3d4_d5d6_d7d8u64),
        };
        assert_eq!(
            access::<InMemoryDB>(
                &id,
                b256!("8017559a85b12c04b14a1a425d53486d1015f833714a09bd62f04152a7e2ae9b")
            )
            .unwrap(),
            [
                b256!("01000000d1d2d3d4d5d6d7d8a1a2a3a4a5a6a7a8c1c2c3c4c5c6c7c8b1b2b3b4"),
                b256!("03749e87fd7789575de9906569deb05aaf220dc4cfab3d8abbfd34a2e1d7d357")
            ]
        );
    }
}
