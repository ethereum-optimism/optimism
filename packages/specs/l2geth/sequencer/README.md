# Sequencer specs

## Contents

- [Batch Submitter](./batch-submitter.md) (empty file)
- [Fees](./fees.md) (empty file)

### Flow of a tx sent to a Sequencer

1. An L2 transaction is submitted to [l2geth](https://github.com/ethereum-optimism/go-ethereum)'s RPC server.
2. The transaction then sent to the [`Sync Service`](https://github.com/ethereum-optimism/specs/blob/main/transaction-ingestor.md)'s (aka `transaction-ingestor`) `applyTransaction(..) function.
3. The raw transaction data is supplied as the input to the [`run(...)`](https://github.com/ethereum-optimism/contracts/blob/master/contracts/optimistic-ethereum/OVM/execution/OVM_ExecutionManager.sol#L155-L158) function in the ExecutionManager.
4. The block including a state root, transaction receipt, and transaction are all stored in Geth's default `EthDB`.
5. The batch submitter periodically queries L1 & L2Geth for unsubmitted transactions & if any are detected queries L2Geth for a range of blocks.
6. The GetBlock endpoint gets the block & transaction data from the EthDB.
7. Blocks are returned to the batch submitter.
8. The batch submitter generates a `appendSequencerBatch` transaction to the CTC on L1 and broadcasts it to its local L1Geth node. From there the transactions makes it into the Eth mainchain for availability & ordering.

<pre>
                                        Ethereum Mainnet
                                      ┌──────────────────────────────┐
                                      │┼────────────────────────────┼│
                                      ││ Ethereum L1 Mempool/Miners ││
                                      │┼────────────────────────────┼│
                                      └────────────▲─────────────────┘
                                                   │
                   Sequencer                       │
                  ┌────────────────────────────────┼─────────────────────────────────────┐
                  │                                │                                     │
                  │                        ┌───────▼────────┐ 8)AppendBatch (many blocks)│
                  │                        │     L1Geth     ◄──────────────┐             │
                  │                        └────────────────┘              │             │
                  │       L2Geth                                           │             │
                  │    ┌────────────────────────────────────┐              │             │
                  │    │                                    │              │             │
                  │    │ ┌──────────────────────────┐  5)Get│Blocks ┌──────┴────────┐    │
                  │    │ │           <a href="https://github.com/ethereum-optimism/go-ethereum/blob/master/internal/ethapi/api.go">RPC</a>            ◄───────┼───────┤               │    │
                  │    │ │            │             │     7)│Blocks │               │    │
1)Transaction┌────┼────┼─► SendTx(..) │GetBlock(..) ├───────┼───────►<a href="/sequencer/batch-submitter.md">Batch Submitter</a>│    │
 ────────────┘    │    │ └──────┬─────┴─────┬──▲────┘       │       └───────────────┘    │
                  │    │    2)Tx│           │6)│Get Block   │                            │
                  │    │        │       ┌───▼──┴────┐       │                            │
                  │    │ ┌──────▼─────┐ │           │       │                            │
                  │    │ │<a href="/l2-geth/transaction-ingestor.md">Sync Service</a>│ │   <a href="https://github.com/ethereum-optimism/go-ethereum/tree/83593f20c213129f6dceac6321e7cbbad0035a26/core/rawdb">EthDB</a>   │       │                            │
                  │    │ └──────┬─────┘ │           │       │                            │
                  │    │        │       └─────▲─────┘       │                            │
                  │    │    3)Tx│           4)│Tx, Receipts │                            │
                  │    │        │             │State root   │                            │
                  │    │ ┌──────▼─────────────┴───┐         │                            │
                  │    │ │          <a href="/ovm/README.md">OVM</a>           │         │                            │
                  │    │ └────────────────────────┘         │                            │
                  │    │                                    │                            │
                  │    └────────────────────────────────────┘                            │
                  │                                                                      │
                  └──────────────────────────────────────────────────────────────────────┘
</pre>

### Verifier Syncing from Mainnet

1. Batches full of L2 transactions are pulled in from an L1Geth node syncing mainnet Ethereum.
2. Each transaction is applied to the Sync Service (AKA the `transaction-ingestor`).
3. The Sync Service pipes the transaction into the OVM (using Geth's Clique Miner).
4. The OVM executes the transaction and stores the resulting state & blocks in the EthDB.
5. The Fraud Prover service pulls all of the recent state roots from L2Geth.
6. The RPC returns state roots stored in the EthDB that were originally stored by the OVM.
7. Now with all of the Verifier computed state roots, the Fraud Prover pulls all of the proposed state roots on L1. It compares the state roots that were computed **locally** against the **proposed** state roots in the State Commitment Chain. _UH OH_ the proposed state root DOES NOT equal the locally computed (and therefore correct) state root! We need to prove fraud!
8. The Fraud Prover requests a full fraud proof from the L2Geth node. This includes the contracts and storage slots that were accessed during the execution of the transaction.
9. The Fraud Prover signs Ethereum transactions and executes a fraud proof on L1! Claiming the bond of the state root proposer.

Yay the chain is secured by L1 and cryptoeconomics!

<pre>
               Ethereum Mainnet
             ┌──────────────────────────────┐
             │┼────────────────────────────┼│
             ││ Ethereum L1 Mempool/Miners ││
             │┼────────────────────────────┼│
             └────────────▲─────────────────┘
 Verifier                 │
┌─────────────────────────┼───────────────────────────────────┐
│                         │                                   │
│                 ┌───────▼────────┐  9)Execute Fraud proof   │
│                 │     L1Geth     │◄─────────────────────┐   │
│                 └─┬────┬─────────┘                      │   │
│                   │    │7)State Commitment Chain        │   │
│                   │    └───────────────────────┐        │   │
│  L2Geth           │                            │        │   │
│ ┌─────────────────┼───────────────────────┐ ┌──▼────────┴─┐ │
│ │               1)│Batches                │ │<a href="/verifier/fraud-prover.md">Fraud Prover</a> │ │
│ │                 │            ┌─────────┬┤ └─▲─────────▲─┘ │
│ │         ┌───────▼────────┐   │   <a href="https://github.com/ethereum-optimism/go-ethereum/blob/master/internal/ethapi/api.go">RPC</a>   ││   │         │   │
│ │         │<a href="/l2-geth/l1-data-indexer.md">Tx Indexer (DTL)</a>│   │         ◄├───┘         │   │
│ │         └──┬─────────────┘   └▲───────┬─┤5)Get State  │   │
│ │        2)Tx│                6)│State  │ │  roots      │   │
│ │            │                  │roots, │ │             │   │
│ │            │                  │       └─┼─────────────┘   │
│ │     ┌──────▼─────┐ ┌──────────▼┐        │    8)Fraud      │
│ │     │<a href="/l2-geth/transaction-ingestor.md">Sync Service</a>│ │   <a href="https://github.com/ethereum-optimism/go-ethereum/tree/83593f20c213129f6dceac6321e7cbbad0035a26/core/rawdb">EthDB</a>   │        │      proof      │
│ │     └──────┬─────┘ └─────▲─────┘        │      data       │
│ │        3)Tx│           4)│Tx, Receipts, │                 │
│ │            │             │State roots   │                 │
│ │     ┌──────▼─────────────┴───┐          │                 │
│ │     │           <a href="/ovm/README.md">OVM</a>          │          │                 │
│ │     └────────────────────────┘          │                 │
│ │                                         │                 │
│ └─────────────────────────────────────────┘                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
</pre>
