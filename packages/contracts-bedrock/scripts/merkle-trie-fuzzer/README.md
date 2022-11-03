# `merkle-trie-fuzzer`

A lightweight, efficient Merkle Trie fuzzer for use in Optimism bedrock's [Merkle Trie Verifier](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/libraries/trie/MerkleTrie.sol).

## Usage

```
patricia [OPTIONS] [COMMAND]

Commands:
  valid
          Generate a valid test case
  help
          Print this message or the help of the given subcommand(s)

Options:
  -p
          Pretty print the [TrieTestCase]

  -h, --help
          Print help information (use `-h` for a summary)
```
