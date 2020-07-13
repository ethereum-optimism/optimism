import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger } from '@eth-optimism/core-utils'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  CHAIN_ID,
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT,
  signTransaction,
  getSignedComponents,
  getWallets,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-recover-eoa-address', true)

export const abi = new ethers.utils.AbiCoder()

/* Tests */
describe('Execution Manager -- Recover EOA Address', () => {
  const [wallet] = getWallets()

  let ExecutionManager: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
    )
  })

  describe('recoverEOAAddress', async () => {
    it('correctly recovers EOA addresses which are sent to contracts', async () => {
      // Generate a dummy tx to sign
      const eoaTx = {
        nonce: 1,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: '0x' + '91'.repeat(20),
        value: 0,
        data: '0xdeadbeef',
        chainId: CHAIN_ID,
      }
      // Sign the message
      const signedMessage = await signTransaction(wallet, eoaTx)
      // Extract signature
      const [v, r, s] = getSignedComponents(signedMessage)
      // Call the executionManager's recover address function
      const recoveredAddress = await executionManager.recoverEOAAddress(
        eoaTx.nonce,
        eoaTx.to,
        eoaTx.data,
        v,
        r,
        s
      )
      // Check that the recovered address matches the wallet address
      recoveredAddress.should.equal(await wallet.getAddress())
      // Done!
    })

    // TODO: Handle contract creation in a less error-prone way
    it('correctly recovers EOA addresses which create contracts', async () => {
      // Generate a dummy tx to sign
      const eoaTx = {
        nonce: 1,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        value: 0,
        data: '0xdeadbeef',
        chainId: CHAIN_ID,
      }
      // Sign the message
      const signedMessage = await signTransaction(wallet, eoaTx)
      // Extract signature
      const [v, r, s] = getSignedComponents(signedMessage)
      // Call the executionManager's recover address function
      const recoveredAddress = await executionManager.recoverEOAAddress(
        eoaTx.nonce,
        '0x' + '00'.repeat(20), // Replace the TO so that it makes a CREATE tx
        eoaTx.data,
        v,
        r,
        s
      )
      // Check that the recovered address matches the wallet address
      recoveredAddress.should.equal(await wallet.getAddress())
      // Done!
    })
  })
})
