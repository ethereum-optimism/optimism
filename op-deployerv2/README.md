# op-deployerv2

`op-deployerv2` is an experimental replacement design for `op-deployer`.

The core goal is to keep the tool independent from any specific OPCMv2 release.
The tool may hardcode user workflows like `upgrade`, but it must not hardcode
the Solidity input structs used by a particular OPCM version.

The intended model is that the core tool is pointed at a contracts commit or
release source. That source provides the ABIs, generated bindings, and
intent/state adapters for that contracts version. Those adapters are not built
into the core `op-deployerv2` release.

Current design notes:

- [Upgrade interface](docs/upgrade-interface.md)
- [op-deployer usage and transition analysis](docs/op-deployer-usage-transition.md)
- [Legacy CLI command map](docs/legacy-cli-command-map.md)
- [Native v2 CLI](docs/native-cli.md)
- [op-deployer v1 coupling inventory](docs/op-deployer-v1-coupling-inventory.md)
- [Generated binding pilot](docs/generated-binding-pilot.md)
- [Implementation plan](docs/implementation-plan.md)
