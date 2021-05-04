import { expect } from '../setup'

/* Imports: External */
import hre from 'hardhat'

/* Imports: Internal */
import { makeActionBundleFromConfig } from '../../src'

describe('ChugSplash hardhat tooling', () => {
  describe('makeActionBundleFromConfig', () => {
    it('should generate an action bundle from a basic config file', async () => {
      // TODO: What's the best way to test this?
      await makeActionBundleFromConfig(hre, {
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'OVM_ExecutionManager',
            variables: {},
          },
        },
      })
    })
  })
})
