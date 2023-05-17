# [services/query/erc20.go](./erc20.go)

Provides a function NewERC20 which is used to create a new instance of an ERC20 token contract and retrieve its name, symbol, and decimal properties. 

- **Overview**

The function takes as input the address of the ERC20 contract and an Ethereum client instance.

It first attempts to create a new instance of the ERC20 contract using the provided address and client. If there's an error at this stage, it's returned immediately.

It then calls Name, Symbol, and Decimals functions on the contract instance. These functions are part of the ERC20 token standard, and they return the name, symbol, and the number of decimal places of the token respectively. If there's an error calling any of these functions, it's returned immediately.

Finally, it creates a db.Token struct with the retrieved name, symbol, and decimal values, and returns it.

