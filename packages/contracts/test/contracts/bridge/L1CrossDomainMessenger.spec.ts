import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Internal Imports */
import {
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  makeAccountStorageProofTest,
  TxChainBatch,
  StateChainBatch,
  AccountStorageProofTest,
} from '../../test-helpers'
import { remove0x, numberToHexString } from '@eth-optimism/core-utils'

const getXDomainCalldata = (
  messenger: Contract,
  target: string,
  sender: string,
  message: string,
  nonce: number
): string => {
  return messenger.interface.encodeFunctionData('relayMessage', [
    target,
    sender,
    message,
    nonce,
  ])
}

const encodeL1ToL2Tx = (
  sender: string,
  target: string,
  gasLimit: number,
  data: string
): string => {
  return ethers.utils.defaultAbiCoder.encode(
    ['address', 'address', 'uint32', 'bytes'],
    [sender, target, gasLimit, data]
  )
}

const getMappingStorageSlot = (key: string, index: number): string => {
  const hexIndex = remove0x(
    ethers.BigNumber.from(index).toHexString()
  ).padStart(64, '0')
  return ethers.utils.keccak256(key + hexIndex)
}

const appendTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[]
): Promise<number[]> => {
  const blockNumber = await canonicalTransactionChain.provider.getBlockNumber()
  const timestamp = Math.floor(Date.now() / 1000)

  const startsAtIndex = await canonicalTransactionChain.cumulativeNumElements()
  await canonicalTransactionChain
    .connect(sequencer)
    .appendSequencerBatch(batch, timestamp, blockNumber, startsAtIndex)

  return [timestamp, blockNumber]
}

