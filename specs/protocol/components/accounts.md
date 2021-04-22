# Accounts

This document is a WIP. It is kept here to maintain a consistent structure mirroring the [implementation directory](../../../contracts/contracts/optimistic-ethereum/OVM/).

<!--
### ProxyEOAs

- Created by call to ovmCREATEEOA with signature
- `ecrecover` is used to retrieve address and deploy the trusted ProxyEOA contract, which implements a simple DelegateCall pattern there.

### OVM_ECDSAContractAccount

The ECDSA Contract Account can be used as the implementation for a ProxyEOA deployed by the
ovmCREATEEOA operation. It enables backwards compatibility with Ethereum's Layer 1, by
providing EIP155 formatted transaction encodings.

#### Value Transfer

Value transfer is currently only supported in the first call frame of a transaction on L2. -->
