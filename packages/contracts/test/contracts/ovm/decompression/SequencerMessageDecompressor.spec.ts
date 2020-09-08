import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Wallet, UnsignedTransaction, Contract } from 'ethers'
import { remove0x } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  AddressResolverMapping,
  makeAddressResolver,
  getWallets,
  ZERO_ADDRESS,
  getSignedComponents,
  getRawSignedComponents,
  deployAndRegister,
  getDefaultGasMeterParams,
  manuallyDeployOvmContract,
  executeTransaction,
} from '../../../test-helpers'

const encodeSequencerCalldata = async (
  wallet: Wallet,
  transaction: UnsignedTransaction,
  transactionType: number
) => {
  transaction.chainId = 108

  let serializedTransaction: string
  if (transactionType === 2) {
    serializedTransaction = ethers.utils.defaultAbiCoder.encode(
      ['uint256', 'uint256', 'uint256', 'address', 'bytes'],
      [
        transaction.nonce,
        transaction.gasLimit,
        transaction.gasPrice,
        transaction.to,
        transaction.data,
      ]
    )
  } else {
    serializedTransaction = ethers.utils.serializeTransaction(transaction)
  }

  const transactionHash = ethers.utils.keccak256(serializedTransaction)

  let v: string
  let r: string
  let s: string
  let messageHash: string
  if (transactionType === 2) {
    const transactionHashBytes = ethers.utils.arrayify(transactionHash)
    const transactionSignature = await wallet.signMessage(transactionHashBytes)
    ;[v, r, s] = getRawSignedComponents(transactionSignature).map(
      (component) => {
        return remove0x(component)
      }
    )
    messageHash = ethers.utils.hashMessage(transactionHashBytes)
  } else {
    const transactionSignature = await wallet.signTransaction(transaction)
    ;[v, r, s] = getSignedComponents(transactionSignature).map((component) => {
      return remove0x(component)
    })
    messageHash = transactionHash
  }

  let calldata = `0x0${transactionType}${v}${r}${s}`
  if (transactionType === 0) {
    calldata = `${calldata}${remove0x(messageHash)}`
  } else {
    calldata = `${calldata}${remove0x(serializedTransaction)}`
  }

  return calldata
}

describe('SequencerMessageDecompressor', () => {
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
  let SequencerMessageDecompressorFactory: ContractFactory
  let SimpleStorageFactory: ContractFactory
  before(async () => {
    ExecutionManagerFactory = await ethers.getContractFactory(
      'ExecutionManager'
    )
    StateManagerFactory = await ethers.getContractFactory('FullStateManager')
    SequencerMessageDecompressorFactory = await ethers.getContractFactory(
      'SequencerMessageDecompressor'
    )
    SimpleStorageFactory = await ethers.getContractFactory('SimpleStorage')
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
      }
    )
  })

  let SequencerMessageDecompressorAddress: string
  let SimpleStorageAddress: string
  beforeEach(async () => {
    SequencerMessageDecompressorAddress = await manuallyDeployOvmContract(
      wallet,
      resolver.contracts.executionManager.provider,
      ExecutionManger,
      SequencerMessageDecompressorFactory,
      [],
      1
    )
    SimpleStorageAddress = await manuallyDeployOvmContract(
      wallet,
      resolver.contracts.executionManager.provider,
      ExecutionManger,
      SimpleStorageFactory,
      [],
      1
    )
  })

  describe('fallback()', async () => {
    it('should call ovmCREATEEOA if the transaction type is zero', async () => {
      const calldata = await encodeSequencerCalldata(
        wallet,
        {
          to: wallet.address,
          nonce: 1,
          data: '0x',
        },
        0
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        SequencerMessageDecompressorAddress,
        calldata,
        true
      )

      const ecdsaPrototypeBytecode = await ethers.provider.getCode(
        ECDSAContractAccountPrototype.address
      )
      const codeContractAddress = await StateManager.ovmAddressToCodeContractAddress(
        wallet.address
      )
      const codeContractBytecode = await ethers.provider.getCode(
        codeContractAddress
      )

      expect(codeContractAddress).to.not.equal(ZERO_ADDRESS)
      expect(codeContractBytecode).to.equal(ecdsaPrototypeBytecode)
    })

    it('should call an ECDSAContractAccount when the transaction type is 1', async () => {
      const ovmCREATEEOAcalldata = await encodeSequencerCalldata(
        wallet,
        {
          to: wallet.address,
          nonce: 1,
          data: '0x',
        },
        0
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        SequencerMessageDecompressorAddress,
        ovmCREATEEOAcalldata,
        true
      )

      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = await encodeSequencerCalldata(
        wallet,
        {
          to: SimpleStorageAddress,
          nonce: 5,
          gasLimit: 2000000,
          data: SimpleStorageFactory.interface.encodeFunctionData(
            'setStorage',
            [expectedKey, expectedVal]
          ),
        },
        1
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        SequencerMessageDecompressorAddress,
        calldata,
        true
      )

      const codeContractAddress = await StateManager.ovmAddressToCodeContractAddress(
        SimpleStorageAddress
      )
      const SimpleStorage = SimpleStorageFactory.attach(codeContractAddress)
      const actualVal = await SimpleStorage.getStorage(expectedKey)
      expect(actualVal).to.equal(expectedVal)
    })

    it('should call an ECDSAContractAccount when the transaction type is 2', async () => {
      const ovmCREATEEOAcalldata = await encodeSequencerCalldata(
        wallet,
        {
          to: wallet.address,
          nonce: 1,
          data: '0x',
        },
        0
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        SequencerMessageDecompressorAddress,
        ovmCREATEEOAcalldata,
        true
      )

      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = await encodeSequencerCalldata(
        wallet,
        {
          to: SimpleStorageAddress,
          nonce: 5,
          gasLimit: 2000000,
          gasPrice: 0,
          data: SimpleStorageFactory.interface.encodeFunctionData(
            'setStorage',
            [expectedKey, expectedVal]
          ),
        },
        2
      )

      await executeTransaction(
        ExecutionManger,
        wallet,
        SequencerMessageDecompressorAddress,
        calldata,
        true
      )

      const codeContractAddress = await StateManager.ovmAddressToCodeContractAddress(
        SimpleStorageAddress
      )
      const SimpleStorage = SimpleStorageFactory.attach(codeContractAddress)
      const actualVal = await SimpleStorage.getStorage(expectedKey)
      expect(actualVal).to.equal(expectedVal)
    })
  })
})
