/* eslint-disable @typescript-eslint/no-empty-function */
import { expect, testEnv } from '../setup'
import { AccountType, StateDump } from '../../scripts/types'
import { classifiers } from '../../scripts/classifiers'

const eoaClassifier = classifiers[AccountType.EOA]

describe('EOAs', () => {
  describe('standard EOA', () => {
    let eoas: StateDump
    before(async () => {
      await testEnv.init()
      eoas = testEnv.surgeryDataSources.dump.filter(a => eoaClassifier(a, testEnv.surgeryDataSources))
    })

    // Iterate through all of the EOAs and check that they have no code
    // in the new node
    it('should not have any code', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % 10000 === 0) {
          console.log(`Checking code for account ${i}`)
        }
        const code = await testEnv.postL2Provider.getCode(eoa.address)
        expect(code).to.eq('0x')
      }
    })

    it.skip('should have the null code hash', async () => {})
    it.skip('should have the null storage root', async () => {})

    it('should have the same balance as it had before', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % 10000 === 0) {
          console.log(`Checking balance for account ${i}`)
        }
        const preBalance = await testEnv.preL2Provider.getBalance(eoa.address, testEnv.config.stateDumpHeight)
        const postBalance = await testEnv.postL2Provider.getBalance(eoa.address)
        try {
          expect(preBalance).to.deep.eq(postBalance)
        } catch (e) {
          console.log(`Balance mismatch ${eoa.address}`)
          console.log(e)
        }
      }
    })

    it('should have the same nonce as it had before', async () => {
      for (const [i, eoa] of eoas.entries()) {
        if (i % 10000 === 0) {
          console.log(`Checking nonce for account ${i}`)
        }
        const preNonce = await testEnv.preL2Provider.getTransactionCount(eoa.address, testEnv.config.stateDumpHeight)
        const postNonce = await testEnv.postL2Provider.getTransactionCount(eoa.address)
        try {
          expect(preNonce).to.deep.eq(postNonce)
        } catch (e) {
          console.log(`Nonce mismatch ${eoa.address}`)
          console.log(e)
        }
      }
    })
  })

  describe('1inch deployer', () => {
    it('should not have any code', async () => {})

    it('should have the null code hash', async () => {})

    it('should have the null storage root', async () => {})

    it('should have the same balance as it had before', async () => {})

    it('should have a nonce equal to zero', async () => {})
  })
})
