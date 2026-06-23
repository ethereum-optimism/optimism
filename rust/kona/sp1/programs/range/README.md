# `range`

This binary contains the client program for executing the Optimism rollup state transition across a range of blocks, which can be used to generate an on chain validity proof. Depending on the compilation pipeline, it will compile to be run either in native mode or in zkVM mode. In native mode, the data for verifying
the batch validity is fetched from RPC, while in zkVM mode, the data is supplied by the host binary to the verifiable program.