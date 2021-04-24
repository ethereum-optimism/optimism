import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Wallet, BigNumber } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import {
  NON_ZERO_ADDRESS,
  DEFAULT_EIP155_TX,
  decodeSolidityError,
} from '../../../helpers'
import {
  getContractFactory,
  getContractInterface,
  predeploys,
} from '../../../../src'

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

const iOVM_ETH = getContractInterface('OVM_ETH')

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

          let addr: string
          try {
            addr = ethers.utils.recoverAddress(databuf.slice(0, 32), {
              v: BigNumber.from(databuf.slice(32, 64)).toNumber(),
              r: toHexString(databuf.slice(64, 96)),
              s: toHexString(databuf.slice(96, 128)),
            })
          } catch (err) {
            addr = ethers.constants.AddressZero
          }

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
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction]
      )

      // The ovmCALL is the 2nd call because the first call transfers the fee.
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[1]
      expect(ovmCALL._address).to.equal(DEFAULT_EIP155_TX.to)
      expect(ovmCALL._calldata).to.equal(DEFAULT_EIP155_TX.data)
    })

    it(`should ovmCREATE if EIP155Transaction.to is zero address`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction]
      )

      const ovmCREATE: any =
        Mock__OVM_ExecutionManager.smocked.ovmCREATE.calls[0]
      expect(ovmCREATE._bytecode).to.equal(transaction.data)
    })

    it(`should revert on invalid signature`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = ethers.utils.serializeTransaction(
        transaction,
        '0x' + '00'.repeat(65)
      )

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction]
      )
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Signature provided for EOA transaction execution is invalid.'
      )
    })

    it(`should revert on incorrect nonce`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, nonce: 99 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction]
      )
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Transaction nonce does not match the expected nonce.'
      )
    })

    it(`should revert on incorrect chainId`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, chainId: 421 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction]
      )
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Lib_EIP155Tx: Transaction signed with wrong chain ID'
      )
    })

    // TEMPORARY: Skip gas checks for minnet.
    it.skip(`should revert on insufficient gas`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, gasLimit: 200000000 }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Gas is not sufficient to execute the transaction.'
      )
    })

    it(`should revert if fee is not transferred to the relayer`, async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with(
        (gasLimit, target, data) => {
          if (target === predeploys.OVM_ETH) {
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
        [encodedTransaction],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Fee was not transferred to relayer.'
      )
    })

    it(`should transfer value if value is greater than 0`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, data: '0x' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction],
        40000000
      )

      // First call transfers fee, second transfers value (since value > 0).
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[1]
      expect(ovmCALL._address).to.equal(predeploys.OVM_ETH)
      expect(ovmCALL._calldata).to.equal(
        iOVM_ETH.encodeFunctionData('transfer', [
          transaction.to,
          transaction.value,
        ])
      )
    })

    it(`should revert if the value is not transferred to the recipient`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, data: '0x' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with(
        (gasLimit, target, data) => {
          if (target === predeploys.OVM_ETH) {
            const [recipient, amount] = iOVM_ETH.decodeFunctionData(
              'transfer',
              data
            )
            if (recipient === transaction.to) {
              return [
                true,
                '0x0000000000000000000000000000000000000000000000000000000000000000',
              ]
            } else {
              return [
                true,
                '0x0000000000000000000000000000000000000000000000000000000000000001',
              ]
            }
          } else {
            return [true, '0x']
          }
        }
      )

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Value could not be transferred to recipient.'
      )
    })

    it(`should revert if trying to send value with a contract creation`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Value transfer in contract creation not supported.'
      )
    })

    it(`should revert if trying to send value with non-empty transaction data`, async () => {
      const transaction = { ...DEFAULT_EIP155_TX, value: 1234, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await callPredeploy(
        Helper_PredeployCaller,
        OVM_ECDSAContractAccount,
        'execute',
        [encodedTransaction],
        40000000
      )

      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(decodeSolidityError(ovmREVERT._data)).to.equal(
        'Value transfer in contract creation not supported.'
      )
    })
  })
})