const appendAndGenerateTransactionBatch = async (
  canonicalTransactionChain: Contract,
  sequencer: Signer,
  batch: string[],
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<TxChainBatch> => {
  const [timestamp, blockNumber] = await appendTransactionBatch(
    canonicalTransactionChain,
    sequencer,
    batch
  )

  const localBatch = new TxChainBatch(
    timestamp,
    blockNumber,
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
  batchIndex: number = 0,
  cumulativePrevElements: number = 0
): Promise<StateChainBatch> => {
  const startsAtIndex = await stateCommitmentChain.cumulativeNumElements()
  await stateCommitmentChain.appendStateBatch(batch, startsAtIndex)

  const localBatch = new StateChainBatch(
    batchIndex,
    cumulativePrevElements,
    batch
  )

  await localBatch.generateTree()

  return localBatch
}

describe('L1CrossDomainMessenger', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let L1ToL2TransactionQueueFactory: ContractFactory
  let L1CrossDomainMessengerFactory: ContractFactory
  let L2CrossDomainMessengerFactory: ContractFactory
  let CrossDomainSimpleStorageFactory: ContractFactory
  let StateCommitmentChainFactory: ContractFactory
  let CanonicalTransactionChainFactory: ContractFactory
  before(async () => {
    L1ToL2TransactionQueueFactory = await ethers.getContractFactory(
      'L1ToL2TransactionQueue'
    )
    L1CrossDomainMessengerFactory = await ethers.getContractFactory(
      'L1CrossDomainMessenger'
    )
    L2CrossDomainMessengerFactory = await ethers.getContractFactory(
      'L2CrossDomainMessenger'
    )
    CrossDomainSimpleStorageFactory = await ethers.getContractFactory(
      'CrossDomainSimpleStorage'
    )
    StateCommitmentChainFactory = await ethers.getContractFactory(
      'StateCommitmentChain'
    )
    CanonicalTransactionChainFactory = await ethers.getContractFactory(
      'CanonicalTransactionChain'
    )
  })

  const L2_MESSAGE_PASSER_ADDRESS = '0x4200000000000000000000000000000000000000'
  const DUMMY_L2_MESSENGER_ADDRESS = '0x' + 'dead'.repeat(10)
  const WAITING_PERIOD = 60 * 60 * 24 * 7 //Waiting period is 1 week
  let L1ToL2TransactionQueue: Contract
  let L1CrossDomainMessenger: Contract
  let L2CrossDomainMessenger: Contract
  let CrossDomainSimpleStorage: Contract
  let StateCommitmentChain: Contract
  let CanonicalTransactionChain: Contract
  beforeEach(async () => {
    L1ToL2TransactionQueue = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'L1ToL2TransactionQueue',
      {
        factory: L1ToL2TransactionQueueFactory,
        params: [resolver.addressResolver.address],
      }
    )
    StateCommitmentChain = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'StateCommitmentChain',
      {
        factory: StateCommitmentChainFactory,
        params: [resolver.addressResolver.address],
      }
    )
    CanonicalTransactionChain = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'CanonicalTransactionChain',
      {
        factory: CanonicalTransactionChainFactory,
        params: [
          resolver.addressResolver.address,
          await wallet.getAddress(),
          100000,
        ],
      }
    )
    L1CrossDomainMessenger = await L1CrossDomainMessengerFactory.deploy(
      resolver.addressResolver.address,
      WAITING_PERIOD
    )
    L2CrossDomainMessenger = await L2CrossDomainMessengerFactory.deploy(
      L2_MESSAGE_PASSER_ADDRESS,
      L2_MESSAGE_PASSER_ADDRESS,
      WAITING_PERIOD
    )
    CrossDomainSimpleStorage = await CrossDomainSimpleStorageFactory.deploy()

    await L1CrossDomainMessenger.setTargetMessengerAddress(
      DUMMY_L2_MESSENGER_ADDRESS
    )
    await CrossDomainSimpleStorage.setMessenger(L1CrossDomainMessenger.address)
  })

  describe('relayMessage()', () => {
    const expectedKey = ethers.utils.keccak256('0x1234')
    const expectedVal = ethers.utils.keccak256('0x5678')
    let calldata: string
    let test: AccountStorageProofTest
    let xDomainCalldata: string
    beforeEach(async () => {
      calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )
      xDomainCalldata = getXDomainCalldata(
        L2CrossDomainMessenger,
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        0
      )
      const messageHash = ethers.utils.keccak256(xDomainCalldata)

      const storageKey = getMappingStorageSlot(messageHash, 0)
      const stateTrie = {
        [L2_MESSAGE_PASSER_ADDRESS]: {
          state: {
            nonce: 0,
            balance: 0,
            storageRoot: null,
            codeHash: null,
          },
          storage: [
            {
              key: storageKey,
              val: numberToHexString(1, 32),
            },
          ],
        },
      }

      test = await makeAccountStorageProofTest(
        stateTrie,
        L2_MESSAGE_PASSER_ADDRESS,
        storageKey
      )
    })

    it('should relay a message with a valid proof', async () => {
      await appendAndGenerateTransactionBatch(
        CanonicalTransactionChain,
        wallet,
        ['0x1234']
      )

      const stateBatch = await appendAndGenerateStateBatch(
        StateCommitmentChain,
        [test.stateTrieRoot]
      )

      await L1CrossDomainMessenger.relayMessage(
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        0,
        {
          stateRoot: test.stateTrieRoot,
          stateRootIndex: 0,
          stateRootProof: await stateBatch.getElementInclusionProof(0),
          stateTrieWitness: test.stateTrieWitness,
          storageTrieWitness: test.storageTrieWitness,
        }
      )

      const actualVal = await CrossDomainSimpleStorage.getStorage(expectedKey)
      expect(actualVal).to.equal(expectedVal)
    })

    it('should reject a message with an invalid state batch proof', async () => {
      await appendAndGenerateTransactionBatch(
        CanonicalTransactionChain,
        wallet,
        ['0x1234', '0x5678']
      )

      const stateBatch = await appendAndGenerateStateBatch(
        StateCommitmentChain,
        [test.stateTrieRoot, '0x' + '11'.repeat(32)]
      )

      await expect(
        L1CrossDomainMessenger.relayMessage(
          CrossDomainSimpleStorage.address,
          await wallet.getAddress(),
          calldata,
          0,
          {
            stateRoot: test.stateTrieRoot,
            stateRootIndex: 0,
            stateRootProof: await stateBatch.getElementInclusionProof(1), // Wrong index
            stateTrieWitness: test.stateTrieWitness,
            storageTrieWitness: test.storageTrieWitness,
          }
        )
      ).to.be.rejectedWith('Provided message could not be verified.')
    })

    it('should reject a message with an invalid account trie proof', async () => {
      const messageHash = ethers.utils.keccak256(xDomainCalldata)

      const storageKey = getMappingStorageSlot(messageHash, 0)
      const stateTrie = {
        [L2_MESSAGE_PASSER_ADDRESS]: {
          state: {
            nonce: 0,
            balance: 0,
            storageRoot: null,
            codeHash: null,
          },
          storage: [
            {
              key: storageKey,
              val: numberToHexString(0, 32), // Zero means false
            },
          ],
        },
      }

      test = await makeAccountStorageProofTest(
        stateTrie,
        L2_MESSAGE_PASSER_ADDRESS,
        storageKey
      )

      await appendAndGenerateTransactionBatch(
        CanonicalTransactionChain,
        wallet,
        ['0x1234']
      )

      const stateBatch = await appendAndGenerateStateBatch(
        StateCommitmentChain,
        [test.stateTrieRoot]
      )

      await expect(
        L1CrossDomainMessenger.relayMessage(
          CrossDomainSimpleStorage.address,
          await wallet.getAddress(),
          calldata,
          0,
          {
            stateRoot: test.stateTrieRoot,
            stateRootIndex: 0,
            stateRootProof: await stateBatch.getElementInclusionProof(0),
            stateTrieWitness: test.stateTrieWitness,
            storageTrieWitness: test.storageTrieWitness,
          }
        )
      ).to.be.rejectedWith('Provided message could not be verified.')
    })
  })

  describe('sendMessage()', () => {
    it('should add the correct message to the L1ToL2TransactionQueue', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      const gasLimit = 1000000

      const messageNonce = await L1CrossDomainMessenger.messageNonce()

      await L1CrossDomainMessenger.sendMessage(
        CrossDomainSimpleStorage.address,
        calldata,
        gasLimit
      )

      const xDomainCalldata = getXDomainCalldata(
        L2CrossDomainMessenger,
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        messageNonce
      )

      const l1ToL2Tx = encodeL1ToL2Tx(
        L1CrossDomainMessenger.address,
        DUMMY_L2_MESSENGER_ADDRESS,
        gasLimit,
        xDomainCalldata
      )

      const txHash = ethers.utils.keccak256(l1ToL2Tx)
      const batchHeader = await L1ToL2TransactionQueue.peek()
      expect(batchHeader[0]).to.equal(txHash)

      const newMessageNonce = await L1CrossDomainMessenger.messageNonce()
      expect(newMessageNonce.toNumber()).to.equal(1)

      const messageHash = ethers.utils.keccak256(xDomainCalldata)
      const messageSent = await L1CrossDomainMessenger.sentMessages(messageHash)
      expect(messageSent).to.equal(true)
    })
  })

  describe('replayMessage()', () => {
    it('should replay a previously sent message', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      const gasLimit = 1000000

      const messageNonce = await L1CrossDomainMessenger.messageNonce()

      await L1CrossDomainMessenger.sendMessage(
        CrossDomainSimpleStorage.address,
        calldata,
        gasLimit
      )

      const xDomainCalldata = getXDomainCalldata(
        L2CrossDomainMessenger,
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        messageNonce
      )

      const l1ToL2Tx = encodeL1ToL2Tx(
        L1CrossDomainMessenger.address,
        DUMMY_L2_MESSENGER_ADDRESS,
        gasLimit,
        xDomainCalldata
      )

      const txHash = ethers.utils.keccak256(l1ToL2Tx)
      const batchHeader = await L1ToL2TransactionQueue.peek()
      expect(batchHeader[0]).to.equal(txHash)

      const batchHeadersLength = await L1ToL2TransactionQueue.getBatchHeadersLength()
      expect(batchHeadersLength.toNumber()).to.equal(1)

      await L1CrossDomainMessenger.replayMessage(
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        messageNonce,
        gasLimit
      )

      const newBatchHeader = await L1ToL2TransactionQueue.peek()
      expect(newBatchHeader[0]).to.equal(txHash)

      const newBatchHeadersLength = await L1ToL2TransactionQueue.getBatchHeadersLength()
      expect(newBatchHeadersLength.toNumber()).to.equal(2)
    })

    it('should fail if attempting to replay a message not previously sent', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      const gasLimit = 1000000

      const messageNonce = await L1CrossDomainMessenger.messageNonce()

      await expect(
        L1CrossDomainMessenger.replayMessage(
          CrossDomainSimpleStorage.address,
          await wallet.getAddress(),
          calldata,
          messageNonce,
          gasLimit
        )
      ).to.be.revertedWith('Provided message has not already been sent.')
    })
  })
})
