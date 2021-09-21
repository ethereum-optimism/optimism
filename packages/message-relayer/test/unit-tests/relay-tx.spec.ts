import { expect } from '../setup'

/* Imports: External */
import hre from 'hardhat'
import { Contract, Signer } from 'ethers'
import { getContractFactory } from '@eth-optimism/contracts'
import { smockit } from '@eth-optimism/smock'
import { toPlainObject } from 'lodash'

/* Imports: Internal */
import {
  getMerkleTreeProof,
  getMessagesAndProofsForL2Transaction,
  getStateRootBatchByTransactionIndex,
  getStateBatchAppendedEventByTransactionIndex,
  getMessagesByTransactionHash,
} from '../../src/relay-tx'

describe('relay transaction generation functions', () => {
  const ethers = (hre as any).ethers
  const l1RpcProvider = ethers.provider
  const l2RpcProvider = ethers.provider

  let signer1: Signer
  before(async () => {
    ;[signer1] = await ethers.getSigners()
  })

  let MockL2CrossDomainMessenger: Contract
  beforeEach(async () => {
    const factory = await ethers.getContractFactory(
      'MockL2CrossDomainMessenger'
    )
    MockL2CrossDomainMessenger = await factory.deploy()
  })

  let StateCommitmentChain: Contract
  beforeEach(async () => {
    const factory1 = getContractFactory('Lib_AddressManager')
    const factory2 = getContractFactory('OVM_ChainStorageContainer')
    const factory3 = getContractFactory('OVM_StateCommitmentChain')

    const mockBondManager = await smockit(getContractFactory('OVM_BondManager'))
    const mockCanonicalTransactionChain = await smockit(
      getContractFactory('OVM_CanonicalTransactionChain')
    )

    mockBondManager.smocked.isCollateralized.will.return.with(true)
    mockCanonicalTransactionChain.smocked.getTotalElements.will.return.with(
      999999
    )

    const AddressManager = await factory1.connect(signer1).deploy()
    const ChainStorageContainer = await factory2
      .connect(signer1)
      .deploy(AddressManager.address, 'OVM_StateCommitmentChain')
    StateCommitmentChain = await factory3
      .connect(signer1)
      .deploy(AddressManager.address, 0, 0)

    await AddressManager.setAddress(
      'OVM_ChainStorageContainer-SCC-batches',
      ChainStorageContainer.address
    )

    await AddressManager.setAddress(
      'OVM_StateCommitmentChain',
      StateCommitmentChain.address
    )

    await AddressManager.setAddress('OVM_BondManager', mockBondManager.address)

    await AddressManager.setAddress(
      'OVM_CanonicalTransactionChain',
      mockCanonicalTransactionChain.address
    )
  })

  describe('getMessageByTransactionHash', () => {
    it('should throw an error if a transaction with the given hash does not exist', async () => {
      await expect(
        getMessagesByTransactionHash(
          l2RpcProvider,
          MockL2CrossDomainMessenger.address,
          ethers.constants.HashZero
        )
      ).to.be.rejected
    })

    it('should return null if the transaction did not emit a SentMessage event', async () => {
      const tx = await MockL2CrossDomainMessenger.doNothing()

      expect(
        await getMessagesByTransactionHash(
          l2RpcProvider,
          MockL2CrossDomainMessenger.address,
          tx.hash
        )
      ).to.deep.equal([])
    })

    it('should return the parsed event if the transaction emitted exactly one SentMessage event', async () => {
      const message = {
        target: ethers.constants.AddressZero,
        sender: ethers.constants.AddressZero,
        message: '0x',
        messageNonce: 0,
      }
      const tx = await MockL2CrossDomainMessenger.emitSentMessageEvent(message)

      expect(
        await getMessagesByTransactionHash(
          l2RpcProvider,
          MockL2CrossDomainMessenger.address,
          tx.hash
        )
      ).to.deep.equal([message])
    })

    it('should return the parsed events if the transaction emitted more than one SentMessage event', async () => {
      const messages = [
        {
          target: ethers.constants.AddressZero,
          sender: ethers.constants.AddressZero,
          message: '0x',
          messageNonce: 0,
        },
        {
          target: ethers.constants.AddressZero,
          sender: ethers.constants.AddressZero,
          message: '0x',
          messageNonce: 1,
        },
      ]

      const tx = await MockL2CrossDomainMessenger.emitMultipleSentMessageEvents(
        messages
      )

      expect(
        await getMessagesByTransactionHash(
          l2RpcProvider,
          MockL2CrossDomainMessenger.address,
          tx.hash
        )
      ).to.deep.equal(messages)
    })
  })

  describe('getStateBatchAppendedEventByTransactionIndex', () => {
    it('should return null when there are no batches yet', async () => {
      expect(
        await getStateBatchAppendedEventByTransactionIndex(
          l1RpcProvider,
          StateCommitmentChain.address,
          0
        )
      ).to.equal(null)
    })

    it('should return null if a batch for the index does not exist', async () => {
      // Should have a total of 1 element now.
      await StateCommitmentChain.appendStateBatch(
        [ethers.constants.HashZero],
        0
      )

      expect(
        await getStateBatchAppendedEventByTransactionIndex(
          l1RpcProvider,
          StateCommitmentChain.address,
          1 // Index 0 is ok but 1 should return null
        )
      ).to.equal(null)
    })

    it('should return the batch if the index is part of the first batch', async () => {
      // 5 elements
      await StateCommitmentChain.appendStateBatch(
        [
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
        ],
        0
      )

      // Add another 5 so we have two batches and can isolate tests against the first.
      await StateCommitmentChain.appendStateBatch(
        [
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
        ],
        5
      )

      const event = await getStateBatchAppendedEventByTransactionIndex(
        l1RpcProvider,
        StateCommitmentChain.address,
        1
      )

      expect(toPlainObject(event.args)).to.deep.include({
        _batchIndex: ethers.BigNumber.from(0),
        _batchSize: ethers.BigNumber.from(5),
        _prevTotalElements: ethers.BigNumber.from(0),
      })
    })

    it('should return the batch if the index is part of the last batch', async () => {
      // 5 elements
      await StateCommitmentChain.appendStateBatch(
        [
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
        ],
        0
      )

      // Add another 5 so we have two batches and can isolate tests against the second.
      await StateCommitmentChain.appendStateBatch(
        [
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
          ethers.constants.HashZero,
        ],
        5
      )

      const event = await getStateBatchAppendedEventByTransactionIndex(
        l1RpcProvider,
        StateCommitmentChain.address,
        7
      )

      expect(toPlainObject(event.args)).to.deep.include({
        _batchIndex: ethers.BigNumber.from(1),
        _batchSize: ethers.BigNumber.from(5),
        _prevTotalElements: ethers.BigNumber.from(5),
      })
    })

    for (const numBatches of [1, 2, 8]) {
      const elementsPerBatch = 8
      describe(`when there are ${numBatches} batch(es) of ${elementsPerBatch} elements each`, () => {
        const totalElements = numBatches * elementsPerBatch
        beforeEach(async () => {
          for (let i = 0; i < numBatches; i++) {
            await StateCommitmentChain.appendStateBatch(
              new Array(elementsPerBatch).fill(ethers.constants.HashZero),
              i * elementsPerBatch
            )
          }
        })

        for (let i = 0; i < totalElements; i += elementsPerBatch) {
          it(`should be able to get the correct event for the ${i}th/st/rd/whatever element`, async () => {
            const event = await getStateBatchAppendedEventByTransactionIndex(
              l1RpcProvider,
              StateCommitmentChain.address,
              i
            )

            expect(toPlainObject(event.args)).to.deep.include({
              _batchIndex: ethers.BigNumber.from(i / elementsPerBatch),
              _batchSize: ethers.BigNumber.from(elementsPerBatch),
              _prevTotalElements: ethers.BigNumber.from(i),
            })
          })
        }
      })
    }
  })

  describe('getStateRootBatchByTransactionIndex', () => {
    it('should return null if a batch for the index does not exist', async () => {
      // Should have a total of 1 element now.
      await StateCommitmentChain.appendStateBatch(
        [ethers.constants.HashZero],
        0
      )

      expect(
        await getStateRootBatchByTransactionIndex(
          l1RpcProvider,
          StateCommitmentChain.address,
          1 // Index 0 is ok but 1 should return null
        )
      ).to.equal(null)
    })

    it('should return the full batch for a given index when it exists', async () => {
      // Should have a total of 1 element now.
      await StateCommitmentChain.appendStateBatch(
        [ethers.constants.HashZero],
        0
      )

      const batch = await getStateRootBatchByTransactionIndex(
        l1RpcProvider,
        StateCommitmentChain.address,
        0 // Index 0 is ok but 1 should return null
      )

      expect(batch.header).to.deep.include({
        batchIndex: ethers.BigNumber.from(0),
        batchSize: ethers.BigNumber.from(1),
        prevTotalElements: ethers.BigNumber.from(0),
      })

      expect(batch.stateRoots).to.deep.equal([ethers.constants.HashZero])
    })
  })

  describe('makeRelayTransactionData', () => {
    it('should throw an error if the transaction does not exist', async () => {
      await expect(
        getMessagesAndProofsForL2Transaction(
          l1RpcProvider,
          l2RpcProvider,
          StateCommitmentChain.address,
          MockL2CrossDomainMessenger.address,
          ethers.constants.HashZero
        )
      ).to.be.rejected
    })

    it('should throw an error if the transaction did not send a message', async () => {
      const tx = await MockL2CrossDomainMessenger.doNothing()

      await expect(
        getMessagesAndProofsForL2Transaction(
          l1RpcProvider,
          l2RpcProvider,
          StateCommitmentChain.address,
          MockL2CrossDomainMessenger.address,
          tx.hash
        )
      ).to.be.rejected
    })

    it('should throw an error if the corresponding state batch has not been submitted', async () => {
      const tx = await MockL2CrossDomainMessenger.emitSentMessageEvent({
        target: ethers.constants.AddressZero,
        sender: ethers.constants.AddressZero,
        message: '0x',
        messageNonce: 0,
      })

      await expect(
        getMessagesAndProofsForL2Transaction(
          l1RpcProvider,
          l2RpcProvider,
          StateCommitmentChain.address,
          MockL2CrossDomainMessenger.address,
          tx.hash
        )
      ).to.be.rejected
    })

    // Unfortunately this is hard to test here because hardhat doesn't support eth_getProof.
    // Because this function is embedded into the message relayer, we should be able to use
    // integration tests to sufficiently test this.
    it.skip('should otherwise return the encoded transaction data', () => {
      // TODO?
    })
  })
})

