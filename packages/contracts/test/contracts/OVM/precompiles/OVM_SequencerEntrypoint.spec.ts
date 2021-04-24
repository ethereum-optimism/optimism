import { expect } from '../../../setup'

/* External Imports */
import { waffle, ethers } from 'hardhat'
import { ContractFactory, Wallet, Contract, BigNumber } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'

/* Internal Imports */
import { DEFAULT_EIP155_TX } from '../../../helpers'
import { getContractInterface, getContractFactory } from '../../../../src'

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
    Mock__OVM_ExecutionManager.smocked.ovmCALL.will.return.with(
      (gasLimit, target, data) => {
        if (target === wallet.address) {
          return [
            true,
            iOVM_ECDSAContractAccount.encodeFunctionResult('execute', [
              true,
              '0x',
            ]),
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

    Helper_PredeployCaller = await (
      await ethers.getContractFactory('Helper_PredeployCaller')
    ).deploy()

    Helper_PredeployCaller.setTarget(Mock__OVM_ExecutionManager.address)
  })

  let OVM_SequencerEntrypointFactory: ContractFactory
  before(async () => {
    OVM_SequencerEntrypointFactory = getContractFactory(
      'OVM_SequencerEntrypoint',
      wallet,
      true
    )
  })

  const iOVM_ECDSAContractAccount = getContractInterface(
    'OVM_ECDSAContractAccount',
    true
  )

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

      const expectedEOACalldata = iOVM_ECDSAContractAccount.encodeFunctionData(
        'execute',
        [encodedTransaction]
      )
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

      const expectedEOACalldata = iOVM_ECDSAContractAccount.encodeFunctionData(
        'execute',
        [encodedTransaction]
      )
      const ovmCALL: any = Mock__OVM_ExecutionManager.smocked.ovmCALL.calls[0]
      expect(ovmCALL._address).to.equal(await wallet.getAddress())
      expect(ovmCALL._calldata).to.equal(expectedEOACalldata)
    })

    it(`should call ovmCreateEOA when ovmEXTCODESIZE returns 0`, async () => {
      let isFirstCheck = true
      Mock__OVM_ExecutionManager.smocked.ovmEXTCODESIZE.will.return.with(() => {
        if (isFirstCheck) {
          isFirstCheck = false
          return 0
        } else {
          return 1
        }
      })

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
