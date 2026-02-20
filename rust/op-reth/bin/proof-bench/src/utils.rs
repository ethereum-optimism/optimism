use alloy_primitives::{Address, B256, U256, keccak256};

pub const CONTRACT: &str = "0x4200000000000000000000000000000000000006";

/// Calculate the storage slot for `balanceOf[addr]`
/// Solidity mappings: keccak256(abi.encode(key, slot_position))
pub fn balance_of_slot(addr: Address) -> B256 {
    // Left-pad address (20 bytes) to 32 bytes
    let mut data = Vec::with_capacity(64);
    data.extend_from_slice(&[0u8; 12]);
    data.extend_from_slice(addr.as_slice());

    // Pad slot position (3) to 32 bytes
    let slot_bytes: [u8; 32] = U256::from(3).to_be_bytes();
    data.extend_from_slice(&slot_bytes);

    keccak256(data)
}

pub fn get_addresses() -> Vec<Address> {
    vec![
        "0x48107537B9e358B1894c7a491C17E4bF035AFC74".parse().unwrap(),
        "0x917AbB78953902213F63e16268E78feBAC362846".parse().unwrap(),
        "0xA32Ce4EB5802809EB89032E6cc0FB06EB51bde38".parse().unwrap(),
        "0x8AE9Ed8aB2abF45376cDFb671c05170353dd1F0E".parse().unwrap(),
        "0x2195DbA1ab41966E91C22e4C601Be6517a40f2aB".parse().unwrap(),
        "0x04bF3799798077629cb627DfF76E48a015f0B3CB".parse().unwrap(),
        "0x5aaFa65D234e962121C6f44fd570EE353Ac52Bf5".parse().unwrap(),
        "0x2a58adA546c2e9cd3134c163FBfC0E335Ff91AfA".parse().unwrap(),
        "0x8AE9Ed8aB2abF45376cDFb671c05170353dd1F0E".parse().unwrap(),
        "0x8524771B4c5a8122E8959cFDeB641E3f498188AF".parse().unwrap(),
        "0xf530AD425154CC9635CAaD538e8bf3C638191a4E".parse().unwrap(),
        "0x73a5bB60b0B0fc35710DDc0ea9c407031E31Bdbb".parse().unwrap(),
        "0xfE978E4Dc6f3d716121c603311b0c37a9acd7234".parse().unwrap(),
        "0xcAAd4EB9ABfc93Ab9eA86FB5733B8F85c952200b".parse().unwrap(),
        "0xd15b5531050AC78Aa78AeF8A6DE4256Fa4536107".parse().unwrap(),
    ]
}
