# withdrawal

The `withdrawal` tool provides utilities for performing withdrawals from an OP Stack.

All subcommands support the

## Usage

### init

The `init` subcommand supports sending a basic transaction to the `L2ToL1MessagePasser` contract to initiate a
withdrawal.

```shell
go run . init --l2 <l2-el-rpc> --private-key <private-key> [--value <wei>]
```

On success, the withdrawal transaction hash is reported which can be used with the `prove` and `finalize` subcommands
to complete the withdrawal.

### prove

The `prove` subcommand proves the withdrawal transaction against a published output root. It must be called after a
valid proposal is made for a L2 block at or after the initiating transaction was included.

```shell
go run . prove --l1 <l1-el-rpc> --l2 <l2-el-rpc> --tx <init-tx-hash> --portal-address <portal-addr> --private-key <private-key>
```

When proving super roots, you'll need to provide additional flags:

```
shell
go run . prove --l1 <l1-el-rpc> --l2 <l2-el-rpc> --tx <init-tx-hash> --portal-address <portal-addr> --private-key <private-key>\
  --supervisor <supervisor-rpc> --rollup.config <path-to-rollup-config> --depset <path-to-dependency-set-json>
```


### finalize

The `finalize` subcommand finalizes a withdrawal that has previously been proven. The dispute game must have resolved as
defender wins and the air gap period for the game and withdrawal proof must have elapsed. Normally this takes 7 days.

```shell
go run . finalize --l1 <l1-el-rpc> --l2 <l2-el-rpc> --tx <init-tx-hash> --portal-address <portal-addr> --private-key <private-key>
```
