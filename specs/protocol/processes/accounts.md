# Account Abstraction

**WIP:** This is currently just a disorganized place to dump notes.


### ProxyEOAs

- Created by call to ovmCREATEEOA with signature
- ecrecover is used to retrieve address and deploy the trusted ProxyEOA contract, which implements a simple DelegateCall pattern there.
-


### OVM_ECDSAContractAccount

The ECDSA Contract Account can be used as the implementation for a ProxyEOA deployed by the
ovmCREATEEOA operation. It enables backwards compatibility with Ethereum's Layer 1, by
providing EIP155 formatted transaction encodings.

#### Value Transfer
<!-- Maybe this should go into execution.md -->

Value transfer is currently only supported in the