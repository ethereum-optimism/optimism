import { ethers } from 'ethers'

import { DrippieConfig, Time } from '../../src'

const config: DrippieConfig = {
  GelatoBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckGelatoLow',
    checkparams: {
      treasury: '0x527a819db1eb0e34426297b03bae11F2f8B3A19E',
      recipient: '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5',
      threshold: ethers.utils.parseEther('0.1'),
    },
    actions: [
      {
        target: '0x527a819db1eb0e34426297b03bae11F2f8B3A19E',
        value: ethers.utils.parseEther('1'),
        data: {
          fn: 'depositFunds',
          args: [
            // receiver
            '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5',
            // token
            '0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee',
            // amount
            ethers.utils.parseEther('1'),
          ],
        },
      },
    ],
  },
  TonyOptimismKovanFaucet: {
    interval: 1 * Time.WEEK,
    dripcheck: 'CheckBalanceLow',
    checkparams: {
      target: '0xa8019d6F7bC3008a0a708A422f223Ccb21b61eAD',
      threshold: ethers.utils.parseEther('20'),
    },
    actions: [
      {
        target: '0xa8019d6F7bC3008a0a708A422f223Ccb21b61eAD',
        value: ethers.utils.parseEther('100'),
      },
    ],
  },
}

export default config
