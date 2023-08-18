# boba-bindings

This package contains built go bindings of the smart contracts. It must be
updated after any changes to the smart contracts to ensure that the bindings are
up to date.

The bindings include the bytecode for each contract so that go based tests
can deploy the contracts. There are also `more` files that include the deployed
bytecode as well as the storage layout. These are used to dynamically set
bytecode and storage slots in state.

**Note**

The `registry.go` file is added manually. The other binding files including `_more.go` files are created by the code.
