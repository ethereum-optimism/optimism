import { expect } from '../setup'

/* Imports: External */
import hre from 'hardhat'
import { ethers } from 'ethers'
import { remove0x } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  ChugSplashActionType,
  getContractDefinition,
  makeActionBundleFromConfig,
} from '../../src'
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../helpers'

describe('ChugSplash hardhat tooling', () => {
  describe('makeActionBundleFromConfig', () => {
    it('should make a bundle from config with one contract and no variables', async () => {
      const bundle = await makeActionBundleFromConfig(hre, {
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
        },
      })

      expect(bundle.actions.length).to.equal(1)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
    })

    it('should make a bundle from config with two contracts and no variables', async () => {
      const bundle = await makeActionBundleFromConfig(hre, {
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
          MyContract2: {
            address: `0x${'22'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
        },
      })

      expect(bundle.actions.length).to.equal(2)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
    })

    it('should make a bundle from config with one contract with variables', async () => {
      const bundle = await makeActionBundleFromConfig(hre, {
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _uint8: 123,
              _bytes32: NON_NULL_BYTES32,
            },
          },
        },
      })

      expect(bundle.actions.length).to.equal(3)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
    })

    it('should make a bundle from config with two contracts with variables', async () => {
      const bundle = await makeActionBundleFromConfig(hre, {
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _uint8: 123,
              _bytes32: NON_NULL_BYTES32,
            },
          },
          MyContract2: {
            address: `0x${'22'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _address: NON_ZERO_ADDRESS,
              _bool: true,
            },
          },
        },
      })

      expect(bundle.actions.length).to.equal(6)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
      expect(bundle.actions[3].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[4].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000e',
            `0x000000000000000000000000${remove0x(NON_ZERO_ADDRESS)}`,
          ]
        ),
      })
      expect(bundle.actions[5].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000c',
            '0x0000000000000000000000000000000000000000000000000000000000000001',
          ]
        ),
      })
    })

    it('should make a bundle from config with one contract and templated variables', async () => {
      const bundle = await makeActionBundleFromConfig(
        hre,
        {
          contracts: {
            MyContract1: {
              address: `0x${'11'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _uint8: `{{ env.MY_UINT8_VALUE }}`,
                _bytes32: `{{ env.MY_BYTES32_VALUE }}`,
              },
            },
          },
        },
        {
          MY_UINT8_VALUE: 123,
          MY_BYTES32_VALUE: NON_NULL_BYTES32,
        }
      )

      expect(bundle.actions.length).to.equal(3)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
    })

    it('should make a bundle from config with two contracts with variables and templated variables', async () => {
      const bundle = await makeActionBundleFromConfig(
        hre,
        {
          contracts: {
            MyContract1: {
              address: `0x${'11'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _uint8: 123,
                _bytes32: NON_NULL_BYTES32,
              },
            },
            MyContract2: {
              address: `0x${'22'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _address: `{{ env.MY_ADDRESS_VALUE }}`,
                _bool: `{{ env.MY_BOOLEAN_VALUE }}`,
              },
            },
          },
        },
        {
          MY_ADDRESS_VALUE: NON_ZERO_ADDRESS,
          MY_BOOLEAN_VALUE: true,
        }
      )

      expect(bundle.actions.length).to.equal(6)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
      expect(bundle.actions[3].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: getContractDefinition('Helper_StorageHelper').deployedBytecode,
      })
      expect(bundle.actions[4].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000e',
            `0x000000000000000000000000${remove0x(NON_ZERO_ADDRESS)}`,
          ]
        ),
      })
      expect(bundle.actions[5].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000c',
            '0x0000000000000000000000000000000000000000000000000000000000000001',
          ]
        ),
      })
    })
  })
})
