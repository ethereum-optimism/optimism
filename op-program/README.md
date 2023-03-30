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