describe('getMerkleTreeProof', () => {
  let leaves: string[] = [
    'the',
    'quick',
    'brown',
    'fox',
    'jumps',
    'over',
    'the',
    'lazy',
    'dog',
  ]
  const index: number = 4
  it('should generate a merkle tree proof from an odd number of leaves at the correct index', () => {
    const expectedProof = [
      '0x6f766572',
      '0x123268ec1a3f9aac2bc68e899fe4329eefef783c76265722508b8abbfbf11440',
      '0x12aaa1b2e09f26e14d86aa3b157b94cfeabe815e44b6742d00c47441a576b12d',
      '0x297d90df3f77f93eefdeab4e9f6e9a074b41a3508f9d265e92e9b5449c7b11c8',
    ]
    expect(getMerkleTreeProof(leaves, index)).to.deep.equal(expectedProof)
  })

  it('should generate a merkle tree proof from an even number of leaves at the correct index', () => {
    const expectedProof = [
      '0x6f766572',
      '0x09e430fa7b513203dd9c74afd734267a73f64299d9dac61ef09e96c3b3b3fe96',
      '0x12aaa1b2e09f26e14d86aa3b157b94cfeabe815e44b6742d00c47441a576b12d',
    ]
    leaves = leaves.slice(0, leaves.length - 2)
    expect(getMerkleTreeProof(leaves, index)).to.deep.equal(expectedProof)
  })
})
