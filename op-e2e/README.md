# op-e2e

The end to end tests in this repo depend on genesis state that is
created with the `bedrock-devnet` package. To create this state,
run the following commands from the root of the repository:

```bash
make install-geth
make cannon-prestate
make devnet-allocs
```

This will leave artifacts in the `.devnet` directory that will be
read into `op-e2e` at runtime. The default deploy configuration
used for starting all `op-e2e` based tests can be found in
`packages/contracts-bedrock/deploy-config/devnetL1.json`. There
are some values that are safe to change in memory in `op-e2e` at
runtime, but others cannot be changed or else it will result in
broken tests. Any changes to `devnetL1.json` should result in
rebuilding the `.devnet` artifacts before the new values will
be present in the `op-e2e` tests.
