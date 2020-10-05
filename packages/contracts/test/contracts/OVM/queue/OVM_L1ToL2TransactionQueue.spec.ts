import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer, BigNumber } from 'ethers'
import { keccak256 } from 'ethers/lib/utils'
import _ from 'lodash'

/* Internal Imports */
import {
  ZERO_ADDRESS,
  NON_ZERO_ADDRESS,
  DUMMY_BYTES32,
  remove0x,
  getBlockTime,
  makeAddressManager,
} from '../../../helpers'

const calcBatchRoot = (element: any): string => {
  return keccak256(
    element.target +
      remove0x(BigNumber.from(element.gasLimit).toHexString()).padStart(
        64,
        '0'
      ) +
      remove0x(element.data)
  )
}

const makeQueueElements = (count: number): any => {
  return [...Array(count)].map((el, idx) => {
    return {
      target: NON_ZERO_ADDRESS,
      gasLimit: idx + 30_000,
      data: DUMMY_BYTES32[0],
    }
  })
}

describe('OVM_L1ToL2TransactionQueue', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Factory__OVM_L1ToL2TransactionQueue: ContractFactory
  before(async () => {
    Factory__OVM_L1ToL2TransactionQueue = await ethers.getContractFactory(
      'OVM_L1ToL2TransactionQueue'
    )
  })

  let OVM_L1ToL2TransactionQueue: Contract
  beforeEach(async () => {
    OVM_L1ToL2TransactionQueue = await Factory__OVM_L1ToL2TransactionQueue.deploy(
      AddressManager.address
    )
  })

  describe('enqueue()', () => {
    it('should allow users to enqueue an element', async () => {
      const [element] = makeQueueElements(1)
      await expect(
        OVM_L1ToL2TransactionQueue.enqueue(
          element.target,
          element.gasLimit,
          element.data
        )
      ).to.not.be.reverted
    })

    it('should allow users to enqueue more than one element', async () => {
      for (const element of makeQueueElements(10)) {
        await expect(
          OVM_L1ToL2TransactionQueue.enqueue(
            element.target,
            element.gasLimit,
            element.data
          )
        ).to.not.be.reverted
      }
    })
  })

  describe('dequeue()', () => {
    describe('when the sender is not the OVM_CanonicalTransactionChain', () => {
      before(async () => {
        await AddressManager.setAddress(
          'OVM_CanonicalTransactionChain',
          ZERO_ADDRESS
        )
      })

      it('should revert', async () => {
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.be.revertedWith(
          'Sender is not allowed to enqueue.'
        )
      })
    })

    describe('when the sender is the OVM_CanonicalTransactionChain', () => {
      before(async () => {
        await AddressManager.setAddress(
          'OVM_CanonicalTransactionChain',
          await signer.getAddress()
        )
      })

      it('should revert if the queue is empty', async () => {
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.be.revertedWith(
          'Queue is empty.'
        )
      })

      it('should allow users to dequeue an element', async () => {
        const [element] = makeQueueElements(1)
        await OVM_L1ToL2TransactionQueue.enqueue(
          element.target,
          element.gasLimit,
          element.data
        )
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.not.be.reverted
      })

      it('should allow users to dequeue more than one element', async () => {
        const elements = makeQueueElements(10)

        for (const element of elements) {
          await OVM_L1ToL2TransactionQueue.enqueue(
            element.target,
            element.gasLimit,
            element.data
          )
        }

        for (const element of elements) {
          await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.not.be.reverted
        }
      })
    })
  })

  describe('size()', () => {
    before(async () => {
      await AddressManager.setAddress(
        'OVM_CanonicalTransactionChain',
        await signer.getAddress()
      )
    })

    it('should return zero when no elements are in the queue', async () => {
      const size = await OVM_L1ToL2TransactionQueue.size()
      expect(size).to.equal(0)
    })

    it('should increase when new elements are enqueued', async () => {
      const elements = makeQueueElements(10)
      for (let i = 0; i < elements.length; i++) {
        const element = elements[i]
        await OVM_L1ToL2TransactionQueue.enqueue(
          element.target,
          element.gasLimit,
          element.data
        )

        const size = await OVM_L1ToL2TransactionQueue.size()
        expect(size).to.equal(i + 1)
      }
    })

    it('should decrease when elements are dequeued', async () => {
      const elements = makeQueueElements(10)

      for (const element of elements) {
        await OVM_L1ToL2TransactionQueue.enqueue(
          element.target,
          element.gasLimit,
          element.data
        )
      }

      for (let i = 0; i < elements.length; i++) {
        await OVM_L1ToL2TransactionQueue.dequeue()
        const size = await OVM_L1ToL2TransactionQueue.size()
        expect(size).to.equal(elements.length - i - 1)
      }
    })
  })

  describe('peek()', () => {
    before(async () => {
      await AddressManager.setAddress(
        'OVM_CanonicalTransactionChain',
        await signer.getAddress()
      )
    })

    it('should revert when the queue is empty', async () => {
      await expect(OVM_L1ToL2TransactionQueue.peek()).to.be.revertedWith(
        'Queue is empty.'
      )
    })

    it('should return the front element if only one exists', async () => {
      const [element] = makeQueueElements(1)

      const result = await OVM_L1ToL2TransactionQueue.enqueue(
        element.target,
        element.gasLimit,
        element.data
      )

      const timestamp = BigNumber.from(
        await getBlockTime(ethers.provider, result.blockNumber)
      )

      expect(
        _.toPlainObject(await OVM_L1ToL2TransactionQueue.peek())
      ).to.deep.include({
        timestamp,
        batchRoot: calcBatchRoot(element),
        isL1ToL2Batch: true,
      })
    })

    it('should return the front if more than one exists', async () => {
      const elements = makeQueueElements(10)
      const result = await OVM_L1ToL2TransactionQueue.enqueue(
        elements[0].target,
        elements[0].gasLimit,
        elements[0].data
      )
      const timestamp = BigNumber.from(
        await getBlockTime(ethers.provider, result.blockNumber)
      )

      for (const element of elements.slice(1)) {
        expect(
          _.toPlainObject(await OVM_L1ToL2TransactionQueue.peek())
        ).to.deep.include({
          timestamp,
          batchRoot: calcBatchRoot(elements[0]),
          isL1ToL2Batch: true,
        })
      }
    })

    it('should return the new front when elements are dequeued', async () => {
      const elements = makeQueueElements(10)
      const timestamps: BigNumber[] = []

      for (const element of elements) {
        const result = await OVM_L1ToL2TransactionQueue.enqueue(
          element.target,
          element.gasLimit,
          element.data
        )

        timestamps.push(
          BigNumber.from(
            await getBlockTime(ethers.provider, result.blockNumber)
          )
        )
      }

      for (let i = 0; i < elements.length - 1; i++) {
        expect(
          _.toPlainObject(await OVM_L1ToL2TransactionQueue.peek())
        ).to.deep.include({
          timestamp: timestamps[i],
          batchRoot: calcBatchRoot(elements[i]),
          isL1ToL2Batch: true,
        })

        await OVM_L1ToL2TransactionQueue.dequeue()
      }
    })
  })
})
