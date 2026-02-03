# RPC Types

The [`op-alloy-rpc-types`][rpc] crate contains RPC-related types.

The [`OpTransactionRequest`][req] type acts as a builder for
[`OpTypedTransaction`][typed].

[`Transaction`][tx] is a transaction type.

Related to receipts, [`op-alloy-rpc-types`][rpc] contains the
[`OpTransactionReceipt`][receipt] type and it's field types.


<!-- Links -->

[rpc]: https://crates.io/crates/op-alloy-rpc-types
[typed]: https://docs.rs/op-alloy-consensus/latest/op_alloy_consensus/transaction/enum.OpTypedTransaction.html
[tx]: https://docs.rs/op-alloy-rpc-types/latest/op_alloy_rpc_types/transaction/struct.Transaction.html
[req]: https://docs.rs/op-alloy-rpc-types/latest/op_alloy_rpc_types/receipt/struct.OpTransactionReceipt.html
[receipt]: https://docs.rs/op-alloy-rpc-types/latest/op_alloy_rpc_types/receipt/struct.OpTransactionReceipt.html
