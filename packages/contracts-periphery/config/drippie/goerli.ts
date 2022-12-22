import { ethers } from 'ethers'

import { DrippieConfig, Time } from '../../src'

const config: DrippieConfig = {
  BatcherBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckBalanceLow',
    checkparams: {
      target: '0x7431310e026b69bfc676c0013e12a1a11411eec9',
      threshold: ethers.utils.parseEther('50'),
    },
    actions: [
      {
        target: '0x7431310e026b69bfc676c0013e12a1a11411eec9',
        value: ethers.utils.parseEther('100'),
      },
    ],
  },
  ProposerBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckBalanceLow',
    checkparams: {
      target: '0x02b1786a85ec3f71fbbba46507780db7cf9014f6',
      threshold: ethers.utils.parseEther('50'),
    },
    actions: [
      {
        target: '0x02b1786a85ec3f71fbbba46507780db7cf9014f6',
        value: ethers.utils.parseEther('100'),
      },
    ],
  },
  GelatoBalance: {
    interval: 1 * Time.DAY,
    dripcheck: 'CheckGelatoLow',
    checkparams: {
      treasury: '0xf381dfd7a139caab83c26140e5595c0b85ddadcd',
      recipient: '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5',
      threshold: ethers.utils.parseEther('0.1'),
    },
    actions: [
      {
        target: '0xf381dfd7a139caab83c26140e5595c0b85ddadcd',
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
