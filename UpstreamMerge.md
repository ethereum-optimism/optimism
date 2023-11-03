There is typically a large amount of churn each week, and most merges, even
after a day or two are likely to generate conflicts.

After resolving the conflicts (see below for more details on conflict
resolution) the most useful way to verify basic functionality is to:

```
# In v3-anchorage
make nuke
make
make devnet-allocs
make cannon-prestate
make -C op-e2e test-external-erigon |& tee erigion-test.output
```

If failures are encountered, it's useful to check if the failing test was
introduced by the merge.  If so, simply add it to the `test_parms.json` file
in the external_erigon directory with a note that this is a new failure to be
investigated.

## Likely conflicts

### `.circleci/config.yaml`

This conflicts in this file will hopefully start reducing soon, now that the
GCP bits are proper parameters instead of hardcoded.  Still, we have
additional steps in the pipeline, such as the hardhat tests, the erigon tests,
and care should be taken to ensure that these additional jobs are not removed.

### `Makefile`

Similar to the circleci file, hopefully the churn here will be diminishing.
The extra targets to watch out for included the hardhat devnet and tests, as
well as the erigon build.

### `go.mod`/`go.sum`

For the `go.mod` generally the simplest thing to do is to remove all duplicate
dependencies (leaving possibly extraneous ones) while preferring the most up
to date version of the dependency.  Once done, re-run `go mod tidy` which will
rewrite both the `go.mod` and `go.sum`.  Note, any conflicts in `*.go` files
must be resolved before attempting to run the tidy.

### `packages/contracts-bedrock/package.json`

This file frequently has conflicts because of the additional hardhat
dependencies, and hardhat build targets.  Unfortunately there's no obvious
rote way to resolve these conflicts, so manual merging is necessary.

### `ops-bedrock/docker-compose.yml`

The primary modification here is the addition of the KMS ops bits, as well as
replacing of all of the oplabs references with the generic GCP variables.
Hopefully we can upstream the GCP bits at least soon.

### op-bindings/bindings/*

First ensure that you have the correct ci-builder image available.  Generally,
you can do this by running:

```
make ci-builder
```

Then regenerate the bindings by running:

```
 docker run -v $PWD:/optimism --rm ci-builder:latest 'make -C /optimism/op-bindings'
```

Note, locally on my system for some unexplained reason, this call modifies the
`weth9` contracts erroneously.  We are investigating, but, manually reverting
the changes to those contracts with:

```
git checkout op-bindings/bindings/weth9.go op-bindings/bindings/weth9_more.go
```

may be necessary.

Next, it's important to regenerate `boba-bindings/bindings/*`

Although there will generally never be any merge conflicts because these files
do no exist upstream, any conflicts in the op-bindings indicates that
boba-bindings must be regenerated as well.  This is done in a similar fashion
with:

```
 docker run -v $PWD:/optimism --rm ci-builder:latest 'make -C /optimism/boba-bindings'
```

Similar with the op-bindings, any changes to the weth9 contracts should be
reverted:

```
git checkout boba-bindings/bindings/weth9.go boba-bindings/bindings/weth9_more.go
```

Note: After running these docker commands, it's sometimes necessary to fix
file ownership with a `chown -R`.  Alternatively, the docker commands can be
run as your user.

### `pnpm-lock.yaml`

This file invariably has collisions.  Generally, they can be addressed via a
simple:

```
git checkout --theirs pnpm-lock.yaml
pnpm install --no-frozen-lockfile
```

### Patch failures

We have multiple builds where we expect to patch files before building.  In
particular, this occurs with the open-zeppelin contracts, as well as with the
hardhat build.  If patch failures occur, check to see if the version of these
dependencies has change, and update the patch as needed.
