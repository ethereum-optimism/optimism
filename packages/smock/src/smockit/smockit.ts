/* Imports: External */
import hre from 'hardhat'
import { Contract, ContractFactory, ethers } from 'ethers'
import { toHexString, fromHexString } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  isArtifact,
  isContract,
  isContractFactory,
  isInterface,
  MockContract,
  MockContractFunction,
  MockReturnValue,
  SmockedVM,
  SmockOptions,
  SmockSpec,
} from './types'
import { bindSmock, unbindSmock } from './binding'
import { makeRandomAddress } from '../utils'
import { findBaseHardhatProvider } from '../common'

/**
 * Generates an ethers Interface instance when given a smock spec. Meant for standardizing the
 * various input types we might reasonably want to support.
 *
 * @param spec Smock specification object. Thing you want to base the interface on.
 * @param hre Hardhat runtime environment. Used so we can
 * @return Interface generated from the spec.
 */
const makeContractInterfaceFromSpec = async (
  spec: SmockSpec
): Promise<ethers.utils.Interface> => {
  if (spec instanceof Contract) {
    return spec.interface
  } else if (spec instanceof ContractFactory) {
    return spec.interface
  } else if (spec instanceof ethers.utils.Interface) {
    return spec
  } else if (isInterface(spec)) {
    return spec as any
  } else if (isContractFactory(spec)) {
    return (spec as any).interface
  } else if (isContract(spec)) {
    return (spec as any).interface
  } else if (isArtifact(spec)) {
    return new ethers.utils.Interface(spec.abi)
  } else if (typeof spec === 'string') {
    try {
      return new ethers.utils.Interface(spec)
    } catch (err) {
      return (await (hre as any).ethers.getContractFactory(spec)).interface
    }
  } else {
    return new ethers.utils.Interface(spec)
  }
}

/**
 * Creates a mock contract function from a real contract function.
 *
 * @param contract Contract object to make a mock function for.
 * @param functionName Name of the function to mock.
 * @param vm Virtual machine reference, necessary for call assertions to work.
 * @return Mock contract function.
 */
const smockifyFunction = (
  contract: Contract,
  functionName: string,
  vm: SmockedVM
): MockContractFunction => {
  return {
    reset: () => {
      return
    },
    get calls() {
      return (vm._smockState.calls[contract.address.toLowerCase()] || [])
        .map((calldataBuf: Buffer) => {
          const sighash = toHexString(calldataBuf.slice(0, 4))
          const fragment = contract.interface.getFunction(sighash)

          let data: any = toHexString(calldataBuf)
          try {
            data = contract.interface.decodeFunctionData(
              fragment.format(),
              data
            )
          } catch (e) {
            console.error(e)
          }

          return {
            functionName: fragment.name,
            functionSignature: fragment.format(),
            data,
          }
        })
        .filter((functionResult: any) => {
          return (
            functionResult.functionName === functionName ||
            functionResult.functionSignature === functionName
          )
        })
        .map((functionResult: any) => {
          return functionResult.data
        })
    },
    will: {
      get return() {
        const fn: any = () => {
          this.resolve = 'return'
          this.returnValue = undefined
        }

        fn.with = (returnValue?: MockReturnValue): void => {
          this.resolve = 'return'
          this.returnValue = returnValue
        }

        return fn
      },
      get revert() {
        const fn: any = () => {
          this.resolve = 'revert'
          this.returnValue = undefined
        }

        fn.with = (revertValue?: string): void => {
          this.resolve = 'revert'
          this.returnValue = revertValue
        }

        return fn
      },
      resolve: 'return',
    },
  }
}

/**
 * Turns a specification into a mock contract.
 *
 * @param spec Smock contract specification.
 * @param opts Optional additional settings.
 */
