import { expect } from '../../../setup'

/* External Imports */
import { waffle, ethers } from 'hardhat'
import { ContractFactory, Wallet, Contract } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import { getContractInterface } from '../../../../src'
import { DEFAULT_EIP155_TX } from '../../../helpers'

describe('OVM_SequencerEntrypoint', () => {
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

    Mock__OVM_ExecutionManager.smocked.ovmCHAINID.will.return.with(420)
    Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with([true, '0x'])

    Helper_PredeployCaller = await (
      await ethers.getContractFactory('Helper_PredeployCaller')
    ).deploy()

    Helper_PredeployCaller.setTarget(Mock__OVM_ExecutionManager.address)
  })

  let OVM_SequencerEntrypointFactory: ContractFactory
  before(async () => {
    OVM_SequencerEntrypointFactory = await ethers.getContractFactory(
      'OVM_SequencerEntrypoint'
    )
  })

  let OVM_SequencerEntrypoint: Contract
  beforeEach(async () => {
    OVM_SequencerEntrypoint = await OVM_SequencerEntrypointFactory.deploy()
    Mock__OVM_ExecutionManager.smocked.ovmEXTCODESIZE.will.return.with(1)
    Mock__OVM_ExecutionManager.smocked.ovmREVERT.will.revert()
  })

  describe('fallback()', async () => {
    it('should call EIP155', async () => {
      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      await Helper_PredeployCaller.callPredeploy(
        OVM_SequencerEntrypoint.address,
        encodedTransaction
      )

      const expectedEOACalldata = getContractInterface(
        'OVM_ECDSAContractAccount'
      ).encodeFunctionData('execute', [encodedTransaction])
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[0]
      expect(ovmCALL._address).to.equal(await wallet.getAddress())
      expect(ovmCALL._calldata).to.equal(expectedEOACalldata)
    })

    it('should send correct calldata if tx is a create', async () => {
      const transaction = { ...DEFAULT_EIP155_TX, to: '' }
      const encodedTransaction = await wallet.signTransaction(transaction)

      await Helper_PredeployCaller.callPredeploy(
        OVM_SequencerEntrypoint.address,
        encodedTransaction
      )

      const expectedEOACalldata = getContractInterface(
        'OVM_ECDSAContractAccount'
      ).encodeFunctionData('execute', [encodedTransaction])
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[0]
      expect(ovmCALL._address).to.equal(await wallet.getAddress())
      expect(ovmCALL._calldata).to.equal(expectedEOACalldata)
    })

    it(`should call ovmCreateEOA when ovmEXTCODESIZE returns 0`, async () => {
      Mock__OVM_ExecutionManager.smocked.ovmEXTCODESIZE.will.return.with(0)

      const transaction = DEFAULT_EIP155_TX
      const encodedTransaction = await wallet.signTransaction(transaction)

      await Helper_PredeployCaller.callPredeploy(
        OVM_SequencerEntrypoint.address,
        encodedTransaction
      )

      const call: any = Mock__OVM_ExecutionManager.smocked.ovmCREATEEOA.calls[0]
      const eoaAddress = ethers.utils.recoverAddress(call._messageHash, {
        v: call._v + 27,
        r: call._r,
        s: call._s,
      })

      expect(eoaAddress).to.equal(await wallet.getAddress())
    })
  })
})
