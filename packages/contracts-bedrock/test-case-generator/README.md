# `ctb-test-case-generator`

A lightweight input fuzzing utility used for testing various Bedrock contracts.

<pre>
├── test-case-generator
│   ├── <a href="./cmd">cmd</a>: `ctb-test-case-genertor`'s binary
│   └── <a href="./trie">trie</a>: Utility for generating random merkle trie roots / inclusion proofs
</pre>

## Usage

To build, run `yarn build:fuzz` from this directory or the `contract-bedrock` package.

To generate an abi-encoded fuzz case, pass in a mode via the `-m` flag as well as an optional variant via the `-v` flag.

### Available Modes

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
