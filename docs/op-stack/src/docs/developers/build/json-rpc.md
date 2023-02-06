---
title: JSON-RPC API
lang: en-US
---

<details>
<summary><b>Pre-bedrock (current version)</b></summary>

Optimism shares the same [JSON-RPC API](https://eth.wiki/json-rpc/API) as Ethereum.
Some custom methods have been introduced to simplify certain Optimism specific interactions.

## Custom JSON-RPC Methods

### `eth_getBlockRange`

Like `eth_getBlockByNumber` but accepts a range of block numbers instead of just a single block.

**Parameters**

1. `QUANTITY|TAG` - integer of the starting block number for the range, or the string `"earliest"`, `"latest"` or `"pending"`, as in the [default block parameter](https://eth.wiki/json-rpc/API#the-default-block-parameter).
2. `QUANTITY|TAG` - integer of the ending block number for the range, or the string `"earliest"`, `"latest"` or `"pending"`, as in the [default block parameter](https://eth.wiki/json-rpc/API#the-default-block-parameter).
3. `BOOLEAN` - If `true` it returns the full transaction objects, if `false` only the hashes of the transactions.

**Returns**

An array of `block` objects.
See [`eth_getBlockByHash`](https://eth.wiki/json-rpc/API#eth_getblockbyhash) for the structure of a `block` object.

**Example**

```json
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_getBlockRange","params":["0x1", "0x2", false],"id":1}' <node url>

// Result
{
  "jsonrpc":"2.0",
  "id":1,
  "result":[
    {
      "difficulty":"0x2",
      "extraData":"0xd98301090a846765746889676f312e31352e3133856c696e75780000000000009c3827892825f0825a7e329b6913b84c9e4f89168350aff0939e0e6609629f2e7f07f2aeb62acbf4b16a739cab68866f4880ea406583a4b28a59d4f55dc2314e00",
      "gasLimit":"0xe4e1c0",
      "gasUsed":"0x3183d",
      "hash":"0xbee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e875453",
      "logsBloom":"0x00000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000400000000000100000000000000200000000002000000000000001000000000000000000004000000000000000000000000000040000400000100400000000000000100000000000000000000000000000020000000000000000000000000000000000000000000000001000000000000000000000100000000000000000000000000000000000000000000000000000000000000088000000080000000000010000000000000000000000000000800008000120000000000000000000000000000000002000",
      "miner":"0x0000000000000000000000000000000000000000",
      "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
      "nonce":"0x0000000000000000",
      "number":"0x1",
      "parentHash":"0x7ca38a1916c42007829c55e69d3e9a73265554b586a499015373241b8a3fa48b",
      "receiptsRoot":"0xf4c97b1186b690ad3318f907c0cdaf46f4598f27f711a5609064b2690a767287",
      "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
      "size":"0x30c",
      "stateRoot":"0xd3ac40854cd2ac17d8effeae6065cea990b04be714f7061544973feeb2f1c95f",
      "timestamp":"0x618d8837",
      "totalDifficulty":"0x3",
      "transactions":["0x5e77a04531c7c107af1882d76cbff9486d0a9aa53701c30888509d4f5f2b003a"],
      "transactionsRoot":"0x19f5efd0d94386e72fcb3f296f1cb2936d017c37487982f76f09c591129f561f",
      "uncles":[]
    },
    {
      "difficulty":"0x2",
      "extraData":"0xd98301090a846765746889676f312e31352e3133856c696e757800000000000064a82cb66c7810b9619e7f14ab65c769a828b1616974987c530684eb3870b65e5b2400c1b61c6d340beef8c8e99127ac0de50e479d21f0833a5e2910fe64b41801",
      "gasLimit":"0xe4e1c0",
      "gasUsed":"0x1c60d",
      "hash":"0x45fd6ce41bb8ebb2bccdaa92dd1619e287704cb07722039901a7eba63dea1d13",
      "logsBloom":"0x00080000000200000000000000000008000000000000000000000100008000000000000000000000000000000000000000000000000000000000400000000000100000000000000000000000020000000000000000000000000000000000004000000000000000000000000000000000400000000400000000000000100000000000000000000000000000020000000000000000000000000000000000000000100000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000008400000000000000000010000000000000000020000000020000000000000000000000000000000000000000000002000",
      "miner":"0x0000000000000000000000000000000000000000",
      "mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000",
      "nonce":"0x0000000000000000",
      "number":"0x2",
      "parentHash":"0xbee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e875453",
      "receiptsRoot":"0x2057c8fb79c0f294062c1436aa56741134dc46d228a4f874929f8b791a7007a4",
      "sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
      "size":"0x30c",
      "stateRoot":"0x87026f3a614318ae24bcef6bc8f7564479afbbbe2b1fb189bc133a5de5a2b0f8",
      "timestamp":"0x618d8837",
      "totalDifficulty":"0x5",
      "transactions":["0xaf6ed8a6864d44989adc47c84f6fe0aeb1819817505c42cde6cbbcd5e14dd317"],
      "transactionsRoot":"0xa39c4d0d2397f8fcb1683ba833d4ab935cd2f4c5ca6f56a7d9a45b9904ea1c69",
      "uncles":[]
    }
  ]
}
```

---

### `rollup_getInfo`

Returns useful L2-specific information about the current node.

**Parameters**

None

**Returns**

`Object`
- `mode`: `STRING` - `"sequencer"` or `"verifier"` depending on the node's mode of operation
- `syncing`: `BOOLEAN` - `true` if the node is currently syncing, `false` otherwise
- `ethContext`: `OBJECT`
  - `blockNumber`: `QUANTITY` - Block number of the latest known L1 block
  - `timestamp`: `QUANTITY` - Timestamp of the latest known L1 block
- `rollupContext`: `OBJECT`
  - `queueIndex`: `QUANTITY` - Index within the CTC of the last L1 to L2 message ingested
  - `index`: `QUANTITY` - Index of the last L2 tx processed
  - `verifiedIndex`: `QUANTITY` - Index of the last tx that was ingested from a batch that was posted to L1

**Example**

```json
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"rollup_getInfo","params":[],"id":1}' <node url>

// Result
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "mode":"verifier",
    "syncing":false,
    "ethContext":{
      "blockNumber":13679735,
      "timestamp":1637791660
    },
    "rollupContext":{
      "index":430948,
      "queueIndex":12481,
      "verifiedIndex":0
    }
  }
}
```

---

### `rollup_gasPrices`

Returns the L1 and L2 gas prices that are being used by the Sequencer to calculate fees.

**Parameters**

None

**Returns**

`Object`
- `l1GasPrice`: `QUANTITY` - L1 gas price in wei that the Sequencer will use to estimate the L1 portion of fees (calldata costs).
- `l2GasPrice`: `QUANTITY` - L2 gas price in wei that the Sequencer will use to estimate the L2 portion of fees (execution costs).

**Example**

```json
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"rollup_gasPrices","params":[],"id":1}' <node url>

// Result
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "l1GasPrice":"0x237aa50984",
    "l2GasPrice":"0xf4240"
  }
}
```

---

## Unsupported JSON-RPC methods

### `eth_getAccounts`

This method is used to retrieve a list of addresses owned by a user.
Optimism nodes do not expose internal wallets for security reasons and therefore block the `eth_getAccounts` method.
You should use external wallet software as an alternative.

### `eth_sendTransaction`

Optimism nodes also block the `eth_sendTransaction` method for the same reasons as `eth_getAccounts`.
You should use external wallet software as an alternative.
Please note that this is not the same as the `eth_sendRawTransaction` method, which accepts a signed transaction as an input.
`eth_sendRawTransaction` _is_ supported by Optimism.

</details>

<details>
<summary><b>Bedrock (coming Q1 2023)</b></summary>

There are several bedrock components with an RPC API:

## Rollup node (op-node)

[*Rollup node*](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md) refers to the component in the protocol specifications. 
The Optimism implementation is called *op-node*.

The `op-node` component implements several RPC methods:

### `optimism_outputAtBlock`

Get the output root at a specific block.
This method is documented in [the specifications](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md#output-method-api).

```sh
curl -X POST -H "Content-Type: application/json" --data  \
   '{"jsonrpc":"2.0","method":"optimism_outputAtBlock","params":["latest"],"id":1}' \
   http://localhost:9545
```

Sample output:

```json
{
   "jsonrpc":"2.0",
   "id":1,
   "result":[
      "0x0000000000000000000000000000000000000000000000000000000000000000",
      "0xabe711e34c1387c8c56d0def8ce77e454d6a0bfd26cef2396626202238442421"
   ]
}
```

### `optimism_syncStatus`

Get the synchronization status.

```sh
curl -X POST -H "Content-Type: application/json" --data \
    '{"jsonrpc":"2.0","method":"optimism_syncStatus","params":[],"id":1}'  \
    http://localhost:9545
```

Sample output:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "current_l1": {
      "hash": "0x5adcfcbd1c2fcf9e06bfdaa8414a4586f84e11f487396abca940299eb0ed2da5",
      "number": 7569281,
      "parentHash": "0xfd022ca8a8c4e0f3bfd67081c18551840ea0717cc01d9a94601e1e41e92616d3",
      "timestamp": 1662862860
    },
    "head_l1": {
      "hash": "0x5c12fde5ea79aefe4b52c0c8cc0e0eb33a2ccb423cb3cd9c9132e18ad42e89b6",
      "number": 8042823,
      "parentHash": "0x74818f8ecaa932431bf9523e929dcfa11ab382c752529d8271a24810884a2551",
      "timestamp": 1669735356
    },
    "safe_l1": {
      "hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "number": 0,
      "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "timestamp": 0
    },
    "finalized_l1": {
      "hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "number": 0,
      "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "timestamp": 0
    },
    "unsafe_l2": {
      "hash": "0x1cad05886ec0e2cda728674e00eadcbb9245ff34c0bfd86c866673a615c1c43a",
      "number": 1752,
      "parentHash": "0x0115dbbd26aaf9563d7e3cad65bad41926d94b2643ccb080f71e394c2c3d62a3",
      "timestamp": 1662861300,
      "l1origin": {
        "hash": "0x43fe1601041056e9a2a5dabaa20715518ae0058abf67a69f5ebdd53b1f6ff02f",
        "number": 7569162
      },
      "sequenceNumber": 0
    },
    "safe_l2": {
      "hash": "0x1cad05886ec0e2cda728674e00eadcbb9245ff34c0bfd86c866673a615c1c43a",
      "number": 1752,
      "parentHash": "0x0115dbbd26aaf9563d7e3cad65bad41926d94b2643ccb080f71e394c2c3d62a3",
      "timestamp": 1662861300,
      "l1origin": {
        "hash": "0x43fe1601041056e9a2a5dabaa20715518ae0058abf67a69f5ebdd53b1f6ff02f",
        "number": 7569162
      },
      "sequenceNumber": 0
    },
    "finalized_l2": {
      "hash": "0x6758307d692d4f2f6650acd3762674749a0c1cc2530b9b481845d0f8ee1bd456",
      "number": 0,
      "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "timestamp": 1662857796,
      "l1origin": {
        "hash": "0xb0bbb79a00fb8485185b1bedfac386812d662e1cddba77b67a26e1ed9ba8f0ec",
        "number": 7568910
      },
      "sequenceNumber": 0
    }
  }
}
```

### `optimism_rollupConfig`

Get the rollup configuration parameters.

```sh
curl -X POST -H "Content-Type: application/json" --data \
    '{"jsonrpc":"2.0","method":"optimism_rollupConfig","params":[],"id":1}'  \
    http://localhost:9545
```

Sample output:

```json
{
   "jsonrpc":"2.0",
   "id":1,
   "result":{
      "genesis":{
         "l1":{
            "hash":"0xb0bbb79a00fb8485185b1bedfac386812d662e1cddba77b67a26e1ed9ba8f0ec",
            "number":7568910
         },
         "l2":{
            "hash":"0x6758307d692d4f2f6650acd3762674749a0c1cc2530b9b481845d0f8ee1bd456",
            "number":0
         },
         "l2_time":1662857796
      },
      "block_time":2,
      "max_sequencer_drift":120,
      "seq_window_size":120,
      "channel_timeout":30,
      "l1_chain_id":5,
      "l2_chain_id":28528,
      "p2p_sequencer_address":"0x59dc8e68a80833cc8a9592d532fed42374c8b5dc",
      "fee_recipient_address":"0xdffc6a1c238ff9504b055ad7efeee0148f2d62bd",
      "batch_inbox_address":"0xfeb2acb903f95fb5f5497157c0727a7d16e3fd16",
      "batch_sender_address":"0x4ff79526ea1d492a3db2aa210d7318ff13f2012c",
      "deposit_contract_address":"0xa581ca3353db73115c4625ffc7adf5db379434a8"
   }
}
```

### `optimism_version`

Get the software version.

```sh
curl -X POST -H "Content-Type: application/json" \
'--data '{"jsonrpc":"2.0","method":"optimism_version","params":[],"id":1}' \
http://localhost:9545
```

Sample output:

```json
{
   "jsonrpc":"2.0",
   "id":1,
   "result":"v0.0.0-"
}
```

### Peer to peer synchronization

Optionally, the rollup node can provide [peer to peer synchronization](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node-p2p.md) to provide pending L2 blocks to other rollup nodes.


## Execution engine (op-geth)

[*Execution engine*](https://github.com/ethereum-optimism/optimism/blob/develop/specs/rollup-node.md) refers to the component in the protocol specifications. 
The Optimism implementation is called *op-geth*.

The execution engine's RPC interface is identical to [the upstream Geth RPC interface](https://geth.ethereum.org/docs/rpc/server). This includes the ability to provide [snap sync](https://github.com/ethereum/devp2p/blob/master/caps/snap.md) functionality to other execution engines.

The responses are nearly identical too, except we also include the L1 gas usage and price information.

## Daisy chain

The daisy chain is a proxy that distributes requests either to the execution engine (if related to post-Bedrock blocks), or the legacy geth (if related to blocks prior to bedrock). 
It accepts [the interface used by L1 execution engines](https://playground.open-rpc.org/?schemaUrl=https://raw.githubusercontent.com/ethereum/execution-apis/assembled-spec/openrpc.json&uiSchema%5BappBar%5D%5Bui:splitView%5D=false&uiSchema%5BappBar%5D%5Bui:input%5D=false&uiSchema%5BappBar%5D%5Bui:examplesDropdown%5D=false).

## Legacy geth

The legacy geth provides information about the blockchain prior to Bedrock.
It implements the read-only methods of [the interface used by L1 execution engines](https://playground.open-rpc.org/?schemaUrl=https://raw.githubusercontent.com/ethereum/execution-apis/assembled-spec/openrpc.json&uiSchema%5BappBar%5D%5Bui:splitView%5D=false&uiSchema%5BappBar%5D%5Bui:input%5D=false&uiSchema%5BappBar%5D%5Bui:examplesDropdown%5D=false).
It does not implement `eth_sendTransaction` and `eth_sendRawTransaction`, because they don't make sense in a read-only copy.

</details>