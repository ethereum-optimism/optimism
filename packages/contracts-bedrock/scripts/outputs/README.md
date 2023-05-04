## L2 Output Oracle Scripts

A collection of scripts to interact with the L2OutputOracle.

### Output Deletion

[DeleteOutput](./DeleteOutput.s.sol) contains a variety of functions that deal
with deleting an output root from the [L2OutputOracle](../../contracts/L1/L2OutputOracle.sol).

To delete an output root, the script can be run as follows, where `<L2_OUTPUT_INDEX>` is
the index of the posted output to delete.

```bash
$ forge script scripts/output/DeleteOutput.s.sol \
  --sig "run(uint256)" \
  --rpc-url $ETH_RPC_URL \
  --broadcast \
  --private-key $PRIVATE_KEY \
  <L2_OUTPUT_INDEX>
```

To find and confirm the output index, there are a variety of helper functions that
can be run using the script `--sig` flag, passing the function signatures in as arguments.
These are outlined below.

### Retrieving an L2 Block Number

The output's associated L2 block number can be retrieved using the following command, where
`<L2_OUTPUT_INDEX>` is the index of the output in the [L2OutputOracle](../../contracts/L1/L2OutputOracle.sol).

```bash
$ forge script scripts/output/DeleteOutput.s.sol \
  --sig "getL2BlockNumber(uint256)" \
  --rpc-url $ETH_RPC_URL \
  <L2_OUTPUT_INDEX>
```


