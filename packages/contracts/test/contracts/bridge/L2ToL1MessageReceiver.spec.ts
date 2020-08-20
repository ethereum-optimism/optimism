import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'

/* Logging */
const log = getLogger('l2-to-l2-msg-receiver', true)

/* Tests */
describe.only('L2ToL1MessageReceiver', () => {
  const provider = ethers.provider
  let wallet: Signer
  let L2ToL1MessageReceiver: ContractFactory

  before(async () => {
    ;[wallet] = await ethers.getSigners()
    L2ToL1MessageReceiver = await ethers.getContractFactory(
      'L2ToL1MessageReceiver'
    )
  })

  let l2ToL1MessageReceiver: Contract
  beforeEach(async () => {
    l2ToL1MessageReceiver = await L2ToL1MessageReceiver.deploy()
  })

  describe('verifyMessage', async () => {
    it('verifies the message', async () => {
    })
  })
})
