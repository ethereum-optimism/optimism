import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import {
  getProxyManager, ZERO_ADDRESS, NULL_BYTES32
} from '../../../helpers'

const parseQueueElement = (result: any[]): any => {
  return {
    timestamp: result[0].toNumber(),
    batchRoot: result[1],
    isL1ToL2Batch: result[2],
  }
}

const makeQueueElements = (count: number): any => {
  const elements = []
  for (let i = 0; i < count; i++) {
    elements.push({
      timestamp: Date.now(),
      batchRoot: NULL_BYTES32,
      isL1ToL2Batch: false,
    })
  }
  return elements
}

describe('OVM_L1ToL2TransactionQueue', () => {
  let signer: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Proxy_Manager: Contract
  before(async () => {
    Proxy_Manager = await getProxyManager()
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
      Proxy_Manager.address
    )
  })

  describe('enqueue()', () => {
    it('should allow users to enqueue an element', async () => {
      const [element] = makeQueueElements(1)
      await expect(OVM_L1ToL2TransactionQueue.enqueue(element)).to.not.be.reverted
    })

    it('should allow users to enqueue more than one element', async () => {
      const elements = makeQueueElements(10)
      for (let i = 0; i < elements.length; i++) {
        await expect(OVM_L1ToL2TransactionQueue.enqueue(elements[i])).to.not.be.reverted
      }
    })
  })

  describe('dequeue()', () => {
    describe('when the sender is not the OVM_CanonicalTransactionChain', () => {
      before(async () => {
        await Proxy_Manager.setProxy(
          'OVM_CanonicalTransactionChain',
          ZERO_ADDRESS
        )
      })

      it('should revert', async () => {
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.be.revertedWith('Sender is not allowed to enqueue.')
      })
    })
  
    describe('when the sender is the OVM_CanonicalTransactionChain', () => {
      before(async () => {
        await Proxy_Manager.setProxy(
          'OVM_CanonicalTransactionChain',
          await signer.getAddress()
        )
      })

      it('should revert if the queue is empty', async () => {
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.be.revertedWith('Queue is empty.')
      })

      it('should allow users to dequeue an element', async () => {
        const [element] = makeQueueElements(1)
        await OVM_L1ToL2TransactionQueue.enqueue(element)
        await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.not.be.reverted
      })

      it('should allow users to dequeue more than one element', async () => {
        const elements = makeQueueElements(10)
        for (let i = 0; i < elements.length; i++) {
          await OVM_L1ToL2TransactionQueue.enqueue(elements[i])
        }
        for (let i = 0; i < elements.length; i++) {
          await expect(OVM_L1ToL2TransactionQueue.dequeue()).to.not.be.reverted
        }
      })
    })
  })

  describe('size()', () => {
    before(async () => {
      await Proxy_Manager.setProxy(
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
        await OVM_L1ToL2TransactionQueue.enqueue(elements[i])
        const size = await OVM_L1ToL2TransactionQueue.size()
        expect(size).to.equal(i + 1)
      }
    })

    it('should decrease when elements are dequeued', async () => {
      const elements = makeQueueElements(10)
      for (let i = 0; i < elements.length; i++) {
        await OVM_L1ToL2TransactionQueue.enqueue(elements[i])
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
      await Proxy_Manager.setProxy(
        'OVM_CanonicalTransactionChain',
        await signer.getAddress()
      )
    })

    it('should revert when the queue is empty', async () => {
      await expect(OVM_L1ToL2TransactionQueue.peek()).to.be.revertedWith('Queue is empty.')
    })

    it('should return the front element if only one exists', async () => {
      const [element] = makeQueueElements(1)
      await OVM_L1ToL2TransactionQueue.enqueue(element)
      const front = await OVM_L1ToL2TransactionQueue.peek()
      expect(parseQueueElement(front)).to.deep.equal(element)
    })

    it('should return the front if more than one exists', async () => {
      const elements = makeQueueElements(10)
      for (let i = 0; i < elements.length; i++) {
        await OVM_L1ToL2TransactionQueue.enqueue(elements[i])
        const front = await OVM_L1ToL2TransactionQueue.peek()
        expect(parseQueueElement(front)).to.deep.equal(elements[0])
      }
    })

    it('should return the new front when elements are dequeued', async () => {
      const elements = makeQueueElements(10)
      for (let i = 0; i < elements.length; i++) {
        await OVM_L1ToL2TransactionQueue.enqueue(elements[i])
      }
      for (let i = 0; i < elements.length - 1; i++) {
        const front = await OVM_L1ToL2TransactionQueue.peek()
        expect(parseQueueElement(front)).to.deep.equal(elements[i + 1])
        await OVM_L1ToL2TransactionQueue.dequeue()
      }
    })
  })
})
