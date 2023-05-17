# [services/addresses.go](./addresses.go)

Abstraction for interacting with contract addresses

- **Overview**

The key component here is the AddressManager interface. It defines methods to get addresses and instances of three contracts:

- L1StandardBridge
- StateCommitmentChain
- OptimismPortal

Two structs `LegacyAddresses` and `BedrockAddresses` implement the `AddressManager` interface. They represent two different configurations of an Ethereum-Optimism setup:

LegacyAddresses is meant for legacy networks where L1StandardBridge and StateCommitmentChain are used. It does not support OptimismPortal, calling this method on a LegacyAddresses object will result in a panic.
BedrockAddresses is for newer setups that use L1StandardBridge and OptimismPortal. It does not support StateCommitmentChain, calling this method will cause a panic.
Each struct has a constructor function (NewLegacyAddresses and NewBedrockAddresses) that takes an Ethereum client (bind.ContractBackend) and the addresses of the relevant contracts. These functions create and return an instance of AddressManager configured with the contracts at the given addresses.

