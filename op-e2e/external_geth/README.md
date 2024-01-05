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

*NOTE:* Attempting to iterate for development requires explicit rebuilding of
the binary being shimmed.  Most likely to accomplish this, you may want to add
initialization code to the TestMain of the e2e to build your binary, or use
some other technique like custom build scripts or IDE integrations which cause
the binary to be rebuilt before executing the tests.

## Arguments

*--config <path>* The config path is a required argument, it points to a JSON
file that contains details of the L2 environment to bring up (including the
`genesis.json` path, the chain ID, the JWT path, and a ready file path).  See
the data structures in `op-e2e/external/config.go` for more details.

## Operation

This shim will first execute a process to initialize the op-geth database.
Then, it will start the op-geth process itself.  It watches the output of the
process and looks for the lines indicating that the HTTP server and Auth HTTP
server have started up.  It then reads the ports which were allocated (because
the requested ports were passed in as ephemeral via the CLI arguments).

## Skipping tests

Although ideally, all tests would be structured such that they may execute
either with an in-process op-geth or with an extra-process ethereum client,
this is not always the case.  You may optionally create a `test_parms.json`
file in the `external_<your-client>` directory, as there is in the
`external_geth` directory which specifies a map of tests to skip, and
accompanying skip text.  See the `op-e2e/external/config.go` file for more
details.

## Generalization

This shim is included to help document and demonstrates the usage of the
external ethereum process e2e test execution.  It is configured to execute in
CI to help ensure that the tests remain compatible with external clients.

To create your own external test client, these files can likely be used as a
starting point, changing the arguments, log scraping, and other details.  Or,
depending on the client and your preference, any binary which is capable of
reading and writing the necessary JSON files should be sufficient (though
will be required to replicate some of the parsing and other logic encapsulated
here).
