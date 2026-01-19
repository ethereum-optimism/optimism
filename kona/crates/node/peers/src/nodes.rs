//! Bootnodes for consensus network discovery.

use crate::BootNode;
use derive_more::Deref;
use lazy_static::lazy_static;

use kona_registry::CHAINS;

/// Bootnodes for OP Stack chains.
#[derive(Debug, Clone, Deref, PartialEq, Eq, Default, derive_more::From)]
pub struct BootNodes(pub Vec<BootNode>);

impl BootNodes {
    /// Returns the bootnodes for the given chain id.
    ///
    /// If the chain id is not recognized, no bootnodes are returned.
    pub fn from_chain_id(id: u64) -> Self {
        let Some(chain) = CHAINS.get_chain_by_id(id) else {
            return Self(vec![]);
        };
        match chain.parent.chain_id() {
            1 => Self::mainnet(),
            11155111 => Self::testnet(),
            _ => Self(vec![]),
        }
    }

    /// Returns the bootnodes for the mainnet.
    pub fn mainnet() -> Self {
        Self(OP_BOOTNODES.clone())
    }

    /// Returns the bootnodes for the testnet.
    pub fn testnet() -> Self {
        Self(OP_TESTNET_BOOTNODES.clone())
    }

    /// Returns the length of the bootnodes.
    pub const fn len(&self) -> usize {
        self.0.len()
    }

    /// Returns if the bootnodes are empty.
    pub const fn is_empty(&self) -> bool {
        self.0.is_empty()
    }
}

lazy_static! {
    /// Default op bootnodes to use.
    static ref OP_BOOTNODES: Vec<BootNode> = OP_RAW_BOOTNODES.iter()
        .map(|raw| BootNode::parse_bootnode(raw))
        .collect();

    /// Default op testnet bootnodes to use.
    static ref OP_TESTNET_BOOTNODES: Vec<BootNode> = OP_RAW_TESTNET_BOOTNODES.iter()
        .map(|raw| BootNode::parse_bootnode(raw))
        .collect();
}

