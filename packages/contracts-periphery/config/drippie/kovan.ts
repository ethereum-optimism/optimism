import { ethers } from 'ethers'

import { DrippieConfig } from '../../src'

const SECOND = 1
const MINUTE = 60 * SECOND
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const WEEK = 7 * DAY

const config: DrippieConfig = {
  TeleportrWithdrawal: {
    interval: 10 * MINUTE,
    dripcheck: 'CheckBalanceHigh',
    checkparams: {
      target: '0x4821975ca220601c153d02353300d6ad34adc362',
      threshold: ethers.utils.parseEther('1'),
    },
    actions: [
      {
        target: '0x78A25524D90E3D0596558fb43789bD800a5c3007',
        data: {
          fn: 'withdrawFromTeleportr',
          args: [],
        },
      },
    ],
  },
  GelatoBalance: {
    interval: 1 * DAY,
    dripcheck: 'CheckGelatoLow',
    checkparams: {
      treasury: '0x340759c8346A1E6Ed92035FB8B6ec57cE1D82c2c',
      recipient: '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5',
      threshold: ethers.utils.parseEther('0.1'),
    },
    actions: [
      {
        target: '0x340759c8346A1E6Ed92035FB8B6ec57cE1D82c2c',
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
    interval: 1 * WEEK,
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
