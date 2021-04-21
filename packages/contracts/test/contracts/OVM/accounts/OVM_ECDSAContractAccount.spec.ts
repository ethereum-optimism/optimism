import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Wallet, BigNumber } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  serializeNativeTransaction,
  signNativeTransaction,
  DEFAULT_EIP155_TX,
  serializeEthSignTransaction,
  signEthSignMessage,
  decodeSolidityError,
  NON_ZERO_ADDRESS,
} from '../../../helpers'
import { getContractFactory, predeploys } from '../../../../src'

const callPredeploy = async (
  Helper_PredeployCaller: Contract,
  predeploy: Contract,
  functionName: string,
  functionParams?: any[],
  gasLimit?: number
): Promise<any> => {
  if (gasLimit) {
    return Helper_PredeployCaller.callPredeploy(
      predeploy.address,
      predeploy.interface.encodeFunctionData(
        functionName,
        functionParams || []
      ),
      { gasLimit }
    )
  }
  return Helper_PredeployCaller.callPredeploy(
    predeploy.address,
    predeploy.interface.encodeFunctionData(functionName, functionParams || [])
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
  let Helper_PredeployCaller: Contract
  before(async () => {
    Mock__OVM_ExecutionManager = await smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )

    Helper_PredeployCaller = await (
      await ethers.getContractFactory('Helper_PredeployCaller')
    ).deploy()

    Helper_PredeployCaller.setTarget(Mock__OVM_ExecutionManager.address)
  })

  let Factory__OVM_ECDSAContractAccount: ContractFactory
  before(async () => {
    Factory__OVM_ECDSAContractAccount = getContractFactory(
      'OVM_ECDSAContractAccount',
      wallet,
      true
    )
  })

  let OVM_ECDSAContractAccount: Contract
  beforeEach(async () => {
    OVM_ECDSAContractAccount = await Factory__OVM_ECDSAContractAccount.deploy()

    Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
      await wallet.getAddress()
    )
    Mock__OVM_ExecutionManager.smocked.ovmEXTCODESIZE.will.return.with(1)
    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
    Mock__OVM_ExecutionManager.smocked.ovmGETNONCE.will.return.with(100)
    Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with(
      (gasLimit, target, data) => {
        if (target === predeploys.OVM_ETH) {
          return [
            true,
            '0x0000000000000000000000000000000000000000000000000000000000000001',
          ]
        } else {
          return [true, '0x']
        }
      }
    )
    Mock__OVM_ExecutionManager.smocked.ovmSTATICCALL.will.return.with(
      (gasLimit, target, data) => {
        // Duplicating the behavior of the ecrecover precompile.
        if (target === '0x0000000000000000000000000000000000000001') {
          const databuf = fromHexString(data)
          const addr = ethers.utils.recoverAddress(databuf.slice(0, 32), {
            v: BigNumber.from(databuf.slice(32, 64)).toNumber(),
            r: toHexString(databuf.slice(64, 96)),
            s: toHexString(databuf.slice(96, 128)),
          })
          const ret = ethers.utils.defaultAbiCoder.encode(['address'], [addr])
          return [true, ret]
        } else {
          return [true, '0x']
        }
      }
    )
    Mock__OVM_ExecutionManager.smocked.ovmCREATE.will.return.with([
      NON_ZERO_ADDRESS,
      '0x',
    ])
    Mock__OVM_ExecutionManager.smocked.ovmCALLER.will.return.with(
      NON_ZERO_ADDRESS
    )
  })

  describe('fallback()', () => {
    it(`should successfully execute an EIP155Transaction`, async () => {
      const message = serializeNativeTransaction(DEFAULT_EIP155_TX)
      const sig = await signNativeTransaction(wallet, DEFAULT_EIP155_TX)

      await callPredeploy(
        Helper_PredeployCaller,
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
      expect(ovmCALL._address).to.equal(DEFAULT_EIP155_TX.to)
      expect(ovmCALL._calldata).to.equal(DEFAULT_EIP155_TX.data)
    })

    it(`should successfully execute an ETHSignedTransaction`, async () => {
      const message = serializeEthSignTransaction(DEFAULT_EIP155_TX)
      const sig = await signEthSignMessage(wallet, DEFAULT_EIP155_TX)

      await callPredeploy(
        Helper_PredeployCaller,
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
      expect(ovmCALL._address).to.equal(DEFAULT_EIP155_TX.to)
      expect(ovmCALL._calldata).to.equal(DEFAULT_EIP155_TX.data)
    })

    it(`should ovmCREATE if EIP155Transaction.to is zero address`, async () => {
      const createTx = { ...DEFAULT_EIP155_TX, to: '' }
      const message = serializeNativeTransaction(createTx)
      const sig = await signNativeTransaction(wallet, createTx)

      await callPredeploy(
        Helper_PredeployCaller,
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

      await callPredeploy(
        Helper_PredeployCaller,
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
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Signature provided for EOA transaction execution is invalid.'
      )
    })

    it(`should revert on incorrect nonce`, async () => {
      const alteredNonceTx = {
        ...DEFAULT_EIP155_TX,
        nonce: 99,
      }
      const message = serializeNativeTransaction(alteredNonceTx)
      const sig = await signNativeTransaction(wallet, alteredNonceTx)

      await callPredeploy(
        Helper_PredeployCaller,
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
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Transaction nonce does not match the expected nonce.'
      )
    })

    it(`should revert on incorrect chainId`, async () => {
      const alteredChainIdTx = {
        ...DEFAULT_EIP155_TX,
        chainId: 421,
      }
      const message = serializeNativeTransaction(alteredChainIdTx)
      const sig = await signNativeTransaction(wallet, alteredChainIdTx)

      await callPredeploy(
        Helper_PredeployCaller,
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
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Transaction chainId does not match expected OVM chainId.'
      )
    })

    // TEMPORARY: Skip gas checks for minnet.
    it.skip(`should revert on insufficient gas`, async () => {
      const alteredInsufficientGasTx = {
        ...DEFAULT_EIP155_TX,
        gasLimit: 200000000,
      }
      const message = serializeNativeTransaction(alteredInsufficientGasTx)
      const sig = await signNativeTransaction(wallet, alteredInsufficientGasTx)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Gas is not sufficient to execute the transaction.'
      )
    })

    it(`should revert if fee is not transferred to the relayer`, async () => {
      const message = serializeNativeTransaction(DEFAULT_EIP155_TX)
      const sig = await signNativeTransaction(wallet, DEFAULT_EIP155_TX)
      Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with(
        (gasLimit, target, data) => {
          if (target === '0x4200000000000000000000000000000000000006') {
            return [
              true,
              '0x0000000000000000000000000000000000000000000000000000000000000000',
            ]
          } else {
            return [true, '0x']
          }
        }
      )

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [
          message,
          0, //isEthSignedMessage
          `0x${sig.v}`, //v
          `0x${sig.r}`, //r
          `0x${sig.s}`, //s
        ],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Fee was not transferred to relayer.'
      )
    })
  })
})