/// OP stack mainnet boot nodes.
pub static OP_RAW_BOOTNODES: &[&str] = &[
    // OP Mainnet Bootnodes
    "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg",
    "enr:-J64QAlTCDa188Hl1OGv5_2Kj2nWCsvxMVc_rEnLtw7RPFbOfqUOV6khXT_PH6cC603I2ynY31rSQ8sI9gLeJbfFGaWGAYYFIrpdgmlkgnY0gmlwhANWgzCHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECkySjcg-2v0uWAsFsZZu43qNHppGr2D5F913Qqs5jDCGDdGNwgiQGg3VkcIIkBg",
    "enr:-J24QGEzN4mJgLWNTUNwj7riVJ2ZjRLenOFccl2dbRFxHHOCCZx8SXWzgf-sLzrGs6QgqSFCvGXVgGPBkRkfOWlT1-iGAYe6Cu93gmlkgnY0gmlwhCJBEUSHb3BzdGFja4OkAwCJc2VjcDI1NmsxoQLuYIwaYOHg3CUQhCkS-RsSHmUd1b_x93-9yQ5ItS6udIN0Y3CCIyuDdWRwgiMr",
    // Base Mainnet Bootnodes
    "enr:-J24QNz9lbrKbN4iSmmjtnr7SjUMk4zB7f1krHZcTZx-JRKZd0kA2gjufUROD6T3sOWDVDnFJRvqBBo62zuF-hYCohOGAYiOoEyEgmlkgnY0gmlwhAPniryHb3BzdGFja4OFQgCJc2VjcDI1NmsxoQKNVFlCxh_B-716tTs-h1vMzZkSs1FTu_OYTNjgufplG4N0Y3CCJAaDdWRwgiQG",
    "enr:-J24QH-f1wt99sfpHy4c0QJM-NfmsIfmlLAMMcgZCUEgKG_BBYFc6FwYgaMJMQN5dsRBJApIok0jFn-9CS842lGpLmqGAYiOoDRAgmlkgnY0gmlwhLhIgb2Hb3BzdGFja4OFQgCJc2VjcDI1NmsxoQJ9FTIv8B9myn1MWaC_2lJ-sMoeCDkusCsk4BYHjjCq04N0Y3CCJAaDdWRwgiQG",
    "enr:-J24QDXyyxvQYsd0yfsN0cRr1lZ1N11zGTplMNlW4xNEc7LkPXh0NAJ9iSOVdRO95GPYAIc6xmyoCCG6_0JxdL3a0zaGAYiOoAjFgmlkgnY0gmlwhAPckbGHb3BzdGFja4OFQgCJc2VjcDI1NmsxoQJwoS7tzwxqXSyFL7g0JM-KWVbgvjfB8JA__T7yY_cYboN0Y3CCJAaDdWRwgiQG",
    "enr:-J24QHmGyBwUZXIcsGYMaUqGGSl4CFdx9Tozu-vQCn5bHIQbR7On7dZbU61vYvfrJr30t0iahSqhc64J46MnUO2JvQaGAYiOoCKKgmlkgnY0gmlwhAPnCzSHb3BzdGFja4OFQgCJc2VjcDI1NmsxoQINc4fSijfbNIiGhcgvwjsjxVFJHUstK9L1T8OTKUjgloN0Y3CCJAaDdWRwgiQG",
    "enr:-J24QG3ypT4xSu0gjb5PABCmVxZqBjVw9ca7pvsI8jl4KATYAnxBmfkaIuEqy9sKvDHKuNCsy57WwK9wTt2aQgcaDDyGAYiOoGAXgmlkgnY0gmlwhDbGmZaHb3BzdGFja4OFQgCJc2VjcDI1NmsxoQIeAK_--tcLEiu7HvoUlbV52MspE0uCocsx1f_rYvRenIN0Y3CCJAaDdWRwgiQG",
    // OP Labs
    "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305",
    "enode://dd751a9ef8912be1bfa7a5e34e2c3785cc5253110bd929f385e07ba7ac19929fb0e0c5d93f77827291f4da02b2232240fbc47ea7ce04c46e333e452f8656b667@34.65.107.0:30305",
    "enode://c5d289b56a77b6a2342ca29956dfd07aadf45364dde8ab20d1dc4efd4d1bc6b4655d902501daea308f4d8950737a4e93a4dfedd17b49cd5760ffd127837ca965@34.65.202.239:30305",
    // Base
    "enode://87a32fd13bd596b2ffca97020e31aef4ddcc1bbd4b95bb633d16c1329f654f34049ed240a36b449fda5e5225d70fe40bc667f53c304b71f8e68fc9d448690b51@3.231.138.188:30301",
    "enode://ca21ea8f176adb2e229ce2d700830c844af0ea941a1d8152a9513b966fe525e809c3a6c73a2c18a12b74ed6ec4380edf91662778fe0b79f6a591236e49e176f9@184.72.129.189:30301",
    "enode://acf4507a211ba7c1e52cdf4eef62cdc3c32e7c9c47998954f7ba024026f9a6b2150cd3f0b734d9c78e507ab70d59ba61dfe5c45e1078c7ad0775fb251d7735a2@3.220.145.177:30301",
    "enode://8a5a5006159bf079d06a04e5eceab2a1ce6e0f721875b2a9c96905336219dbe14203d38f70f3754686a6324f786c2f9852d8c0dd3adac2d080f4db35efc678c5@3.231.11.52:30301",
    "enode://cdadbe835308ad3557f9a1de8db411da1a260a98f8421d62da90e71da66e55e98aaa8e90aa7ce01b408a54e4bd2253d701218081ded3dbe5efbbc7b41d7cef79@54.198.153.150:30301",
    // Conduit
    // "enode://d25ce99435982b04d60c4b41ba256b84b888626db7bee45a9419382300fbe907359ae5ef250346785bff8d3b9d07cd3e017a27e2ee3cfda3bcbb0ba762ac9674@bootnode.conduit.xyz:0?discport=30301",
    "enode://2d4e7e9d48f4dd4efe9342706dd1b0024681bd4c3300d021f86fc75eab7865d4e0cbec6fbc883f011cfd6a57423e7e2f6e104baad2b744c3cafaec6bc7dc92c1@34.65.43.171:0?discport=30305",
    "enode://9d7a3efefe442351217e73b3a593bcb8efffb55b4807699972145324eab5e6b382152f8d24f6301baebbfb5ecd4127bd3faab2842c04cd432bdf50ba092f6645@34.65.109.126:0?discport=30305",
    // Uniswap Labs
    "enode://010800c668896c100e8d64abc388ac5a22a8134a96fb0107c5d0c56d79ba7225c12d9e9e012d3cc0ee2701d7f63dd45f8abf0bbcf6f3c541f91742b1d7a99355@3.134.214.169:9222",
    "enode://b97abcc7011d06299c4bc44742be4a0e631a1a2925a2992adcfe80ed86bec5ff0ddf1b90d015f2dbb5e305560e12c9873b2dad72d84d131ac4be9f2a4c74b763@52.14.30.39:9222",
    "enode://760230a662610620d6d2e4ad846a6dccbceaa4556872dfacf9cdca7c2f5b49e4c66e822ed2e8813debb5fb7391f0519b8d075e565a2a89c79a9e4092e81b3e5b@3.148.100.173:9222",
    "enode://b1a743328188dba3b2ed8c06abbb2688fabe64a3251e43bd77d4e5265bbd5cf03eca8ace4cde8ddb0c49c409b90bf941ebf556094638c6203edd6baa5ef0091b@3.134.214.169:30303",
    "enode://ea9eaaf695facbe53090beb7a5b0411a81459bbf6e6caac151e587ee77120a1b07f3b9f3a9550f797d73d69840a643b775fd1e40344dea11e7660b6a483fe80e@52.14.30.39:30303",
    "enode://77b6b1e72984d5d50e00ae934ffea982902226fe92fa50da42334c2750d8e405b55a5baabeb988c88125368142a64eda5096d0d4522d3b6eef75d166c7d303a9@3.148.100.173:30303",
];

