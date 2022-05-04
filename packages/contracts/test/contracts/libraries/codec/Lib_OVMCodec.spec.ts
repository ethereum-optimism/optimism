import { Contract } from 'ethers'

import { expect } from '../../../setup'
import { NON_ZERO_ADDRESS, deploy } from '../../../helpers'

describe('Lib_OVMCodec', () => {
  let Lib_OVMCodec: Contract
  before(async () => {
    Lib_OVMCodec = await deploy('TestLib_OVMCodec')
  })

  describe('hashTransaction', () => {
    enum QueueOrigin {
      SEQUENCER_QUEUE,
      L1TOL2_QUEUE,
    }

    it('should return the hash of a transaction', async () => {
      const tx = {
        timestamp: 121212,
        blockNumber: 10,
        l1QueueOrigin: QueueOrigin.SEQUENCER_QUEUE,
        l1TxOrigin: NON_ZERO_ADDRESS,
        entrypoint: NON_ZERO_ADDRESS,
        gasLimit: 100,
        data: '0x1234',
      }

      expect(await Lib_OVMCodec.hashTransaction(tx)).to.be.equal(
        '0xf07818e2db63d0140e55c9e68cfaa030f9a2d0962f671d6b339edb2207633ebd'
      )
    })
  })
})
