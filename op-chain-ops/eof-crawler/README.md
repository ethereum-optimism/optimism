# `eof-crawler`

Simple CLI tool to scan all accounts in a geth LevelDB for contracts that begin with the EOF prefix.

## Usage

1. Pass the directory of the Geth DB into the tool
```sh
go run eof_crawler.go <db_path>
```
2. Once the indexing has completed, an array of all EOF-prefixed contracts will be written to `eof_contracts.json`.
