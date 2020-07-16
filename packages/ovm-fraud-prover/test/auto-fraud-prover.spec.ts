import './setup'

/* External Imports */
import * as path from 'path'
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer, BigNumber, Wallet } from 'ethers'
import { keccak256 } from '@ethersproject/keccak256'

/* Internal Imports */
import { AutoFraudProver } from '../src/auto-fraud-prover'
import {
  getContractFactory,
  getContractFactoryFromDefinition,
  transpile,
  TxChainBatch,
  StateChainBatch,
  getWallets,
  DEFAULT_OPCODE_WHITELIST_MASK,
  NULL_ADDRESS,
  FORCE_INCLUSION_PERIOD,
  GAS_LIMIT,
  encodeTransaction,
  signAndSendOvmTransaction,
  makeOvmTransaction,
  getStateTrieProof
} from './test-helpers'
import {
  OVMTransactionData
} from '../src/interfaces'

const appendTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number> => {
  const timestamp = Math.floor(Date.now() / 1000)

  await canonicalTransactionChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp, 
      {
        gasLimit: GAS_LIMIT
      }
    )

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

  const timestamp = await appendTransactionBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

  const batchIndex = await canonicalTransactionChain.getBatchesLength()
  const cumulativePrevElements = await canonicalTransactionChain.cumulativeNumElements()

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
  batch: string[],
): Promise<StateChainBatch> => {
  const batchIndex = await stateCommitmentChain.getBatchesLength()
  const cumulativePrevElements = await stateCommitmentChain.cumulativeNumElements()

  await stateCommitmentChain.appendStateBatch(batch, {
    gasLimit: GAS_LIMIT
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
    "latest",
    false
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

  let ExecutionManager: ContractFactory
  let RollupMerkleUtils: ContractFactory
  let CanonicalTransactionChain: ContractFactory
  let StateCommitmentChain: ContractFactory
  let FraudVerifier: ContractFactory
  before(async () => {
    ExecutionManager = getContractFactory('ExecutionManager', wallet)
    RollupMerkleUtils = getContractFactory('RollupMerkleUtils', wallet)
    StateCommitmentChain = getContractFactory('StateCommitmentChain', wallet)
    CanonicalTransactionChain = getContractFactory('CanonicalTransactionChain', wallet)
    FraudVerifier = getContractFactory('FraudVerifier', wallet)
  })

  let executionManager: Contract
  let rollupMerkleUtils: Contract
  before(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      NULL_ADDRESS,
      GAS_LIMIT,
      true
    )
    rollupMerkleUtils = await RollupMerkleUtils.deploy()
  })

  let canonicalTransactionChain: Contract
  let stateCommitmentChain: Contract
  let fraudVerifier: Contract
  before(async () => {
    canonicalTransactionChain = await CanonicalTransactionChain.deploy(
      rollupMerkleUtils.address,
      sequencer.address,
      l1ToL2TransactionPasser.address,
      FORCE_INCLUSION_PERIOD
    )

    stateCommitmentChain = await StateCommitmentChain.deploy(
      rollupMerkleUtils.address,
      canonicalTransactionChain.address
    )

    fraudVerifier = await FraudVerifier.deploy(
      executionManager.address,
      stateCommitmentChain.address,
      canonicalTransactionChain.address,
      true, // Throw the verifier into testing mode.
      {
        gasLimit: Math.floor(GAS_LIMIT * 2)
      }
    )

    await stateCommitmentChain.setFraudVerifier(fraudVerifier.address)
  })
    
  let FraudTester: ContractFactory
  before(async () => {
    const fraudTesterDefinition = transpile(
      path.resolve(
        __dirname,
        './test-contracts/FraudTester.sol'
      ),
      executionManager.address
    ).FraudTester
    FraudTester = getContractFactoryFromDefinition(fraudTesterDefinition, wallet)
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
        keccak256('0x4567')
      ]),
      makeOvmTransaction(fraudTester, wallet, 'setStorage', [
        keccak256('0x0123'),
        keccak256('0x4567')
      ])
    ]
  })

  let transactionBatches: TxChainBatch[] = []
  let preStateBatches: StateChainBatch[] = []
  before(async () => {
    let preState = [await getCurrentStateRoot(provider)]
    for (const transaction of transactions) {
      await signAndSendOvmTransaction(wallet, transaction)

      const postState = [await getCurrentStateRoot(provider)]

      transactionBatches.push(await appendAndGenerateTransactionBatch(
        canonicalTransactionChain,
        sequencer,
        transactions
      ))

      preStateBatches.push(await appendAndGenerateStateBatch(
        stateCommitmentChain,
        preState
      ))

      preState = postState
    }
  })

  describe('prove', () => {
    let autoFraudProver: AutoFraudProver
    beforeEach(async () => {
      const stateTrieProof = await getStateTrieProof(fraudTester.address)

      autoFraudProver = new AutoFraudProver(
        0,
        preStateBatches[0].elements[0],
        await preStateBatches[0].getElementInclusionProof(0),
        preStateBatches[1].elements[0],
        await preStateBatches[1].getElementInclusionProof(0),
        transactions[0],
        await transactionBatches[0].getElementInclusionProof(0),
        [{
          
        }],
        wallet,
        fraudVerifier
      )
    })

    it('should handle the complete fraud proof process', async () => {

    })
  })
})
