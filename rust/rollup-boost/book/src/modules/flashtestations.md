# Flashtestations

Flashtestations is a rollup-boost module that provides onchain TEE (Trusted Execution Environment) attestations and block proofs to verifiably prove that blocks were built inside a TEE. This provides user guarantees such as priority ordering.

## Architecture

```text
+-------------------------+                  +---------------------+
| TDX VM                  |                  | Onchain Verifier    |
|                         |  Attestation     |                     |
| +-----------------+     |  Quote           | +-----------------+ |
| | TEE Workload    |     | ---------------> | | DCAP Attestation| |
| |                 |     |                  | | Verifier        | |
| | (measurements)  |     |                  | |                 | |
| +-----------------+     |                  | +--------+--------+ |
|                         |                  |          |          |
+-------------------------+                  |          v          |
                                             | +-----------------+ |
+-------------------------+                  | | Intel           | |
| Consumer Contract       |                  | | Endorsements    | |
|                         |                  | |                 | |
| +-----------------+     |                  | +--------+--------+ |
| | Operation       |     |                  |          |          |
| | Authorization   |     |                  |          v          |
| +-----------------+     |                  | +-----------------+ |
|         |               |                  | | Registration    | |
+---------+---------------+                  | | Logic           | |
          |                                  | +--------+--------+ |
          |                                  +----------+----------+
          |                                             |
+---------+---------------+                             v
| Policy                  |                  +---------------------------+
|                         |  isValid         | Flashtestation Registry   |
| +---------------------+ |  Query           |                           |
| | allowedWorkloadIds[]| | <--------------> | {teeAddress: registration}|
| | {registration:      | |                  |   map                     |
| |   workloadId} map   | |                  |                           |
| +---------------------+ |                  +---------------------------+
+-------------------------+
```

### Core Components

1. **Onchain Verifier**: Validates TDX attestation quotes against current Intel endorsements. Provides cryptographic proof that a TEE-controlled address is generated within genuine TDX hardware
2. **Flashtestation Registry**: Tracks TEE-controlled addresses with valid attestations
3. **Policy Registry**: Defines which workloads are acceptable for specific operations
4. **Transparency Log**: Records all attestation events and endorsement changes

## Flashtestations Workflow

Flashtestations involve two main workflows:

- Registering the block builder's TEE attestation 
- Block builder TEE proof

### TEE Attestation Registration

1. **TEE Environment Setup**: The builder runs in a TDX environment with specific measurement registers. These are measurements of a reproducible build for a specific builder code commit and config.
2. **TEE Key Generation**: The builder generates a key pair on startup that never leaves the TEE environment
3. **Quote Generation**: The TEE generates an attestation quote containing:
   - Measurement registers
   - Report data with TEE-controlled address and any extra registration data
3. **Onchain Verification**: The quote is submitted onchain by the builder to the registry contract. The contract verifies that the quote is valid and the TEE address from the report data

### Policy Layer

The policy layer provides authorization for TEE services. This allows operators to authorize that only builders running a specific commit and config can submit TEE block proofs. 

Upon registration, a workloadId is derived from the measurements in the quote and operators can manage which workload ids are considered valid for a specific rollup.

The Policy Registry can store metadata linking workload IDs to source code:

```solidity
struct WorkloadMetadata {
    string commitHash;        // Git commit hash of source code
    string[] sourceLocators;  // URIs pointing to source code (https://, git://, ipfs://)
}
```

### Block Builder TEE Proofs

If the builder has registered successfully with an authorized workload id, builders can append a verification transaction to each block.

The block proof will be signed by the generated TEE address. Since the TEE address never leaves the TEE environment, we can ensure that block proofs signed by that key mean that the block was built inside a TEE.

The builder will submit a transaction containing a block content hash, which is computed as:

```solidity
function ComputeBlockContentHash(block, transactions) {
    transactionHashes = []
    for each tx in transactions:
        txHash = keccak256(rlp_encode(tx))
        transactionHashes.append(txHash)
    
    return keccak256(abi.encode(
        block.parentHash,
        block.number,
        block.timestamp,
        transactionHashes
    ))
}
```

## Security Considerations

### Critical Security Assumptions

- **Private Key Management**: TEE private keys must never leave the TEE boundaries
- **Attestation Integrity**: TEEs must not allow external control over `ReportData` field
- **Reproducible Builds**: Workloads must be built using reproducible processes

### Security Properties

- **Block Authenticity**: Cryptographic proof that blocks were produced by authorized TEEs with specific guarantees with the workload id
- **Auditability**: All proofs are recorded onchain for transparency

## References

- [Intel TDX Specifications](https://www.intel.com/content/www/us/en/developer/tools/trust-domain-extensions/documentation.html)
- [Intel DCAP Quoting Library API](https://download.01.org/intel-sgx/latest/dcap-latest/linux/docs/Intel_TDX_DCAP_Quoting_Library_API.pdf)
- [Automata DCAP Attestation Contract](https://github.com/automata-network/automata-dcap-attestation)
- [Automata On-chain PCCS](https://github.com/automata-network/automata-on-chain-pccs)
