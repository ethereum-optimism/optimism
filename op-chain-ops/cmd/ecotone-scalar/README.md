# ecotone-scalar

A CLI tool for computing the value of `scalar` to use after the Ecotone upgrade in a call to
`setGasConfig(scalar, overhead)` of the L1 `SystemConfigProxy` contract. After the Ecotone upgrade, the
overhead parameter is ignored, and the scalar parameter encodes a versioned `bytes32` that allows
configuring the base fee scalar and blob base fee scalars separately.

#### Usage

Build and run using the [Makefile](../../Makefile) `ecotone-scalar` target. Inside of `/op-chain-ops`, run:
```sh
make ecotone-scalar
```
to create a binary in [../../bin/ecotone-scalar](../../bin/ecotone-scalar) that can
be executed, providing the `--scalar` and `--blob-scalar` flags to specify the base bee scalar and
blob base fee parameters respectively, for example:

```sh
./bin/ecotone-scalar --scalar=7600 --blob-scalar=862000
```

You can also use the utility to decode a versioned value into its components:

```sh
./bin/ecotone-scalar --decode=452312848583266388373324160190187140051835877600158453279134021569375896653
```
