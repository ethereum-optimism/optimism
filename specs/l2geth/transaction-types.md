# Transaction Types

This defines the serialization of the transactions that are submitted to the
Canonical Transaction Chain.
Transaction types are defined by the leading byte which is used as an enum. The
purpose of a transaction type is to enable transaction compression as well as
allowing for new types to be defined that take advantage of the account
abstraction.

## EIP155

A compressed EIP155 transaction. The transaction that is signed follows
[EIP155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md). The type enum is `0`.

| Field    | Size (bytes) |
| -------- | ------------ |
| Type     | 1            |
| R        | 32           |
| S        | 32           |
| V        | 1            |
| gasLimit | 3            |
| gasPrice | 3            |
| nonce    | 3            |
| target   | 20           |
| data     | variable     |

The `gasPrice` must be scaled by a factor of `1,000,000` when encoding and
decoding. This means that precision is lost and must be divisibl by 1 million.
From a user experience perspective, the `gasPrice` must be at least 1 gwei and
at most 16777215 gwei. A partial gwei will result in an invalid transaction.

The max nonce is 16777215. Any nonces greater than that must result in an
invalid transaction.

## EthSign

A compressed EIP155 transaction that uses an alternative signature hashing
algorithm. The data is ABI encoded hashed and then signed with `eth_sign`.
The type enum is `1`.

| Field    | Size (bytes) |
| -------- | ------------ |
| Type     | 1            |
| R        | 32           |
| S        | 32           |
| V        | 1            |
| gasLimit | 3            |
| gasPrice | 3            |
| nonce    | 3            |
| target   | 20           |
| data     | variable     |

The same `gasPrice` and `nonce` rules apply as the `EIP155` transaction.

The following table shows how the fields are ABI encoded before hashing.

| Field    | ABI Type |
| -------- | -------- |
| nonce    | uint256  |
| gasLimit | uint256  |
| gasPrice | uint256  |
| chainId  | uint256  |
| target   | address  |
| data     | bytes    |

The ABI encoded data is hashed with `keccak256` and then prepended with
`\x19Ethereum Signed Message:\n32` before being hashed again with `keccak256`
to create the digest that is signed with the secp256k1 private key.
