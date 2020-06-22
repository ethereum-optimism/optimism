import '../setup'

/* External Imports */
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { Contract } from 'ethers'

/* Logging */
const log = getLogger('safe-contract-registry', true)

/* Contract Imports */
import * as SafeContractRegistry from '../../build/SafeContractRegistry.json'
import * as StubSafetyChecker from '../../build/StubSafetyChecker.json'

/* Begin tests */
describe('SafeContractRegistry', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let stubSafetyChecker
  let safeContractRegistry

  /* Deploy contracts before tests */
  beforeEach(async () => {
    stubSafetyChecker = await deployContract(wallet, StubSafetyChecker, [])

    safeContractRegistry = await deployContract(wallet, SafeContractRegistry, [
      stubSafetyChecker.address
    ])
  })

  describe('registerNewContract', async () => {
    it('does not fail when deploying some simple bytecode', async () => {
      const validBytecode = '0x6080604052348015600f57600080fd5b50608b8061001e6000396000f3fe6080604052348015600f57600080fd5b506004361060285760003560e01c8063f613a68714602d575b600080fd5b6033604d565b604051808215151515815260200191505060405180910390f35b6000600190509056fea265627a7a723158202a23346783a321b4309e7d0f14d0e46dd04f98b328bf388b3fe8b24fbe7ef37f64736f6c63430005110032'
      await safeContractRegistry.registerNewContract(validBytecode)
      // success it worked!
    })

    // TODO: Add some real tests
  })
})
