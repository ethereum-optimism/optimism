import { expect } from '../../../setup'

/* External Imports */
import { ethers, waffle } from '@nomiclabs/buidler'
import { ContractFactory, Contract, Wallet } from 'ethers'
import { MockContract, smockit } from '@eth-optimism/smock'
import { NON_ZERO_ADDRESS } from '../../../helpers/constants'
import { AbiCoder, keccak256 } from 'ethers/lib/utils'
import {
  DEFAULT_EIP155_TX,
  remove0x,
  serializeNativeTransaction,
  signNativeTransaction,
} from '../../../helpers'
import { getContractInterface } from '../../../../src'

const callPrecompile = async (
  Helper_PrecompileCaller: Contract,
  precompile: Contract,
  functionName: string,
  functionParams?: any[],
  ethCall: boolean = false
): Promise<any> => {
  if (ethCall) {
    return Helper_PrecompileCaller.callStatic.callPrecompileAbi(
      precompile.address,
      precompile.interface.encodeFunctionData(
        functionName,
        functionParams || []
      )
    )
  }
  return Helper_PrecompileCaller.callPrecompile(
    precompile.address,
    precompile.interface.encodeFunctionData(functionName, functionParams || [])
  )
}

const addrToBytes32 = (addr: string) => '0x' + '00'.repeat(12) + remove0x(addr)

const eoaDefaultAddr = '0x4200000000000000000000000000000000000003'

describe('OVM_ProxyEOA', () => {
  let wallet: Wallet
  before(async () => {
    const provider = waffle.provider
    ;[wallet] = provider.getWallets()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_ECDSAContractAccount: MockContract
  let Helper_PrecompileCaller: Contract
  before(async () => {
    Mock__OVM_ExecutionManager = smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )

    Helper_PrecompileCaller = await (
      await ethers.getContractFactory('Helper_PrecompileCaller')
    ).deploy()

    Helper_PrecompileCaller.setTarget(Mock__OVM_ExecutionManager.address)

    Mock__OVM_ECDSAContractAccount = smockit(
      await ethers.getContractFactory('OVM_ECDSAContractAccount')
    )
  })

  let OVM_ProxyEOAFactory: ContractFactory
  before(async () => {
    OVM_ProxyEOAFactory = await ethers.getContractFactory('OVM_ProxyEOA')
  })

  let OVM_ProxyEOA: Contract
  beforeEach(async () => {
    OVM_ProxyEOA = await OVM_ProxyEOAFactory.deploy(eoaDefaultAddr)

    Mock__OVM_ExecutionManager.smocked.ovmADDRESS.will.return.with(
      OVM_ProxyEOA.address
    )
    Mock__OVM_ExecutionManager.smocked.ovmCALLER.will.return.with(
      OVM_ProxyEOA.address
    )
  })

  describe('getImplementation()', () => {
    it(`should be created with implementation at precompile address`, async () => {
      const eoaDefaultAddrBytes32 = addrToBytes32(eoaDefaultAddr)
      Mock__OVM_ExecutionManager.smocked.ovmSLOAD.will.return.with(
        eoaDefaultAddrBytes32
      )
      const implAddrBytes32 = await callPrecompile(
        Helper_PrecompileCaller,
        OVM_ProxyEOA,
        'getImplementation',
        [],
        true
      )
      expect(implAddrBytes32).to.equal(eoaDefaultAddrBytes32)
    })
  })
  describe('upgrade()', () => {
    const implSlotKey = '0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead'
    it(`should upgrade the proxy implementation`, async () => {
      const newImpl = `0x${'81'.repeat(20)}`
      const newImplBytes32 = addrToBytes32(newImpl)
      await callPrecompile(Helper_PrecompileCaller, OVM_ProxyEOA, 'upgrade', [
        newImpl,
      ])
      const ovmSSTORE: any =
        Mock__OVM_ExecutionManager.smocked.ovmSSTORE.calls[0]
      expect(ovmSSTORE._key).to.equal(implSlotKey)
      expect(ovmSSTORE._value).to.equal(newImplBytes32)
    })
    it(`should not allow upgrade of the proxy implementation by another account`, async () => {
      Mock__OVM_ExecutionManager.smocked.ovmCALLER.will.return.with(
        await wallet.getAddress()
      )
      const newImpl = `0x${'81'.repeat(20)}`
      await callPrecompile(Helper_PrecompileCaller, OVM_ProxyEOA, 'upgrade', [
        newImpl,
      ])
      const ovmREVERT: any =
        Mock__OVM_ExecutionManager.smocked.ovmREVERT.calls[0]
      expect(ethers.utils.toUtf8String(ovmREVERT._data)).to.equal(
        'EOAs can only upgrade their own EOA implementation'
      )
    })
  })
  describe('fallback()', () => {
    it(`should call delegateCall with right calldata`, async () => {
      Mock__OVM_ExecutionManager.smocked.ovmSLOAD.will.return.with(
        addrToBytes32(Mock__OVM_ECDSAContractAccount.address)
      )
      Mock__OVM_ExecutionManager.smocked.ovmDELEGATECALL.will.return.with([
        true,
        '0x1234',
      ])
      const calldata = '0xdeadbeef'
      await Helper_PrecompileCaller.callPrecompile(
        OVM_ProxyEOA.address,
        calldata
      )

      const ovmDELEGATECALL: any =
        Mock__OVM_ExecutionManager.smocked.ovmDELEGATECALL.calls[0]
      expect(ovmDELEGATECALL._address).to.equal(
        Mock__OVM_ECDSAContractAccount.address
      )
      expect(ovmDELEGATECALL._calldata).to.equal(calldata)
    })
    it.skip(`should return data from fallback`, async () => {
      //TODO test return data from fallback
    })
    it.skip(`should revert in fallback`, async () => {
      //TODO test reversion from fallback
    })
  })
})
