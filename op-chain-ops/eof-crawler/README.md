# `eof-crawler`

Simple CLI tool to scan all accounts in a geth snapshot file for contracts that begin with the EOF prefix and store them.

## Usage

```sh
Usage: eof-crawler --snapshot-file <SNAPSHOT_FILE>

Options:
  -s, --snapshot-file <SNAPSHOT_FILE>  The path to the geth snapshot file
  -h, --help                           Print help
  -V, --version                        Print version
```

1. To begin, create a geth snapshot:
```sh
geth snapshot dump --nostorage >> snapshot.txt
```
1. Once the snapshot has been generated, feed it into the CLI:
```sh
cargo r -- -s ./snapshot.txt
```
1. The CLI will output a file named `eof_contracts.json` containing all found EOF-prefixed contracts.
