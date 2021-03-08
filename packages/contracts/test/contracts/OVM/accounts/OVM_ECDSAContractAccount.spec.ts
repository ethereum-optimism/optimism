import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from 'hardhat'
import { ContractFactory, Contract, Wallet } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'

/* Internal Imports */
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'
import { DEFAULT_EIP155_TX, decodeSolidityError } from '../../../helpers'

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
  before(async () => {
    const provider = waffle.provider
    ;[wallet] = provider.getWallets()
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
        'OVM_ECDSAContractAccount: Transaction was signed with the wrong chain ID.'
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

      Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with([false, '0x'])

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
  })
})
