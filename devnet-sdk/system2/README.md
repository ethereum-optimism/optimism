# System2

## Design choices

- Interfaces FIRST. Composes much better.
- Incremental system composition.
- Type-safety is important. Internals may be more verbose where needed.
- Everything is a resource and has a typed ID
- Embedding and composition de-dup a lot of code.
- Avoid generics in method signatures, these make composition of the general base types through interfaces much harder.
- Each component has access to commons such as logging and a test handle to assert on.
  - TODO: test handle part needs more distinguished sub-problems and choices, test-handle can be bad in some places
- Option pattern for each type, taking the interface, so that the system can be composed by external packages, eg:
  - Kurtosis
  - System like op-e2e
  - Action-test
- Implementations should take `client.RPC` (or equivalent), not raw endpoints. Dialing is best done by the system composer, which can customize retries, in-process RPC pipes, lazy-dialing, etc. as needed.
- The system composer is responsible for tracking raw RPC URLs. These are not portable, and would expose too much low-level detail in the System interface.
- The system compose is responsible for the lifecycle of each component. E.g. kurtosis will keep running, but an in-process system will couple to the test lifecycle and shutdown via `t.Cleanup`.
- Test gates can inspect a system, abort if needed, or remediate shortcomings of the system with help of an `Orchestrator` (the test setup features offered by the system composer or maintainer).
- The `Setup` is a struct that may be extended with additional struct-fields in the future, without breaking the `Option` function-signature.

## Overview

TODO


## Next steps

- Make default component implementations easier to instantiate (rpc/preset structs)
- Complete system op-e2e composer
- Complete kurtosis composer
- Complete action-test composer

## Side-quests

- Improve test-logger:
  - Allow log-level changes via system2.Common, so that each component can be more/less verbose as needed during a test.
  - Extend the logger (in op-geth and the testlogger) with context-logging.
- Move all API interfaces into the `sources` package, and assert that the frontend and client bindings implement the same interfaces.

