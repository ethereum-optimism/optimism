import { ethers } from 'ethers'

import { DrippieConfig, Time } from '../../src'

const config: DrippieConfig = {
  BatcherBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckBalanceLow',
    checkparams: {
      target: '0x6887246668a3b87f54deb3b94ba47a6f63f32985',
      threshold: ethers.utils.parseEther('75'),
    },
    actions: [
      {
        target: '0x6887246668a3b87f54deb3b94ba47a6f63f32985',
        value: ethers.utils.parseEther('125'),
      },
    ],
  },
  ProposerBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckBalanceLow',
    checkparams: {
      target: '0x473300df21d047806a082244b417f96b32f13a33',
      threshold: ethers.utils.parseEther('50'),
    },
    actions: [
      {
        target: '0x473300df21d047806a082244b417f96b32f13a33',
        value: ethers.utils.parseEther('100'),
      },
    ],
  },
  GelatoBalance: {
    interval: 1 * Time.DAY,
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
