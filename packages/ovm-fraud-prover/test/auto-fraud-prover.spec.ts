import './setup'

/* External Imports */
import * as path from 'path'
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer, Wallet } from 'ethers'
import { keccak256 } from '@ethersproject/keccak256'
import { deployAllContracts } from '@eth-optimism/rollup-contracts'

/* Internal Imports */
import { AutoFraudProver } from '../src/auto-fraud-prover'
import {
  getContractFactory,
  getContractFactoryFromDefinition,
  transpile,
  TxChainBatch,
  StateChainBatch,
  getWallets,
  FORCE_INCLUSION_PERIOD,
  GAS_LIMIT,
  encodeTransaction,
  signAndSendOvmTransaction,
  makeOvmTransaction,
  getStateTrieProof,
} from './test-helpers'
import { OVMTransactionData } from '../src/interfaces'

const appendTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number> => {
  const timestamp = Math.floor(Date.now() / 1000)

  await canonicalTransactionChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp, {
      gasLimit: GAS_LIMIT,
    })

  return timestamp
}

const appendAndGenerateTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  transactions: OVMTransactionData[]
): Promise<TxChainBatch> => {
  const batch = transactions.map((transaction) => {
    return encodeTransaction(transaction)
  })

  const batchIndex = await canonicalTransactionChain.getBatchesLength()
  const cumulativePrevElements = await canonicalTransactionChain.cumulativeNumElements()

  const timestamp = await appendTransactionBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

  const localBatch = new TxChainBatch(
    timestamp,
    false,
    batchIndex,
    cumulativePrevElements,
    batch
  )

  await localBatch.generateTree()

  return localBatch
}

const appendAndGenerateStateBatch = async (
  stateCommitmentChain: Contract,
  batch: string[]
): Promise<StateChainBatch> => {
  const batchIndex = await stateCommitmentChain.getBatchesLength()
  const cumulativePrevElements = await stateCommitmentChain.cumulativeNumElements()

  await stateCommitmentChain.appendStateBatch(batch, {
    gasLimit: GAS_LIMIT,
  })

  const localBatch = new StateChainBatch(
    batchIndex,
    cumulativePrevElements,
    batch
  )

  await localBatch.generateTree()

  return localBatch
}

const getCurrentStateRoot = async (provider: any): Promise<string> => {
  const currentBlock = await provider.send('eth_getBlockByNumber', [
    'latest',
    false,
  ])
  return currentBlock.stateRoot
}

describe('AutoFraudProver', () => {
  const provider = ethers.provider

  let wallet: Wallet
  let sequencer: Wallet
  let l1ToL2TransactionPasser: Wallet
  before(async () => {
    ;[wallet, sequencer, l1ToL2TransactionPasser] = getWallets(provider)
  })

  let resolver
  before(async () => {
    const config = {
      signer: wallet as any,
      rollupOptions: {
        gasLimit: GAS_LIMIT,
        forceInclusionPeriod: FORCE_INCLUSION_PERIOD,
        owner: wallet as any,
        sequencer: sequencer as any,
        l1ToL2TransactionPasser: l1ToL2TransactionPasser as any,
      },
    }

    resolver = await deployAllContracts(config)
  })

  let ExecutionManager: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  let FraudVerifier: ContractFactory
  before(async () => {
    ExecutionManager = getContractFactory('ExecutionManager', wallet)
    StateCommitmentChain = getContractFactory('StateCommitmentChain', wallet)
    CanonicalTransactionChain = getContractFactory(
      'CanonicalTransactionChain',
      wallet
    )
    FraudVerifier = getContractFactory('FraudVerifier', wallet)
  })

  let executionManager: Contract
  let canonicalTransactionChain: Contract
  let stateCommitmentChain: Contract
  let fraudVerifier: Contract
  before(async () => {
    executionManager = resolver.contracts.executionManager
    canonicalTransactionChain = resolver.contracts.canonicalTransactionChain
    stateCommitmentChain = resolver.contracts.stateCommitmentChain
    fraudVerifier = resolver.contracts.fraudVerifier
  })

  let FraudTester: ContractFactory
  before(async () => {
    const fraudTesterDefinition = transpile(
      path.resolve(__dirname, './test-contracts/FraudTester.sol'),
      executionManager.address
    ).FraudTester
    FraudTester = getContractFactoryFromDefinition(
      fraudTesterDefinition,
      wallet
    )
  })

  let fraudTester: Contract
  before(async () => {
    fraudTester = await FraudTester.deploy()
  })

  let transactions: OVMTransactionData[]
  before(async () => {
    transactions = [
      makeOvmTransaction(fraudTester, wallet, 'setStorage', [
        keccak256('0x0123'),
        keccak256('0x4567'),
      ]),
      makeOvmTransaction(fraudTester, wallet, 'setStorage', [
        keccak256('0x0123'),
        keccak256('0x4567'),
      ]),
    ]
  })

  const stateTrieProofs: any[] = []
  const transactionBatches: TxChainBatch[] = []
  const preStateBatches: StateChainBatch[] = []
  before(async () => {
    let preState = [await getCurrentStateRoot(provider)]

    for (const transaction of transactions) {
      stateTrieProofs.push(await getStateTrieProof(transaction.ovmEntrypoint))
      stateTrieProofs.push(await getStateTrieProof(wallet.address))

      await signAndSendOvmTransaction(wallet, transaction)

      const postState = [await getCurrentStateRoot(provider)]

      transactionBatches.push(
        await appendAndGenerateTransactionBatch(
          canonicalTransactionChain,
          sequencer,
          transactions
        )
      )

      preStateBatches.push(
        await appendAndGenerateStateBatch(stateCommitmentChain, preState)
      )

      preState = postState
    }
  })

  describe('prove', () => {
    let autoFraudProver: AutoFraudProver
    beforeEach(async () => {
      autoFraudProver = new AutoFraudProver(
        0,
        preStateBatches[0].elements[0],
        await preStateBatches[0].getElementInclusionProof(0),
        preStateBatches[1].elements[0],
        await preStateBatches[1].getElementInclusionProof(0),
        transactions[0],
        await transactionBatches[0].getElementInclusionProof(0),
        stateTrieProofs.slice(0, 2).map((stateTrieProof) => {
          return {
            root: stateTrieProof.root,
            proof: stateTrieProof.proof,
            ovmContractAddress: stateTrieProof.address,
            codeContractAddress: stateTrieProof.address,
            value: {
              nonce: stateTrieProof.account.nonce,
              balance: stateTrieProof.account.balance,
              codeHash: stateTrieProof.account.codeHash,
              storageRoot: stateTrieProof.account.stateRoot,
            },
          }
        }),
        wallet,
        fraudVerifier
      )
    })

    it('should handle the complete fraud proof process', async () => {
      await autoFraudProver.prove()
    })
  })
})
