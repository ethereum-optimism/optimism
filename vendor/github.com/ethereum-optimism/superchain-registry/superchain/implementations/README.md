# implementations

The implementation contract addresses live here. When deployed with a `CREATE2`
[deterministic deployment factory](https://github.com/Arachnid/deterministic-deployment-proxy),
the contract addresses will be the same on all networks given the same salt is used. These
contract addresses live in `implementations.yaml`. If the contract address differs on a per network
basis, it should exist in a `yaml` file named after the network inside of the `networks` directory.

