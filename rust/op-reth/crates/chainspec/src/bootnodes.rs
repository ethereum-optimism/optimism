//! Default OP Stack discv5 bootnodes.
//!
//! A single mainnet pool and a single testnet pool, applied to every OP
//! Stack chain. Operated by OP Labs, Base, and Uniswap Labs.

use alloc::vec::Vec;
use reth_network_peers::NodeRecord;

/// Default OP Stack mainnet bootnodes.
pub(crate) static OP_BOOTNODES: &[&str] = &[
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
    // Uniswap Labs
    "enode://b1a743328188dba3b2ed8c06abbb2688fabe64a3251e43bd77d4e5265bbd5cf03eca8ace4cde8ddb0c49c409b90bf941ebf556094638c6203edd6baa5ef0091b@3.134.214.169:30303",
    "enode://ea9eaaf695facbe53090beb7a5b0411a81459bbf6e6caac151e587ee77120a1b07f3b9f3a9550f797d73d69840a643b775fd1e40344dea11e7660b6a483fe80e@52.14.30.39:30303",
    "enode://77b6b1e72984d5d50e00ae934ffea982902226fe92fa50da42334c2750d8e405b55a5baabeb988c88125368142a64eda5096d0d4522d3b6eef75d166c7d303a9@3.148.100.173:30303",
];

/// Default OP Stack testnet bootnodes.
pub(crate) static OP_TESTNET_BOOTNODES: &[&str] = &[
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

/// Returns parsed OP Stack mainnet bootnodes.
pub(crate) fn op_nodes() -> Vec<NodeRecord> {
    parse_nodes(OP_BOOTNODES)
}

/// Returns parsed OP Stack testnet bootnodes.
pub(crate) fn op_testnet_nodes() -> Vec<NodeRecord> {
    parse_nodes(OP_TESTNET_BOOTNODES)
}

fn parse_nodes(nodes: &[&str]) -> Vec<NodeRecord> {
    nodes.iter().map(|s| s.parse().expect("hardcoded enode is valid")).collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_op_mainnet_bootnodes() {
        let nodes = op_nodes();
        assert_eq!(nodes.len(), OP_BOOTNODES.len());
        assert_eq!(nodes.len(), 11);
    }

    #[test]
    fn parse_op_testnet_bootnodes() {
        let nodes = op_testnet_nodes();
        assert_eq!(nodes.len(), OP_TESTNET_BOOTNODES.len());
        assert_eq!(nodes.len(), 8);
    }
}
