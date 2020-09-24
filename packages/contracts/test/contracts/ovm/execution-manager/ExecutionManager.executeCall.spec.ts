import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  padToLength,
  ZERO_ADDRESS,
  TestUtils,
  getCurrentTime,
  NULL_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Wallet } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  CHAIN_ID,
  ZERO_UINT,
  Address,
  manuallyDeployOvmContract,
  signTransaction,
  getSignedComponents,
  getWallets,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  getDefaultGasMeterParams,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-calls', true)

export const abi = new ethers.utils.AbiCoder()
const DEPLOYER_WHITELIST_OVM_ADDRESS =
  '0x4200000000000000000000000000000000000002'

/* Tests */
describe('Execution Manager -- TX/Call Execution Functions', () => {
  const provider = ethers.provider

  let [wallet] = getWallets()

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ExecutionManager: ContractFactory
  let StateManager: ContractFactory
  let DummyContract: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    StateManager = await ethers.getContractFactory('StateManager')
    DummyContract = await ethers.getContractFactory('DummyContract')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          getDefaultGasMeterParams(),
        ],
      }
    )
  })

  let stateManager: Contract
  let dummyContractAddress: Address
  beforeEach(async () => {
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

    it('utilizes the DeployerWhitelist contract to disallow arbitrary contract deployment', async () => {
      wallet = wallet.connect(executionManager.provider)
      const DeployerWhitelist = await ethers.getContractFactory(
        'DeployerWhitelist'
      )

      // Initialize our deployment whitelist
      await callDeployerWhitelist(
        wallet,
        executionManager,
        DeployerWhitelist.interface.encodeFunctionData('initialize', [
          wallet.address,
          false,
        ])
      )

      // Generate our tx calldata
      const calldata = DeployerWhitelist.getDeployTransaction(
        wallet.address,
        false
      ).data // use the whitelist deployment as example initcode
      const nonce = await stateManager.getOvmContractNonceView(wallet.address)
      const transaction = {
        nonce,
        gasLimit: GAS_LIMIT,
        gasPrice: 0,
        to: ZERO_ADDRESS,
        value: 0,
        data: calldata,
        chainId: CHAIN_ID,
      }

      // Call using Ethers and expect it to fail
      await TestUtils.assertRevertsAsync(
        async () =>
          executionManager.executeTransaction(
            getCurrentTime(),
            0,
            transaction.to,
            transaction.data,
            wallet.address,
            ZERO_ADDRESS,
            true
          ),
        'Sender not allowed to deploy new contracts!'
      )

      // Now add the wallet.address to the whitelist and it should work this time!
      await callDeployerWhitelist(
        wallet,
        executionManager,
        DeployerWhitelist.interface.encodeFunctionData(
          'setWhitelistedDeployer',
          [wallet.address, true]
        )
      )

      await executionManager.executeTransaction(
        getCurrentTime(),
        0,
        transaction.to,
        transaction.data,
        wallet.address,
        ZERO_ADDRESS,
        true
      )

      // And then... set it to false, try to call, and it'll fail again!
      await callDeployerWhitelist(
        wallet,
        executionManager,
        DeployerWhitelist.interface.encodeFunctionData(
          'setWhitelistedDeployer',
          [wallet.address, false]
        )
      )

      await TestUtils.assertRevertsAsync(
        async () =>
          executionManager.executeTransaction(
            getCurrentTime(),
            0,
            transaction.to,
            transaction.data,
            wallet.address,
            ZERO_ADDRESS,
            true
          ),
        'Sender not allowed to deploy new contracts!'
      )

      // Lastly let's just let everyone deploy and make sure it worked!
      await callDeployerWhitelist(
        wallet,
        executionManager,
        DeployerWhitelist.interface.encodeFunctionData(
          'enableArbitraryContractDeployment'
        )
      )

      await executionManager.executeTransaction(
        getCurrentTime(),
        0,
        transaction.to,
        transaction.data,
        '0x' + '1234'.repeat(10),
        ZERO_ADDRESS,
        true
      )
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
        log.debug(e.stack)
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
        log.debug(e.stack)
        failed = true
      }

      failed.should.equal(true, `This call should have reverted!`)
    })
  })
})

const callDeployerWhitelist = async (
  wallet: Wallet,
  executionManager: Contract,
  ovmCallData: string
): Promise<string> => {
  const data = executionManager.interface.encodeFunctionData(
    'executeTransaction',
    [
      getCurrentTime(),
      0,
      DEPLOYER_WHITELIST_OVM_ADDRESS,
      ovmCallData,
      wallet.address,
      ZERO_ADDRESS,
      false,
    ]
  )

  const receipt = await wallet.sendTransaction({
    to: executionManager.address,
    data,
    gasLimit: GAS_LIMIT,
  })

  return receipt.hash
}
