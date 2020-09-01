import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Wallet, Contract, ContractFactory, UnsignedTransaction } from 'ethers'

/* Internal Imports */
import {
  AddressResolverMapping,
  makeAddressResolver,
  getWallets,
  ZERO_ADDRESS,
  signTransaction,
  getSignedComponents,
  deployAndRegister,
  getDefaultGasMeterParams,
  GAS_LIMIT,
  manuallyDeployOvmContract,
  executeTransaction
} from '../../../test-helpers'

describe('ExecutionManager -- EOA Creation Opcodes', () => {
  let wallet: Wallet
  before(async () => {
    ;[wallet] = getWallets(ethers.provider)
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ECDSAContractAccountPrototype: Contract
  before(async () => {
    ECDSAContractAccountPrototype = resolver.contracts.ecdsaContractAccount
  })

  let ExecutionManagerFactory: ContractFactory
  let StateManagerFactory: ContractFactory
  let OVMNonceTesterFactory: ContractFactory
  before(async () => {
    ExecutionManagerFactory = await ethers.getContractFactory('ExecutionManager')
    StateManagerFactory = await ethers.getContractFactory('FullStateManager')
    OVMNonceTesterFactory = await ethers.getContractFactory('OVMNonceTester')
  })

  let ExecutionManger: Contract
  let StateManager: Contract
  beforeEach(async () => {
    ExecutionManger = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManagerFactory,
        params: [
          resolver.addressResolver.address,
          ZERO_ADDRESS,
          getDefaultGasMeterParams(),
        ],
      }
    )
    StateManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'StateManager',
      {
        factory: StateManagerFactory,
        params: [],
      },
    )
  })

  let OVMNonceTesterAddress: string
  beforeEach(async () => {
    OVMNonceTesterAddress = await manuallyDeployOvmContract(
      wallet,
      resolver.contracts.executionManager.provider,
      ExecutionManger,
      OVMNonceTesterFactory,
      [ExecutionManger.address],
      1
    )
  })

  describe('ovmCREATEEOA()', () => {
    let transaction: UnsignedTransaction
    before(async () => {
      transaction = {
        to: ZERO_ADDRESS,
        data: '0x1234'
      }
    })

    it('should create an EOA account given a signed message', async () => {
      const signedTransaction = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedTransaction)
      const serializedTransaction = ethers.utils.serializeTransaction(transaction)
      const transactionHash = ethers.utils.keccak256(serializedTransaction)

      await ExecutionManger.ovmCREATEEOA(
        transactionHash,
        v,
        r,
        s,
        {
          gasLimit: GAS_LIMIT
        }
      )

      const ecdsaPrototypeBytecode = await ethers.provider.getCode(ECDSAContractAccountPrototype.address)
      const codeContractAddress = await StateManager.ovmAddressToCodeContractAddress(wallet.address)
      const codeContractBytecode = await ethers.provider.getCode(codeContractAddress)

      expect(codeContractAddress).to.not.equal(ZERO_ADDRESS)
      expect(codeContractBytecode).to.equal(ecdsaPrototypeBytecode)
    })

    it('should revert if the EOA account already exists', async () => {
      const signedTransaction = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedTransaction)
      const serializedTransaction = ethers.utils.serializeTransaction(transaction)
      const transactionHash = ethers.utils.keccak256(serializedTransaction)

      await ExecutionManger.ovmCREATEEOA(
        transactionHash,
        v,
        r,
        s,
        {
          gasLimit: GAS_LIMIT
        }
      )

      await expect(ExecutionManger.ovmCREATEEOA(
        transactionHash,
        v,
        r,
        s,
        {
          gasLimit: GAS_LIMIT
        }
      )).to.be.revertedWith('EOA account has already been created.')
    })

    it('should revert if the provided signature is invalid', async () => {
      const signedTransaction = await signTransaction(wallet, transaction)
      const [v, r, s] = getSignedComponents(signedTransaction)
      const serializedTransaction = ethers.utils.serializeTransaction(transaction)
      const transactionHash = ethers.utils.keccak256(serializedTransaction)

      await expect(ExecutionManger.ovmCREATEEOA(
        transactionHash,
        v,
        r,
        '0x' + '00'.repeat(32), // Invalid 's' parameter.
        {
          gasLimit: GAS_LIMIT
        }
      )).to.be.revertedWith('Provided signature is invalid.')
    })
  })

  describe('ovmSETNONCE', () => {
    it('should set the nonce for the active address', async () => {
      const expectedNonce = 1234
      const calldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'setNonce',
        [
          expectedNonce
        ]
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        calldata,
        true,
        1
      )

      const actualNonce = await StateManager.getOvmContractNonceView(OVMNonceTesterAddress)
      expect(actualNonce).to.equal(expectedNonce)
    })

    it('should fail if new nonce is same as previous one', async () => {
      const expectedNonce = 1234
      const calldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'setNonce',
        [
          expectedNonce
        ]
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        calldata,
        true,
        1
      )

      await expect(executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        calldata,
        true,
        1
      )).to.be.rejectedWith('New nonce must be greater than the current nonce.')
    })

    it('should fail if new nonce is lower than previous one', async () => {
      const expectedNonce = 1234
      const firstCalldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'setNonce',
        [
          expectedNonce
        ]
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        firstCalldata,
        true,
        1
      )
      
      const secondCalldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'setNonce',
        [
          expectedNonce - 1 // Make nonce lower than previous one.
        ]
      )

      await expect(executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        secondCalldata,
        true,
        1
      )).to.be.rejectedWith('New nonce must be greater than the current nonce.')
    })

    it('should fail if attempting to call outside of an execution context', async () => {
      await expect(
        ExecutionManger.ovmSETNONCE(1234)
      ).to.be.rejectedWith('Must be inside a valid execution context.')
    })
  })

  describe('ovmGETNONCE', () => {
    it('should retrieve the nonce for the active address', async () => {
      const calldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'getNonce'
      )

      const expectedNonce = await StateManager.getOvmContractNonceView(OVMNonceTesterAddress)

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        calldata,
        true,
        1
      )

      // Nonce tester will increment the returned nonce by one to make sure
      // it's actually getting the correct nonce back.
      const actualNonce = await StateManager.getOvmContractNonceView(OVMNonceTesterAddress)
      expect(actualNonce.toNumber()).to.equal(expectedNonce.toNumber() + 1)
    })

    it('should retrieve the nonce after setting it', async () => {
      const expectedNonce = 1234
      const ovmSETNONCEcalldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'setNonce',
        [
          expectedNonce
        ]
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        ovmSETNONCEcalldata,
        true,
        1
      )

      const ovmGETNONCEcalldata = OVMNonceTesterFactory.interface.encodeFunctionData(
        'getNonce'
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        OVMNonceTesterAddress,
        ovmGETNONCEcalldata,
        true,
        1
      )

      // Nonce tester will increment the returned nonce by one to make sure
      // it's actually getting the correct nonce back.
      const actualNonce = await StateManager.getOvmContractNonceView(OVMNonceTesterAddress)
      expect(actualNonce.toNumber()).to.equal(expectedNonce + 1)
    })
  })
})