export const smockit = async (
  spec: SmockSpec,
  opts: SmockOptions = {}
): Promise<MockContract> => {
  // Only support native hardhat runtime, haven't bothered to figure it out for anything else.
  if (hre.network.name !== 'hardhat') {
    throw new Error(
      `[smock]: smock is only compatible with the "hardhat" network, got: ${hre.network.name}`
    )
  }

  // Find the provider object. See comments for `findBaseHardhatProvider`
  const provider = findBaseHardhatProvider(hre)

  // Sometimes the VM hasn't been initialized by the time we get here, depending on what the user
  // is doing with hardhat (e.g., sending a transaction before calling this function will
  // initialize the vm). Initialize it here if it hasn't been already.
  if ((provider as any)._node === undefined) {
    await (provider as any)._init()
  }

  // Generate the contract object that we're going to attach our fancy functions to. Doing it this
  // way is nice because it "feels" more like a contract (as long as you're using ethers).
  const contract = new ethers.Contract(
    opts.address || makeRandomAddress(),
    await makeContractInterfaceFromSpec(spec),
    opts.provider || (hre as any).ethers.provider // TODO: Probably check that this exists.
  ) as MockContract

  // We attach a wallet to the contract so that users can send transactions *from* a smock.
  await hre.network.provider.request({
    method: 'hardhat_impersonateAccount',
    params: [contract.address],
  })

  // Now we actually get the signer and attach it to the mock.
  contract.wallet = await (hre as any).ethers.getSigner(contract.address)

  // Start by smocking the fallback.
  contract.smocked = {
    fallback: smockifyFunction(
      contract,
      'fallback',
      (provider as any)._node._vm
    ),
  }

  // Smock the rest of the contract functions.
  for (const functionName of Object.keys(contract.functions)) {
    contract.smocked[functionName] = smockifyFunction(
      contract,
      functionName,
      (provider as any)._node._vm
    )
  }

  // TODO: Make this less of a hack.
  ;(contract as any)._smockit = async function (
    data: Buffer
  ): Promise<{
    resolve: 'return' | 'revert'
    functionName: string
    rawReturnValue: any
    returnValue: Buffer
    gasUsed: number
  }> {
    let fn: any
    try {
      const sighash = toHexString(data.slice(0, 4))
      fn = this.interface.getFunction(sighash)
    } catch (err) {
      fn = null
    }

    let params: any
    let mockFn: any
    if (fn !== null) {
      params = this.interface.decodeFunctionData(fn, toHexString(data))
      mockFn = this.smocked[fn.name] || this.smocked[fn.format()]
    } else {
      params = toHexString(data)
      mockFn = this.smocked.fallback
    }

    const rawReturnValue =
      mockFn.will?.returnValue instanceof Function
        ? await mockFn.will.returnValue(...params)
        : mockFn.will.returnValue

    let encodedReturnValue: string = '0x'
    if (rawReturnValue !== undefined) {
      if (mockFn.will?.resolve === 'revert') {
        if (typeof rawReturnValue !== 'string') {
          throw new Error(
            `Smock: Tried to revert with a non-string (or non-bytes) type: ${typeof rawReturnValue}`
          )
        }

        if (rawReturnValue.startsWith('0x')) {
          encodedReturnValue = rawReturnValue
        } else {
          const errorface = new ethers.utils.Interface([
            {
              inputs: [
                {
                  name: '_reason',
                  type: 'string',
                },
              ],
              name: 'Error',
              outputs: [],
              stateMutability: 'nonpayable',
              type: 'function',
            },
          ])

          encodedReturnValue = errorface.encodeFunctionData('Error', [
            rawReturnValue,
          ])
        }
      } else {
        if (fn === null) {
          encodedReturnValue = rawReturnValue
        } else {
          try {
            encodedReturnValue = this.interface.encodeFunctionResult(fn, [
              rawReturnValue,
            ])
          } catch (err) {
            if (err.code === 'INVALID_ARGUMENT') {
              try {
                encodedReturnValue = this.interface.encodeFunctionResult(
                  fn,
                  rawReturnValue
                )
              } catch {
                if (typeof rawReturnValue !== 'string') {
                  throw new Error(
                    `Could not properly encode mock return value for ${fn.name}`
                  )
                }

                encodedReturnValue = rawReturnValue
              }
            } else {
              throw err
            }
          }
        }
      }
    } else {
      if (fn === null) {
        encodedReturnValue = '0x'
      } else {
        encodedReturnValue = '0x' + '00'.repeat(2048)
      }
    }

    return {
      resolve: mockFn.will?.resolve,
      functionName: fn ? fn.name : null,
      rawReturnValue,
      returnValue: fromHexString(encodedReturnValue),
      gasUsed: mockFn.gasUsed || 0,
    }
  }

  await bindSmock(contract, provider)

  return contract
}

/**
 * Unbinds a mock contract (meaning the contract will no longer behave as a mock).
 *
 * @param mock Mock contract or address to unbind.
 */
export const unbind = async (mock: MockContract | string): Promise<void> => {
  // Only support native hardhat runtime, haven't bothered to figure it out for anything else.
  if (hre.network.name !== 'hardhat') {
    throw new Error(
      `[smock]: smock is only compatible with the "hardhat" network, got: ${hre.network.name}`
    )
  }

  // Find the provider object. See comments for `findBaseHardhatProvider`
  const provider = findBaseHardhatProvider(hre)

  // Unbind the contract.
  await unbindSmock(mock, provider)
}
