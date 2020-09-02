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
  signTransaction,
  getSignedComponents,
  getRawSignedComponents,
  deployAndRegister,
  getDefaultGasMeterParams,
  GAS_LIMIT,
  manuallyDeployOvmContract,
  executeTransaction,
} from '../../../test-helpers'

const boolToByte = (value: boolean): string => {
  return value ? '01' : '00'
}

const encodeSequencerCalldata = async (
  wallet: Wallet,
  transaction: UnsignedTransaction,
  isEOACreation: boolean,
  isEthSignedMessage: boolean
) => {
  const serializedTransaction = ethers.utils.serializeTransaction(transaction)
  const transactionHash = ethers.utils.keccak256(serializedTransaction)

  let v
  let r
  let s
  let messageHash
  if (isEthSignedMessage) {
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

  let calldata = `0x${boolToByte(isEOACreation)}${v}${r}${s}`
  if (isEOACreation) {
    calldata = `${calldata}${remove0x(messageHash)}`
  } else {
    calldata = `${calldata}${boolToByte(isEthSignedMessage)}${remove0x(
      serializedTransaction
    )}`
  }

  return calldata
}

const getMappingStorageSlot = (key: string, index: number): string => {
  const hexIndex = remove0x(
    ethers.BigNumber.from(index).toHexString()
  ).padStart(64, '0')
  return ethers.utils.keccak256(key + hexIndex)
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
    it('should call ovmCREATEEOA if the first byte is non-zero', async () => {
      const calldata = await encodeSequencerCalldata(
        wallet,
        {
          to: wallet.address,
          nonce: 1,
          data: '0x',
        },
        true,
        false
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

    it('should call an ECDSAContractAccount if the first byte is zero', async () => {
      const ovmCREATEEOAcalldata = await encodeSequencerCalldata(
        wallet,
        {
          to: wallet.address,
          nonce: 1,
          data: '0x',
        },
        true,
        false
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
          data: SimpleStorageFactory.interface.encodeFunctionData(
            'setStorage',
            [expectedKey, expectedVal]
          ),
        },
        false,
        false
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
