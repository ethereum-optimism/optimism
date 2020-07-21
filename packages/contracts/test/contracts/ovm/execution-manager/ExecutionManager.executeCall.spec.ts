import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  padToLength,
  ZERO_ADDRESS,
  TestUtils,
  getCurrentTime,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  CHAIN_ID,
  DEFAULT_OPCODE_WHITELIST_MASK,
  ZERO_UINT,
  Address,
  manuallyDeployOvmContract,
  signTransaction,
  getSignedComponents,
  getWallets,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-calls', true)

export const abi = new ethers.utils.AbiCoder()

/* Tests */
describe('Execution Manager -- Call opcodes', () => {
  const provider = ethers.provider

  const [wallet] = getWallets()

  let ExecutionManager: ContractFactory
  let StateManager: ContractFactory
  let DummyContract: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    StateManager = await ethers.getContractFactory('StateManager')
    DummyContract = await ethers.getContractFactory('DummyContract')
  })

  let executionManager: Contract
  let stateManager: Contract
  let dummyContractAddress: Address
  beforeEach(async () => {
    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
    )

    stateManager = StateManager.attach(
      await executionManager.getStateManagerAddress()
    )

    dummyContractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      DummyContract,
      []
    )

    log.debug(`Contract address: [${dummyContractAddress}]`)
  })

  describe('executeNonEOACall', async () => {
    it('fails if the provided timestamp is 0', async () => {
      // Create the variables we will use for setStorage
      const intParam = 0
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = DummyContract.interface.encodeFunctionData(
        'dummyFunction',
        [intParam, bytesParam]
      )
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: dummyContractAddress,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }
      const signedMessage = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedMessage)

      await TestUtils.assertRevertsAsync(
        'Timestamp must be greater than 0',
        async () => {
          // Call using Ethers
          const tx = await executionManager.executeEOACall(
            0,
            0,
            transaction.nonce,
            transaction.to,
            transaction.data,
            padToLength(v, 4),
            padToLength(r, 64),
            padToLength(s, 64)
          )
          await provider.waitForTransaction(tx.hash)
        }
      )
    })

    it('properly executes a raw call -- 0 param', async () => {
      // Create the variables we will use for setStorage
      const intParam = 0
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = DummyContract.interface.encodeFunctionData(
        'dummyFunction',
        [intParam, bytesParam]
      )
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: dummyContractAddress,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }

      // Call using Ethers
      const tx = await executionManager.executeTransaction(
        getCurrentTime(),
        0,
        transaction.to,
        transaction.data,
        wallet.address,
        ZERO_ADDRESS,
        true
      )
      await provider.waitForTransaction(tx.hash)
    })
  })

  describe('executeEOACall', async () => {
    it('properly executes a raw call -- 0 param', async () => {
      // Create the variables we will use for setStorage
      const intParam = 0
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = DummyContract.interface.encodeFunctionData(
        'dummyFunction',
        [intParam, bytesParam]
      )
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: dummyContractAddress,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }
      const signedMessage = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedMessage)

      // Call using Ethers
      const tx = await executionManager.executeEOACall(
        getCurrentTime(),
        0,
        transaction.nonce,
        transaction.to,
        transaction.data,
        padToLength(v, 4),
        padToLength(r, 64),
        padToLength(s, 64)
      )
      await provider.waitForTransaction(tx.hash)
    })

    it('increments the senders nonce', async () => {
      // Create the variables we will use for setStorage
      const intParam = 0
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = DummyContract.interface.encodeFunctionData(
        'dummyFunction',
        [intParam, bytesParam]
      )
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: dummyContractAddress,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }
      const signedMessage = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedMessage)

      // Call using Ethers
      const tx = await executionManager.executeEOACall(
        getCurrentTime(),
        0,
        transaction.nonce,
        transaction.to,
        transaction.data,
        v,
        r,
        s
      )
      await provider.waitForTransaction(tx.hash)
      const nonceAfter = await stateManager.getOvmContractNonceView(
        wallet.address
      )
      nonceAfter.should.equal(parseInt(nonce, 10) + 1)
    })

    it('properly executes a raw call -- 1 param', async () => {
      const intParam = 1
      const bytesParam = '0xdeadbeef'
      // Generate our tx calldata
      const calldata = DummyContract.interface.encodeFunctionData(
        'dummyFunction',
        [intParam, bytesParam]
      )
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: dummyContractAddress,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }
      const signedMessage = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedMessage)

      // Call using Ethers
      const tx = await executionManager.executeEOACall(
        getCurrentTime(),
        0,
        transaction.nonce,
        transaction.to,
        transaction.data,
        padToLength(v, 4),
        padToLength(r, 64),
        padToLength(s, 64)
      )
      await provider.waitForTransaction(tx.hash)
    })

    it('reverts when it makes a call that reverts', async () => {
      // Generate our tx internalCalldata
      const internalCalldata = DummyContract.interface.encodeFunctionData(
        'dummyRevert',
        []
      )

      const calldata = ExecutionManager.interface.encodeFunctionData(
        'executeTransaction',
        [
          ZERO_UINT,
          ZERO_UINT,
          dummyContractAddress,
          internalCalldata,
          wallet.address,
          ZERO_ADDRESS,
          true,
        ]
      )
      const nonce = await provider.getTransactionCount(wallet.address)

      let failed = false
      try {
        await wallet.provider.call({
          nonce,
          gasLimit: GAS_LIMIT,
          gasPrice: 0,
          to: executionManager.address,
          value: 0,
          data: calldata,
          chainId: CHAIN_ID,
        })
      } catch (e) {
        log.debug(JSON.stringify(e) + '  ' + e.stack)
        failed = true
      }

      failed.should.equal(true, `This call should have reverted!`)
    })

    it('reverts when it makes a call that fails a require', async () => {
      // Generate our tx internalCalldata
      const internalCalldata = DummyContract.interface.encodeFunctionData(
        'dummyFailingRequire',
        []
      )

      const calldata = ExecutionManager.interface.encodeFunctionData(
        'executeTransaction',
        [
          ZERO_UINT,
          ZERO_UINT,
          dummyContractAddress,
          internalCalldata,
          wallet.address,
          ZERO_ADDRESS,
          true,
        ]
      )
      const nonce = await provider.getTransactionCount(wallet.address)

      let failed = false
      try {
        await wallet.provider.call({
          nonce,
          gasLimit: GAS_LIMIT,
          gasPrice: 0,
          to: executionManager.address,
          value: 0,
          data: calldata,
          chainId: CHAIN_ID,
        })
      } catch (e) {
        log.debug(JSON.stringify(e) + '  ' + e.stack)
        failed = true
      }

      failed.should.equal(true, `This call should have reverted!`)
    })
  })
})
