# op-deployer-types

`op-deployer-types` is the experimental typed authoring package for
source-backed OP Stack deployments.

The package is generated from contracts artifacts and a bindings manifest. It
is intended for Go callers that want compile-time structs for the contracts ref
they are targeting. Those structs should serialize into the same helper
protocol used by the `op-deployer` shell; they are not a separate deployment
engine.

Generate the package with:

```bash
go generate ./op-deployer-types
```
