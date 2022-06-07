import hre from 'hardhat'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'

import { expect } from '../../../../setup'
import { deploy } from '../../../../helpers'
import { encodeDripCheckParams } from '../../../../../src'

describe('CheckBalanceLow', () => {
  const RECIPIENT = '0x' + '11'.repeat(20)
  const THRESHOLD = 100

  let CheckBalanceLow: Contract
  before(async () => {
    CheckBalanceLow = await deploy('CheckBalanceLow')
  })

  describe('check', () => {
    it('should return true when balance is below threshold', async () => {
      await hre.ethers.provider.send('hardhat_setBalance', [
        RECIPIENT,
        toRpcHexString(THRESHOLD - 1),
      ])

      expect(
        await CheckBalanceLow.check(
          encodeDripCheckParams(CheckBalanceLow.interface, {
            target: RECIPIENT,
            threshold: THRESHOLD,
          })
        )
      ).to.equal(true)
    })

    it('should return false when balance is above threshold', async () => {
      await hre.ethers.provider.send('hardhat_setBalance', [
        RECIPIENT,
        toRpcHexString(THRESHOLD + 1),
      ])

      expect(
        await CheckBalanceLow.check(
          encodeDripCheckParams(CheckBalanceLow.interface, {
            target: RECIPIENT,
            threshold: THRESHOLD,
          })
        )
      ).to.equal(false)
    })
  })
})
