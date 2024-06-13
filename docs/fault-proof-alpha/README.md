## Fault Proofs Alpha

The fault proof alpha is a pre-release version of the OP Stack fault proof system.
This documentation provides an overview of the system and instructions on how to help
test the fault proof system.

The overall design of this system along with the APIs and interfaces it exposes are not
finalized and may change without notice.

### Getting Started

* [Architecture Overview Video](https://www.youtube.com/watch?v=nIN5sNc6nQM)
* [Fault Proof Alpha Deployment Information (Goerli)](./deployments.md)
* [Security Researchers - Bug Bounty Program](./immunefi.md)

### Contents

 * Specifications
   * [Generic Fault Proof System](https://github.com/ethereum-optimism/specs/blob/main/specs/fault-proof/index.md)
   * [Generic Dispute Game Interface](https://github.com/ethereum-optimism/specs/blob/main/specs/fault-proof/stage-one/dispute-game-interface.md)
   * [Fault Dispute Game](https://github.com/ethereum-optimism/specs/blob/main/specs/fault-proof/stage-one/fault-dispute-game.md)
   * [Cannon VM](https://github.com/ethereum-optimism/specs/blob/main/specs/fault-proof/cannon-fault-proof-vm.md)
 * [Deployment Details](./deployments.md)
 * [Manual Usage](./manual.md)
   * [Creating Traces with Cannon](./cannon.md)
 * [Automation with `op-challenger`](./run-challenger.md)
 * [Challenging Invalid Output Proposals](./invalid-proposals.md)
