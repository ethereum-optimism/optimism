# op-program

Implements a fault proof program that runs through the rollup state-transition to verify an L2 output from L1 inputs.
This verifiable output can then resolve a disputed output on L1.

The program is designed such that it can be run in a deterministic way such that two invocations with the same input
data wil result in not only the same output, but the same program execution trace. This allows it to be run in an
on-chain VM as part of the dispute resolution process.

## Compiling

To build op-program, from within the `op-program` directory run:

```shell
make op-program
```

This resulting executable will be in `./bin/op-program`

## Testing

To run op-program unit tests, from within the `op-program` directory run:

```shell
make test
```

## Lint

To run the linter, from within the `op-program` directory run:
```shell
make lint
```

This requires having `golangci-lint` installed.

## Running

From within the `op-program` directory, options can be reviewed with:

```shell
./bin/op-program --help
```

### Verifying Optimism Goerli Output Claim

The fault proof program can be used to verify a claimed output root on the Optimism Goerli testnet.

#### Prerequisites

* An L1 node with block and receipt history exposing the `eth` JSON-RPC namespace
* An archive L2 op-geth node exposing the `eth` and `debug` JSON-RPC namespaces. Note that public RPC providers don't currently expose the required `debug_dbGet` API.
* Compile the op-program binary (see above)

#### Find the Program Inputs

The fault proof program needs to be given information about the claim to verify and agreed starting points:

* `--l1.head` - Hash of the L1 head block. Derivation stops after this block is processed
* `--l2.head` - Hash of the agreed L2 block to start derivation from
* `--l2.claim` - Claimed L2 output root to validate
* `--l2.blocknumber` - Number of the L2 block that the claim is from

It then uses those inputs to run the derivation process to either prove or disprove the claimed L2 output root.

Normally, the values for these inputs would be found by monitoring the L2 output oracle contract for new output commitments.
The `--l2.claim` and `--l2.blocknumber` inputs are taken from the L2 output data received from the output oracle.

The `--l1.head` input can be set to any L1 block hash that's at least one sequence window after the output commitment was published.
This ensures that any batches required to reproduce that output root have had time to be included on chain. Note that in most
cases the batches will have already been included on L1 before the output commitment is published.

To find the `--l2.head` input, search back through prior output commitments from the L2 output oracle to find the first
one that is correct. The `optimism_outputAtBlock` RPC method provided by `op-node` can be used to retrieve the expected
output root at a given block number.

#### Fetch Pre-Image Data

The fault proof program requires data from the L1 and L2 chains in order to run the derivation process. When running
locally, this data can be fetched from the JSON-RPC endpoints of the L1 and L2 nodes. However to run as part of the
dispute game on-chain, that pre-image data must be pre-populated into a pre-image oracle.

The fault proof program can pre-populate these pre-images by being run in "online" mode by specifying:
* the L1 JSON-RPC endpoint with the `--l1` option
* the L2 JSON-RPC endpoint with the `--l2` option
* the directory to store pre-image data in via the `--datadir` option

```shell
./bin/op-program \
  --datadir oracledata \
  --network=goerli \
  --l1=http://l1node.example.com:8545 \
  --l2=http://l2node.example.com:8545 \
  --l1.head=0x6a473d9d16fa1cbd1fa0feda8d49e18ab7536a8767add03ec883a59e4234eccf \
  --l2.head=0xaf121f101d24d2be26f4485c610aa3aa74ad07bedb5c35ce1b454f6c72630fad \
  --l2.claim=0xc946560741fabd57022a39ded469142db8909694dde7c2d4bdd1bda161a4d426 \
  --l2.blocknumber=6585388
```

Once the pre-image data has been pre-populated, the fault proof program can be run in "offline" mode by perform the same
verification process using only the data from the pre-image oracle by removing the `--l1` and `--l2` options:

```shell
./bin/op-program \
  --datadir oracledata \
  --network=goerli \
  --l1.head=0x6a473d9d16fa1cbd1fa0feda8d49e18ab7536a8767add03ec883a59e4234eccf \
  --l2.head=0xaf121f101d24d2be26f4485c610aa3aa74ad07bedb5c35ce1b454f6c72630fad \
  --l2.claim=0xc946560741fabd57022a39ded469142db8909694dde7c2d4bdd1bda161a4d426 \
  --l2.blocknumber=6585388
```

#### Verification Result

The result of verification is logged when `op-program` completes. When the claim is verified as correct `op-program`
logs `Claim successfully verified` and completes with an exit code of 0. If the claim is invalid, `op-program` logs
`Claim is invalid` and completes with an exit code of 1.

#### Demo Script

To simplify retrieving the required inputs, the `verifyGoerli.sh` script has been provided. It will run the fault proof
program in online mode to verify the latest published output commitment, starting 100 blocks before the output
commitment's block. It will then display the command to re-run the same derivation in offline mode.

```shell
./verifyGoerli.sh <L1 RPC URL> <L2 RPC URL>
```
