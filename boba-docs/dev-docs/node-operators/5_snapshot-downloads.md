# Node Snapshots

This page has a list of important snapshots for node operators. Data directories and node snapshots are not required but can significantly simplify the node operation process.

## Snapshots

Always verify snapshots by comparing the sha256sum of the downloaded file to the sha256sum listed on this page. Check the sha256sum of the downloaded file by running `sha256sum <filename>`in a terminal.

### BOBA Mainnet (Archive Node)

| Client | Snapshot Date | Size     | Download Link                                                | Sha256sum                                                    |
| ------ | ------------- | -------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Reth   | 2026-03-20    | -        | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-reth-db-20260320.tar.zst) | `ed8dbdab30d01612b2797276274315579be60413a3509e783f1ee462fa1b6f80` |
| Reth   | initial (block 1149019) | 74MB     | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-reth-db-initial.tar.zst) | `22b26c59ecbfc7b8f77525dc210d1780aa2bbf072084aae2b9d2e88b1f1d3301` |
| Erigon | 2024-04-16    | 1016.4MB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-erigon-db-1149019.tgz) | `98bfd73716585f412a6388bb51a8bfb945170d0d228efb4d218d98d523d76168` |
| Geth   | 2024-04-16    | 16.3GB   | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-114909.tgz) | `102922968680e86afe0588cf22924639f6f2ab32aee1c1e2325c3026b262692b` |
| Geth   | 2024-07-30    | 49.7GB   | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-snapshot-5683043.tgz) | `9d20e16434bbdb8d2fdcdfee848f7941337e6b2cfb8feece441820c9a6364d09` |
| Geth   | 2025-10-10    | 146GB   | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-20251010.tar.zst) | `4f4c0a1d2376382110ecc90d46329850222feaedd9d320b146c326359d5b1353` |

### BOBA Mainnet (Reth Migration Artifacts)

These are the source artifacts used to generate the initial reth database via `op-reth init-state`. They allow independent verification and reproducibility without needing a synced geth/erigon node.

| File | Description | Size | Download Link | Sha256sum |
| ---- | ----------- | ---- | ------------- | --------- |
| State dump | All accounts at migration block 1149019 in reth JSONL format | 150MB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-reth-state-initial.jsonl) | `1b226876bf6de17deed4865226ad87c77fd57dd43f47f67d5c870d937396463a` |
| Block header | RLP-encoded header for block 1149019 | 508B | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-reth-header-initial.rlp) | `323a5337d151204724817a826d7846cb29a7aef0738448c9c0dd9250a8402eb8` |

### BOBA Mainnet (Legacy)

| Client | Snapshot Date | Size   | Download Link                                                | Sha256sum                                                    |
| ------ | ------------- | ------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Geth   | 2024-04-16    | 16.3GB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/mainnet/boba-mainnet-geth-db-legacy.tgz) | `5864b5da7cebe0810a2be4d9cdcc0fdca91f2ee6b278c87ef518e8a852f0da72` |

### BOBA Sepolia Testnet (Archive Node)

| Client | Snapshot Date | Size  | Download Link                                                | Sha256sum                                                    |
| ------ | ------------- | ----- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Reth   | 2026-04-15    | -     | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-reth-db-20260415.tar.zst) | `56b5deb405c74d294e4ddbf6f32ce59f812f12dd5047e053e175828eda8e2c15` |
| Reth   | initial (block 511) | 343KB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-reth-db-initial.tar.zst) | `703cb2c0c33a4689d7cfabb10483e7ec0900068f597da9455db0dcba06a9d8bf` |
| Erigon | 2024-01-18    | 912KB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz) | `b887d2e0318e9299e844da7d39ca32040e3d0fb6a9d7abe2dd2f8624eca1cade` |
| Geth   | 2024-01-18    | 2MB   | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz) | `b229c8e51e41a26bb21a84b329d3134ae5cc6541b04eb160aebd573f0e6b94ae` |
| Erigon | 2024-02-13    | 586M  | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-snapshot-1126371.tgz) | `688fd431656b673d3f2c690d79277b6d659a51c48c7b73a5e36bb8fbfdbdea80` |
| Erigon | 2024-3-01     | 956M  | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-snapshot-1857820.tgz) | `3464fa9c1f669ad3d26359e5c463c33d3d60735a7aafb8e10d2dfd4719a71c07` |
| Geth   | 2025-10-02    | 135GB   | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db-20251002.tar.zst) | `578cdc9ae438986a7895e3c16485e8268c2eae9fafa622dd3f0c778b30a2a5a8` |

### BOBA Sepolia Testnet (Reth Migration Artifacts)

| File | Description | Size | Download Link | Sha256sum |
| ---- | ----------- | ---- | ------------- | --------- |
| State dump | All accounts at migration block 511 in reth JSONL format | 1.1MB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-reth-state-initial.jsonl) | `79dbeaaff89db3422ff057b67e78f3a576494213e5927347ce6f22cc829c0943` |
| Block header | RLP-encoded header for block 511 | 1KB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-reth-header-initial.rlp) | `0514713a96e46856b3db8618d2c76c98433e1e506431f47c76473d77b5a676b8` |

### BOBA Sepolia Testnet (Legacy)

| Client | Snapshot Date | Size  | Download Link                                                | Sha256sum                                                    |
| ------ | ------------- | ----- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Geth   | 2024-01-18    | 1.5MB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db-legacy.tgz) | `e8aa790f15e46bdd63cc6532c4b1df77d78cda83fcd6e55568317d23eeabc4c3` |


### Optimism Sepolia Testnet (Archive Node)

| Client | Snapshot Date | Size  | Download Link                                                | Sha256sum                                                    |
| ------ | ------------- | ----- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Erigon | 2023-08-11    | 912KB | [Link](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/optimism-sepolia-erigon-db.tgz) | `10a5dd5cf58932df2bd90ef6844f2029b42c8a7fb2655ab2d558125db8db9c21` |

