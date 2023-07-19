# external_geth shim

This shim is an example of how to write an adapter for an external ethereum
client to allow for its use in the op-e2e tests.

## Invocation

Generally speaking, you can utilize this shim by simply executing:

```
make test-external-geth
```

The `Makefile` is structured such that if you duplicate this directory and
tweak this code, you may simply execute:

```
make test-external-<your-client>
```

and the execution should happen as well.

*NOTE:* The `Makefile` will generally _not_ rebuild this shim code unless the
corresponding go files have been modified, but it _will_ trigger rebuilding of
`op-geth` with every invocation of the test suite.  This is intentional as the
overhead of building the binary is relatively low, it's difficult to determine
if changes have occurred, and while iterating for development picking up
changes quickly and reliably is important.

## Arguments

*--init* The init argument is passed which triggers the creation of an
`op-geth` binary in this directory.  It is built using the dependencies
specified in the top level `go.mod` file.  The `--init` call is made once per
execution of the e2e test suite (if this shim is passed in as a command line
flag).

*--config <path>* The config path is specified as an option for normal test
invocations.  This points to a JSON file which contains details of the L2
environment to bring up (including the `genesis.json` path, the chain ID, the
JWT path, and a ready file path).  See the data structures in `external.go`
for more details.

## Operation

When passed the `--config` flag, this shim will first execute a process to
initialize the op-geth database.  Then, it will start the op-geth process
itself.  It watches the output of the process and looks for the lines
indicating that the HTTP server and Auth HTTP server have started up.  It then
reads the ports which were allocated (because the requested ports were
passed in as ephemeral via the CLI arguments).

## Generalization

This shim is included to help document an demonstrate the usage of the
external ethereum process e2e test execution.  It is configured to execute in
CI to help ensure that the tests remain compatible with external clients.

To create your own external test client, these files can likely be used as a
starting point, changing the arguments, log scraping, and other details.  Or,
depending on the client and your preference, any binary which is capable of
reading and writing the necessary JSON files should be sufficient (though
will be required to replicate some of the parsing and other logic encapsulated
here).
