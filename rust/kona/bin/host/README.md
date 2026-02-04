# `kona-host`

kona-host is a CLI application that runs the [pre-image server][p-server] and [client program][client-program].

## Modes

**Host Modes**

| Mode     | Description                                                                   |
|----------|-------------------------------------------------------------------------------|
| `single` | Runs the preimage server + client program for a single-chain (pre-interop.)   |
| `super`  | Runs the preimage server + client program for a superchain cluster (interop.) |

**Preimage Server Modes**

| Mode     | Description                                                                                                                                                                             |
| -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `server` | Starts with the preimage server only, expecting the client program to have been invoked by the host process. This mode is intended for use by the FPVM when running the client program. |
| `native` | Starts both the preimage oracle and client program in a native process. This mode is useful for witness generation as well as testing.                                                  |

## Usage

```txt
kona-host is a CLI application that runs the Kona pre-image server and client program. The host
can run in two modes: server mode and native mode. In server mode, the host runs the pre-image
server and waits for the client program in the parent process to request pre-images. In native
mode, the host runs the client program in a separate thread with the pre-image server in the
primary thread.

Usage: kona-host [OPTIONS] <COMMAND>

Commands:
  single  Run the host in single-chain mode
  super   Run the host in super-chain (interop) mode
  help    Print this message or the help of the given subcommand(s)

Options:
  -v, --v...     Verbosity level (0-2)
  -h, --help     Print help
  -V, --version  Print version
```

[p-server]: https://specs.optimism.io/fault-proof/index.html#pre-image-oracle
[client-program]: https://specs.optimism.io/fault-proof/index.html#fault-proof-program
