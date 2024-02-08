# external_erigon shim

This shim is a close cousin of the external geth shim.  (It was actually
authored first, but, does not fit nicely in tree so is not the example).

## Invocation

Generally speaking, you can utilize this shim by simply executing:

```
make test-external-erigon
```

## Arguments

*--config <path>* The config path is specified as an option for normal test
invocations.  This points to a JSON file which contains details of the L2
environment to bring up (including the `genesis.json` path, the chain ID, the
JWT path, and a ready file path).  See the data structures in `external.go`
for more details.

## Operation

When passed the `--config` flag, this shim will first execute a process to
initialize the op-erigon database.  Then, it will start the op-erigon process
itself.  It watches the output of the process and looks for the lines
indicating that the HTTP server and Engine server have started up.  It then
reads the ports which were allocated (because the requested ports were
passed in as ephemeral via the CLI arguments).  Finally, it watches for the
staged sync to complete before writing the ready file JSON.
