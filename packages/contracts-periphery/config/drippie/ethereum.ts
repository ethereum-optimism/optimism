import { ethers } from 'ethers'

import { DrippieConfig } from '../../src'

const config: DrippieConfig = {
  TeleportrWithdrawalV2: {
    interval: 60 * 60 * 24,
    dripcheck: 'CheckBalanceHigh',
    checkparams: {
      target: '0x52ec2f3d7c5977a8e558c8d9c6000b615098e8fc',
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
    interval: 60 * 60 * 24,
    dripcheck: 'CheckGelatoLow',
    checkparams: {
      treasury: '0x2807B4aE232b624023f87d0e237A3B1bf200Fd99',
      recipient: '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5',
      threshold: ethers.utils.parseEther('0.1'),
    },
    actions: [
      {
        target: '0x2807B4aE232b624023f87d0e237A3B1bf200Fd99',
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
}

export default config
