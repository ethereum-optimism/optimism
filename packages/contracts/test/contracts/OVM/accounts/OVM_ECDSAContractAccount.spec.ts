import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from '@nomiclabs/buidler'
import { ContractFactory, Contract, Wallet } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { NON_ZERO_ADDRESS, ZERO_ADDRESS } from '../../../helpers/constants'
import {
  serializeNativeTransaction,
  signNativeTransaction,
  DEFAULT_EIP155_TX,
  serializeEthSignTransaction,
  signEthSignMessage,
  EIP155Transaction,
} from '../../../helpers'
import { defaultAbiCoder } from 'ethers/lib/utils'

const callPrecompile = async (
  Helper_PrecompileCaller: Contract,
  precompile: Contract,
  functionName: string,
  functionParams?: any[]
): Promise<any> => {
  return Helper_PrecompileCaller.callPrecompile(
    precompile.address,
    precompile.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

describe('OVM_ECDSAContractAccount', () => {
  let wallet: Wallet
  let badWallet: Wallet
  before(async () => {
    const provider = waffle.provider
    ;[wallet, badWallet] = provider.getWallets()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Helper_PrecompileCaller: Contract
  before(async () => {
    Mock__OVM_ExecutionManager = smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )

    Helper_PrecompileCaller = await (
      await ethers.getContractFactory('Helper_PrecompileCaller')
    ).deploy()
    Helper_PrecompileCaller.setTarget(Mock__OVM_ExecutionManager.address)
  })

  let Factory__OVM_ECDSAContractAccount: ContractFactory
  before(async () => {
    Factory__OVM_ECDSAContractAccount = await ethers.getContractFactory(
      'OVM_ECDSAContractAccount'
    )
  })

  let OVM_ECDSAContractAccount: Contract
  beforeEach(async () => {
    OVM_ECDSAContractAccount = await Factory__OVM_ECDSAContractAccount.deploy()

    Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
      await wallet.getAddress()
    )
    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
    Mock__OVM_ExecutionManager.smocked.ovmGETNONCE.will.return.with(100)
    Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with([true, '0x'])
    Mock__OVM_ExecutionManager.smocked.ovmCREATE.will.return.with(
      NON_ZERO_ADDRESS
    )
    Mock__OVM_ExecutionManager.smocked.ovmCALLER.will.return.with(
      NON_ZERO_ADDRESS
    )
  })

  describe('fallback()', () => {
    it(`should successfully execute an EIP155Transaction`, async () => {
      const message = serializeNativeTransaction(DEFAULT_EIP155_TX)
      const sig = await signNativeTransaction(wallet, DEFAULT_EIP155_TX)

      await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, // isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ]
      )

      // The ovmCALL is the 2nd call because the first call transfers the fee.
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[1]
      expect(ovmCALL._gasLimit).to.equal(DEFAULT_EIP155_TX.gasLimit)
      expect(ovmCALL._address).to.equal(DEFAULT_EIP155_TX.to)
      expect(ovmCALL._calldata).to.equal(DEFAULT_EIP155_TX.data)

      const ovmSETNONCE: any =
        Mock__OVM_ExecutionManager.smocked.ovmSETNONCE.calls[0]
      expect(ovmSETNONCE._nonce).to.equal(DEFAULT_EIP155_TX.nonce + 1)
    })

    it(`should successfully execute an ETHSignedTransaction`, async () => {
      const message = serializeEthSignTransaction(DEFAULT_EIP155_TX)
      const sig = await signEthSignMessage(wallet, DEFAULT_EIP155_TX)

      await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          1, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ]
      )

      // The ovmCALL is the 2nd call because the first call transfers the fee.
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[1]
      expect(ovmCALL._gasLimit).to.equal(DEFAULT_EIP155_TX.gasLimit)
      expect(ovmCALL._address).to.equal(DEFAULT_EIP155_TX.to)
      expect(ovmCALL._calldata).to.equal(DEFAULT_EIP155_TX.data)

      const ovmSETNONCE: any =
        Mock__OVM_ExecutionManager.smocked.ovmSETNONCE.calls[0]
      expect(ovmSETNONCE._nonce).to.equal(DEFAULT_EIP155_TX.nonce + 1)
    })

    it(`should ovmCREATE if EIP155Transaction.to is zero address`, async () => {
      const createTx = { ...DEFAULT_EIP155_TX, to: '' }
      const message = serializeNativeTransaction(createTx)
      const sig = await signNativeTransaction(wallet, createTx)

      await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ]
      )

      const ovmCREATE: any =
        Mock__OVM_ExecutionManager.smocked.ovmCREATE.calls[0]
      expect(ovmCREATE._bytecode).to.equal(createTx.data)
    })

    it(`should revert on invalid signature`, async () => {
      const message = serializeNativeTransaction(DEFAULT_EIP155_TX)
      const sig = await signNativeTransaction(badWallet, DEFAULT_EIP155_TX)

      await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ]
      )
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(ethers.utils.toUtf8String(ovmREVERT._data)).to.equal(
        'Signature provided for EOA transaction execution is invalid.'
      )
    })

    it(`should revert on incorrect nonce`, async () => {
      const alteredNonceTx = DEFAULT_EIP155_TX
      alteredNonceTx.nonce = 99
      const message = serializeNativeTransaction(alteredNonceTx)
      const sig = await signNativeTransaction(wallet, alteredNonceTx)

      await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ]
      )
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(ethers.utils.toUtf8String(ovmREVERT._data)).to.equal(
        'Transaction nonce does not match the expected nonce.'
      )
    })
  })
})
