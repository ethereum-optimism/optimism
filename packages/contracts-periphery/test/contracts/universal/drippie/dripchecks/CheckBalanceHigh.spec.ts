import hre from 'hardhat'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'

import { expect } from '../../../../setup'
import { deploy } from '../../../../helpers'
import { encodeDripCheckParams } from '../../../../../src'

describe('CheckBalanceHigh', () => {
  const RECIPIENT = '0x' + '11'.repeat(20)
  const THRESHOLD = 100

  let CheckBalanceHigh: Contract
  before(async () => {
    CheckBalanceHigh = await deploy('CheckBalanceHigh')
  })

  describe('check', () => {
    it('should return true when balance is above threshold', async () => {
      await hre.ethers.provider.send('hardhat_setBalance', [
        RECIPIENT,
        toRpcHexString(THRESHOLD + 1),
      ])

      expect(
        await CheckBalanceHigh.check(
          encodeDripCheckParams(CheckBalanceHigh.interface, {
            target: RECIPIENT,
            threshold: THRESHOLD,
          })
        )
      ).to.equal(true)
    })

    it('should return false when balance is below threshold', async () => {
      await hre.ethers.provider.send('hardhat_setBalance', [
        RECIPIENT,
        toRpcHexString(THRESHOLD - 1),
      ])

      expect(
        await CheckBalanceHigh.check(
          encodeDripCheckParams(CheckBalanceHigh.interface, {
            target: RECIPIENT,
            threshold: THRESHOLD,
          })
        )
      ).to.equal(false)
    })
  })
})
