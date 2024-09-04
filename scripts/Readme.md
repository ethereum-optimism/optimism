# Scripts

A collection of simple scripts to make research and understanding the op-node easier with help of devnet. This is WIP by all means, so feel free to include improvements.

## Usage

Just run `go run .` in this folder

```bach
go run . l2-tx
```

The first script sends a simple L2 transaction. It will also print the `L1 Origin` (epoch number) associated with a block in which the transaction was included (helpful when trying to find corresponding L1 batch).