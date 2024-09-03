# `ctb-go-ffi`

A lightweight binary for utilities accessed via `forge`'s `ffi` cheatcode in the `contracts-bedrock` test suite.

<pre>
├── go-ffi
│   ├── <a href="./bin.go">bin</a>: `go-ffi`'s binary
│   ├── <a href="./trie.go">trie</a>: Utility for generating random merkle trie roots / inclusion proofs
│   └── <a href="./differential-testing.go">diff-testing</a>: Utility for differential testing Solidity implementations against their respective Go implementations.
</pre>

## Usage

To build, run `just build-go-ffi` from this directory or the `contract-bedrock` package.

### In a Forge Test

To use `go-ffi` in a forge test, simply invoke the binary using the solidity `Process` library's `run` method.

```solidity
function myFFITest() public {
    string[] memory commands = new string[](3);
    commands[0] = "./scripts/go-ffi/go-ffi";
    commands[1] = "trie";
    commands[2] = "valid";
    bytes memory result = Process.run(commands);

    // Do something with the result of the command
}
```

### Available Modes

There are two modes available in `go-ffi`: `diff` and `trie`. Each are present as a subcommand to the `go-ffi` binary, with their own set of variants.

#### `diff`

> **Note**
> Variant required for diff mode.

| Variant                               | Description                                                                                                        |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------
| `decodeVersionedNonce`                | Decodes a versioned nonce and prints the decoded arguments                                                         |
| `encodeCrossDomainMessage`            | Encodes a cross domain message and prints the encoded message                                                      |
| `hashCrossDomainMessage`              | Encodes and hashes a cross domain message and prints the digest                                                    |
| `hashDepositTransaction`              | Encodes and hashes a deposit transaction and prints the digest                                                     |
| `encodeDepositTransaction`            | RLP encodes a deposit transaction                                                                                  |
| `hashWithdrawal`                      | Hashes a withdrawal message and prints the digest                                                                  |
| `hashOutputRootProof`                 | Hashes an output root proof and prints the digest                                                                  |
| `getProveWithdrawalTransactionInputs` | Generates the inputs for a `getProveWithdrawalTransaction` call to the `OptimismPortal` given a withdrawal message |
| `cannonMemoryProof`                   | Computes a merkle proof of Cannon's memory                                                                         |

#### `trie`

> **Note**
> Variant required for `trie` mode.

| Variant                       | Description                                                                                                                               |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `valid`                       | Generate a test case with a valid proof of inclusion for the k/v pair in the trie.                                                        |
| `extra_proof_elems`           | Generate an invalid test case with an extra proof element attached to an otherwise valid proof of inclusion for the passed k/v.           |
| `corrupted_proof`             | Generate an invalid test case where the proof is malformed.                                                                               |
| `invalid_data_remainder`      | Generate an invalid test case where a random element of the proof has more bytes than the length designates within the RLP list encoding. |
| `invalid_large_internal_hash` | Generate an invalid test case where a long proof element is incorrect for the root.                                                       |
| `invalid_internal_node_hash`  | Generate an invalid test case where a small proof element is incorrect for the root.                                                      |
| `prefixed_valid_key`          | Generate a valid test case with a key that has been given a random prefix                                                                 |
| `empty_key`                   | Generate a valid test case with a proof of inclusion for an empty key.                                                                    |
| `partial_proof`               | Generate an invalid test case with a partially correct proof                                                                              |
