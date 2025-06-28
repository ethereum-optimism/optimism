# Bootnodes

Synchronizing your node occurs in two ways.

Firstly, op-node pulls blocks from the L1 and decodes the data committed there to construct the finalized version of the blockchain.  However, the sequencer only periodically commits the data to the L1 (resulting in cost savings for all users) so the L1 version of the blockchain is always some period of time (usually no more than an hour) behind the sequencer.

Secondly, op-node pulls blocks from P2P which the sequencer has signed, indicating that the sequencer intends to eventually commit those block to the L1.  This P2P process allows for replicas in the network to near-instantly reflect the current state of the sequencer.  Since the block sharing process is P2P, your node needs to discover other nodes in the P2P network.  There are special nodes for facilitating this initial discovery called bootnodes that should be specified in your node config.

They can be specified to op-node with either the `--p2p.bootnodes` flag, or via the `OP_NODE_P2P_BOOTNODES` environment variable.

## Boba Mainnet

```
OP_NODE_P2P_BOOTNODES=enode://a38db98391708094dd679aacdd356efdbe10ad7c9ba4b87d8f4b79eb24ac26328e7fdf911f8c5c3b7786c2505bc77d896a0ecd1816bd56107592f728dcff3945@35.153.183.193:0?discport=30301,enode://a92b84bac2893ef868659364a6784a6eeb146cb04ec0bb3b1e9925f5bb895af9e1c50deaf0703ab26db366a3d82272e710e06d925cb0b7245129ed76d54ccac9@13.221.254.11:0?discport=30301
```

## Boba Sepolia Testnet

```
OP_NODE_P2P_BOOTNODES=enode://0d9f376f90b72f65c8cbeae4c7d34732678822109412c7e8253952b207618733a736f20b43c4c7c62e95b7e6389f04cc9e9e8dbe79ba75af884b0fcf5d580a3d@10.26.0.189:0?discport=30301,enode://48eb348c252f9c527ae40dee64995a65a5a6100725b1462184bb5b86cfbe817864e8ff90b52a78124289a476a1d805b27173d8f01722160c08159fa8880c47c0@10.26.4.58:0?discport=30301
```