/// OP stack testnet boot nodes.
pub static OP_RAW_TESTNET_BOOTNODES: &[&str] = &[
    // OP Labs
    "enode://2bd2e657bb3c8efffb8ff6db9071d9eb7be70d7c6d7d980ff80fc93b2629675c5f750bc0a5ef27cd788c2e491b8795a7e9a4a6e72178c14acc6753c0e5d77ae4@34.65.205.244:30305",
    "enode://db8e1cab24624cc62fc35dbb9e481b88a9ef0116114cd6e41034c55b5b4f18755983819252333509bd8e25f6b12aadd6465710cd2e956558faf17672cce7551f@34.65.173.88:30305",
    "enode://bfda2e0110cfd0f4c9f7aa5bf5ec66e6bd18f71a2db028d36b8bf8b0d6fdb03125c1606a6017b31311d96a36f5ef7e1ad11604d7a166745e6075a715dfa67f8a@34.65.229.245:30305",
    // Base
    "enode://548f715f3fc388a7c917ba644a2f16270f1ede48a5d88a4d14ea287cc916068363f3092e39936f1a3e7885198bef0e5af951f1d7b1041ce8ba4010917777e71f@18.210.176.114:30301",
    "enode://6f10052847a966a725c9f4adf6716f9141155b99a0fb487fea3f51498f4c2a2cb8d534e680ee678f9447db85b93ff7c74562762c3714783a7233ac448603b25f@107.21.251.55:30301",
    // Uniswap Labs
    "enode://9e138a8ec4291c4f2fe5851aaee44fc73ae67da87fb26b75e3b94183c7ffc15b2795afc816b0aa084151b95b3a3553f1cd0d1e9dd134dcf059a84d4e0b429afc@3.146.117.118:30303",
    "enode://34d87d649e5c58a17a43c1d59900a2020bd82d5b12ea39467c3366bee2946aaa9c759c77ede61089624691291fb2129eeb2a47687b50e2463188c78e1f738cf2@52.15.54.8:30303",
    "enode://c2405194166fe2c0e6c61ee469745fed1a6802f51c8fc39e1c78c21c9a6a15a7c55304f09ee37e430da9a1ce8117ca085263c6b0f474f6946811e398347611ef@3.146.213.65:30303",
];

