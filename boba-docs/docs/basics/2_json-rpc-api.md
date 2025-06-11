# JSON-RPC API

Boba Network shares the same [JSON-RPC API (opens new window)](https://eth.wiki/json-rpc/API) as Ethereum. Some new custom methods have been introduced and some other have been made unsupported for operation.

## Debug JSON RPC Methods

You can use the Tenderly node to debug your transactions.

**Example**

```
curl https://1rpc.io/boba/eth -X POST --header 'Content-type: application/json' --data '{"jsonrpc":"2.0", "method":"debug_traceTransaction", "params":["0xf97b6fdce473b96d9cb00bb45d8fbfbc2911f383c7d525ec9d84d916cc30d347", {}], "id":1}'
```

## Custom JSON RPC Methods

**`eth_getProof`**

Returns the account and storage values of the specified account including the Merkle-proof. This call can be used to verify that the data you are pulling from is not tampered with.

**Parameters**

1. `DATA` - address of the account.
2. `ARRAY` - array of storage-keys which should be proofed and included. See [eth\_getStorageAt (opens new window)](https://eth.wiki/json-rpc/API#eth\_getStorageAt).
3. `QUANTITY|TAG` - integer of the ending block number for the range, or the string `"earliest"`, `"latest"` or `"pending"`, as in the [default block parameter (opens new window)](https://eth.wiki/json-rpc/API#the-default-block-parameter).

**Returns**

`Object`

* `balance`: `QUANTITY` - the balance of the account.
* `codeHash`: `DATA` - hash of the code of the account. For a simple Account without code it will return "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
* `nonce`: `QUANTITY` - nonce of the account
* `storageHash`: `DATA` - SHA3 of the StorageRoot. All storage will deliver a MerkleProof starting with this rootHash.
* `accountProof`: `ARRAY` - Array of rlp-serialized MerkleTree-Nodes, starting with the stateRoot-Node, following the path of the SHA3 (address) as key.
* `storageProof`: `ARRAY` - Array of storage-entries as requested. Each entry is a object with these properties:
  * `key`: `QUANTITY` - the requested storage key
  * `value`: `QUANTITY` - the storage value
  * `proof`: `ARRAY` - Array of rlp-serialized MerkleTree-Nodes, starting with the storageHash-Node, following the path of the SHA3 (key) as path.

**Example**

```
// Request
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getProof","params":["0xa87E6c7E8B9148f9Ef124344d6c384f8E20c3e14",["0x0000000000000000000000000000000000000000000000000000000000000000"],"latest"],"id":1}' <node url>

// Result
{
   "jsonrpc":"2.0",
   "id":1,
   "result":{
      "address":"0xa87e6c7e8b9148f9ef124344d6c384f8e20c3e14",
      "accountProof":[
         "0xf90211a0a70ea8f158b776c1faa3841d915476a0aea9352bb8d8773582b1c70df371dc9aa01c86c632d90e1382e1c9a3375588b4be66a8ccc4b1384454ee646c36a69260f3a0c1e06ac68eacc694a26f95bbcbec9a9d3feb609f9b35a4da78334051422633a8a0d737b127a89853ca0cef2975a051320c71501e40efdd9d5a67c489353d8e0618a0bbe2cd5afebb25fa6935f28dcc333babea67f204509c99ffa98987f57152ced4a0485ffad637642d17560a53df14c94722e17a6b501565ce9a9e035b03a7258878a0046b932283d8c882608d011ede93eabcd40d4531fb710d1115230377087a5cada0225822d83b071c8231c855a6a5a2ef530e32a66438fa3b1f998d1c483958f12ca0180515944ab33cb79b456cfe41f75a5414cae5bbd66118b7274826329162745fa0e776de4262ce8105fd32c37063ef6b7d891f1b635b1f18ad29f0416ccfa30a03a039a20244c11b40117037b9f5e4cab100aa4f070c6d90d1041d5cacd0e8c5fdcca0ce008187de834f8f09baa501cfb90d97d95435d1b9961104b60d2bc315787ab3a01f54648ff45a17104f9283944522a9389419de8c7f7d5c137cdb40848032051ea09ff2ac224c214b196bb38ac76b4b006beacff01050846da729adfd1bb5e9384fa0cc3b666741ebb6498668acf59c2196f491a3d393975cdbad80b8b1a52add3c7ea038be43ec587b0963f73f70ded1b018cb05375cbd69f8e46a3d85ea575b290eb380",
         "0xf90211a08383a520f766db7e094e4ecee49794921a5ca91164f3f807df0691a4fdcdc444a036af982101193d8d55e447ade9b4c4ab2a367c6c3d0e8eb7d5b60b356a87988da0b67f531ced7096ea9170e245e876076225731e260d55bc0f60c69276a780cb17a018b3046c48c31f0bc17b141913b628340c0be644f4d6e9313478f83774a25b3ba0227c5167376fdc443aa5a1d5eba34166240ee5c02254c7c99edffebad70e86b4a006d252bfe6b02dab6948b9863dd1c15386a9029959c01fe8004d4db2cb2d6113a02c20bb4e56ddaf193a5bf212985485de906dab61c631a27c541c788afe46e3bba02dbaa9c119306c58de48809ac6f23c6d04e90ba29ae75d78edb516538e298629a01902fabb96988ee115dbbf60a355d9e3eddfc5627e3b10317207d5dc6c8d5278a0688298269cab6222c003dac29bbdb262a0e8b942bffe6d41beba524fdd07aa4da0641e0c96a1368a74cb24dc40bffc20c8ad75889dd047a08c4dc979dfd7549f38a0034f0d893947d38a3ff37cfdfcda681b7c5cf44418dc186d37d0ea837aa4b71fa0e54d277a9d269a81e43042cf01a0b72ec3da4ccb4e117bf1b1598dc17d1574bba0294e48d0e407cb20f7f716ebb1e4d453eb5826cb951bde0649585f3242095e18a04f8e55afb0a22ec81ed6b0c16e1133d9d6393d673b4a9333a0f8691f3f794d2ba04604e60333f285c4f2ada70ede50402f5ee093176517c216b1c852145ab18c5d80",
         "0xf901f1a01c21b172cc30bfdfb24224ee85a9a16a278543b310164cc5165221bebf380d1ca00adcb99a7711fcd8157a7073a7941d1066d90e0947322561705f99e55512e68aa015e07fcaa7ae35de035f2258ba94c352d35fa86c19720657fa61f89027fa5c53a0187644a70c14f27e6685d5fa84d0b606e1ea54ffcb2bc3926764423628df9c99a0d7d5ebe98c1302633c5c7e8fa04d819571da58469c56aba9415a8b114c9a85f3a0e4f4c792f5dc7d727f087d75878c6dd2a0f373d2db71ad448e4bda39be9df3ed80a03b014aee0890d8bb4fac6c44f91330311bdae93636e51c877e48c303825a948aa0df96464be2702a00924d8cba1cf1ad62b569584b4c2264e2bd6be1d683b5132ba039f9da91a3b196f654fbf378cf17124a4790f2a8ac06f94660e4108d56d21547a07ed45b3feaf56f337b4c0e64291788f46281b5edb1b179e637ab4889557b23f2a011253fc9dc6f56effdd3fe3623792a58e7475bc9daa85a847b1af91a37bed199a09bf605bee19d18b1111b5e4f5b67d38bf77b3792bbbf597a662bb1f64023ec34a08b189b7b1929af2cd91e6a80bb2e3e0ee6dba830d32b70dabaddb2e573826048a09cd4cbd92da507018d3fcb952f48fe9214f1a281260e1756e33bb4d91594db0aa04afbfb856a54a99b380b0cad4e6f08da43ea9d3fc50b0f304211db2e5c47a74380",
         "0xf851a07f3a2c03897659425cb8f87b0a300709c9dac9242a6217806509c1284115fccd80808080a03abd0c6a86c63850c41944bbe6296590ea7c5ae6c8073e487d489a7b23298beb8080808080808080808080"
      ],
      "balance":"0x0",
      "codeHash":"0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
      "nonce":"0x0",
      "storageHash":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
      "storageProof":[
         {
            "key":"0x0000000000000000000000000000000000000000000000000000000000000000",
            "value":"0x0",
            "proof":[

            ]
         }
      ]
   }
}
```

**`eth_estimateGas`**

Generates and returns an estimate of how much gas is necessary to allow the transaction to complete. The transaction will not be added to the blockchain.

**Parameters**

1. `OBJECT` - the tx call object, nonce field is omitted
2. `QUANTITY|TAG` - integer of the ending block number for the range, or the string `"earliest"`, `"latest"` or `"pending"`, as in the [default block parameter (opens new window)](https://eth.wiki/json-rpc/API#the-default-block-parameter).

**Returns**

`QUANTITY` - the amount of gas used.

**Example**

```
// Request
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_getProof","params":[{see above}],"id":1}' <node url>

// Result
{
  "jsonrpc":"2.0",
  "id":1,
  "result": "0x5208" // 21000
}
```

## Unspported JSON RPC Methods

**`eth_getAccounts`**

This method is used to retrieve a list of addresses owned by a user. Boba Network nodes do not expose internal wallets for security reasons and therefore block the `eth_getAccounts` method by default. You should use external wallet software as an alternative.

**`eth_sendTransaction`**

Boba Network nodes also block the `eth_sendTransaction` method for the same reasons as `eth_getAccounts`. You should use external wallet software as an alternative. Please note that this is not the same as the `eth_sendRawTransaction` method, which accepts a signed transaction as an input. `eth_sendRawTransaction` _is_ supported by Boba Network.
