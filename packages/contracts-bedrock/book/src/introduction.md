# Introduction

This package contains the L1 and L2 smart contracts for the OP Stack. Detailed specifications for the contracts
contained within this package can be found at [specs.optimism.io][specs]. High-level information about these contracts
can be found within this book and within the [Optimism Developer Docs][docs]. For more information about contributing
to OP Stack smart contract development, read on in this book.

[specs]: https://specs.optimism.io
[docs]: https://docs.optimism.io

## Contributing

### Contributing Guide

Contributions to the OP Stack are always welcome. Please refer to the [CONTRIBUTING.md][contrib] for general information
about how to contribute to the OP Stack monorepo.

[contrib]: https://github.com/ethereum-optimism/optimism/blob/develop/CONTRIBUTING.md

When contributing to the `contracts-bedrock` package there are some additional steps you should follow. These have been
conveniently packaged into a just command which you should run before pushing your changes.

```bash
just pre-pr
```

### Style Guide

OP Stack smart contracts should be written according to the [style guide][style-guide] found within this book.
Maintaining a consistent code style makes code easier to review and maintain, ultimately making the development process
safer.

[style-guide]: ./contributing/style-guide.md

### Contract Interfaces

OP Stack smart contracts use contract interfaces in a relatively unique way. Please refer to the [interfaces guide]
[ifaces] to read more about how the OP Stack uses contract interfaces.

[ifaces]: ./contributing/interfaces.md

### Solidity Versioning

OP Stack smart contracts are designed to utilize a single, consistent Solidity version. Please refer to
the [Solidity upgrades][solidity-upgrades] guide to understand the process for updating to newer Solidity versions.

[solidity-upgrades]: ./policies/solidity-upgrades.md