#[cfg(test)]
mod tests {
    use discv5::{Enr, enr::EnrPublicKey};
    use std::str::FromStr;

    use kona_genesis::{BASE_MAINNET_CHAIN_ID, OP_MAINNET_CHAIN_ID, OP_SEPOLIA_CHAIN_ID};

    use super::*;

    #[test]
    fn test_validate_bootnode_lens() {
        assert_eq!(OP_RAW_BOOTNODES.len(), 24);
        assert_eq!(OP_RAW_TESTNET_BOOTNODES.len(), 8);
    }

    #[test]
    fn test_parse_raw_bootnodes() {
        for raw in OP_RAW_BOOTNODES.iter() {
            BootNode::parse_bootnode(raw);
        }

        for raw in OP_RAW_TESTNET_BOOTNODES.iter() {
            BootNode::parse_bootnode(raw);
        }
    }

    #[test]
    fn test_bootnodes_from_chain_id() {
        let mainnet = BootNodes::from_chain_id(OP_MAINNET_CHAIN_ID);
        assert_eq!(mainnet.len(), 24);

        let mainnet = BootNodes::from_chain_id(BASE_MAINNET_CHAIN_ID);
        assert_eq!(mainnet.len(), 24);

        let mainnet = BootNodes::from_chain_id(130 /* Unichain Mainnet */);
        assert_eq!(mainnet.len(), 24);

        let testnet = BootNodes::from_chain_id(OP_SEPOLIA_CHAIN_ID);
        assert_eq!(testnet.len(), 8);

        let testnet = BootNodes::from_chain_id(1301 /* Unichain Sepolia */);
        assert_eq!(testnet.len(), 8);

        let unknown = BootNodes::from_chain_id(0);
        assert!(unknown.is_empty());
    }

    #[test]
    fn test_bootnodes_len() {
        let bootnodes = BootNodes::mainnet();
        assert_eq!(bootnodes.len(), 24);

        let bootnodes = BootNodes::testnet();
        assert_eq!(bootnodes.len(), 8);
    }

    #[test]
    fn parse_enr() {
        const ENR: &str = "enr:-Jy4QHRgJ9rnWbTs0oOfv8IHt77NDhHE3rwXf3fCh8RRN8sze4gyuQ2MkAapZwneDd_LH77TGCRS5N4wPGm-J5Hh-oCDAQKOgmlkgnY0gmlwhC36_pOHb3BzdGFja4Xkq4MBAIlzZWNwMjU2azGhAqUtGspoH5IzIIAwaqcipQFWripEU12KAiKFqRKDCZWxg3RjcIIjK4N1ZHCCIys";
        const PEER_ID: &str = "16Uiu2HAm6YT98Hd3qAtop3TFM75uXvuyEhZYwPCfZ9mzRckmFkmW";

        let enr = Enr::from_str(ENR).unwrap();
        let pub_key = enr.public_key();
        let pub_key =
            libp2p_identity::secp256k1::PublicKey::try_from_bytes(&pub_key.encode()).unwrap();

        assert_eq!(PEER_ID, libp2p_identity::PeerId::from_public_key(&pub_key.into()).to_string());
    }

    #[test]
    fn test_bootnodes_empty() {
        let bootnodes = BootNodes(vec![]);
        assert!(bootnodes.is_empty());

        let bootnodes = BootNodes::from_chain_id(OP_MAINNET_CHAIN_ID);
        assert!(!bootnodes.is_empty());
    }
}
