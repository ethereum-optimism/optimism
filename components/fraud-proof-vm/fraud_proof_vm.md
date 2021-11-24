# Fraud Proof VM

A fraud proof VM (FPVM) is a virtual machine which can be evaluated on-chain to settle a disputed block hash assertion. In this spec we assume the fraud proof VM is evaluated using a multi-round dispute game similar to [Truebit](https://people.cs.uchicago.edu/~teutsch/papers/truebit.pdf).

## Key Properties

A Fraud proof VM must satisfy the following properties to be suitable for a multi-round dispute game:
1. Program execution can be split up into deterministic state transitions.
2. Each state transition can be evaluated on-chain with less than or equal to a constant size gas allocation (eg. 10 million L1 gas).
3. The FPVM must be executable off-chain in such a way that state checkpoints (ie. merklized commitments to the VM state) can be generated.

## Usage

### Initialization

The fraud proof VM is initialized with:
1. an executable binary; and
2. an initialization hash for the preimage oracle.

The executable binary defines the program which is executed during a fraud proof, and the initialization hash is used to load data from the FPVM's database called the `Preimage Oracle`.


The fraud proof VM does not have access to a traditional database. Instead, it has access to the "preimage oracle". The preimage oracle is simply a database which exposes the function `get_preimage(hash: bytes)`:

```python
def get_preimage(hash: bytes) -> bytes:
    # The preimage is looked up from a local database held by the fraud prover
    preimage = fetch_preimage(hash)
    assert keccak256(preimage) == hash
    return preimage
```

The preimage oracle must be populated with all `hash->preimage` mappings before they are queried by the FPVM. These preimages are committed to in the initialization hash which is supplied as one of the two inputs to the FPVM. During the course of execution, the FPVM must **never** request a preimage which is unavailable as that would result in the FPVM being indeterminate. In this way, a valid executable binary must account for the structure of the initialization hash and only query available preimages.

### Termination

The fraud proof VM must terminate with either:

1. SUCCESS
2. FAIL
3. INDETERMINATE

If the FPVM returns SUCCESS it may also return a 32 byte value (eg. the valid block hash). If the FPVM returns FAIL it must be handled by the dispute game.

There are a few conditions under which the FPVM may return FAIL:

1. Out of memory -- Too much memory was used during execution.
2. Too many instructions -- Too many instructions were consumed (there is a fixed max instructions to prevent DOS attacks on the FPVM).

The only time in which the FPVM returns INDETERMINATE is when an unavailable preimage is queried.