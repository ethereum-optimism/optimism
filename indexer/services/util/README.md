# [services/util/util.go](./util.go)

Provides a helper function ToBlockNumArg that converts a given big.Int representing a block number into a string that can be used as an argument in Ethereum JSON-RPC calls.

- **Overview**

The ToBlockNumArg function checks if the input number is nil or -1, and returns the string "latest" or "pending" respectively. These are special block identifiers used in Ethereum JSON-RPC calls. If the input number is neither nil nor -1, it encodes the number into a hexadecimal string using hexutil.EncodeBig function from the go-ethereum common package.

Here are the meanings of the special block identifiers:

"latest": Refers to the latest block that the node knows about, which is the tip of the chain.
"pending": Refers to the upcoming block that is being assembled by the node.
This function is useful when calling Ethereum JSON-RPC methods such as eth_call, eth_getBalance, etc., that require a block identifier as one of their parameters.

