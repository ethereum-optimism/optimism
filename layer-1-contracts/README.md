# Layer 1 Contracts

## Design Goals
We want our ORU contracts to be built in compliance with the following goals:
- A **pluggable state machine** which:
    - allows separation of concerns and parallelizable work
    - is fully stateless, i.e. all state transitions can be expressed in the form `bytes32 _inputHash, bytes _witness => bytes32 _outputhash` and are only dependent on the immediately preceding pre-state
- A cryptoeconomic **State Oracle** which:
    - Provides a cryptoeconomic mechanism to propose, assert, challenge, and finalize optimistic states of the L2 machine
        - Does NOT require that the entire execution trace is provable, just the execution output
            - i.e. an honest party may not necessarily ever post a hash of the full trace
        - DOES require that there is a winning strategy for all, and only for all, possible state proposals
    - Provides "useful" state hash proposals, i.e. block roots or state roots, which may not be equal to the `_inputHash`es above
    - Is optimized to minimize the number of rounds of interaction, so as to be most resistant to hostile L1 censorship and griefing
- A set of **data feed** contracts which:
    - allow parties and contracts to provide inputs which are processed by the L2 machine
    - allow verifiability of those inputs for disputes
    - is highly gas-efficient
- A useful **message-passing abstraction** which can be used to easily send messages (e.g. deposits and withdrawals) between L1 and L2 with a consistent interface
