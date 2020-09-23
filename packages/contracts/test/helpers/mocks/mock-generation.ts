/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Signer, Contract, ContractFactory } from 'ethers'
import { FunctionFragment, ParamType } from 'ethers/lib/utils'

/* Internal Imports */
import { MockContract, MockContractFunction } from './mock-contract.types'
import { bindMockContractToVM, bindMockWatcherToVM } from './mock-binding'
import { SolidityCompiler, getDefaultCompiler, compile } from '../solidity'

/**
 * Generates contract code for a mock contract.
 * @param fns Mock contract function definitions.
 * @param contractName Name for the contract.
 * @param compilerVersion Compiler version being used.
 * @returns Contract code.
 */
const getSolidityMockContractCode = (
  fns: MockContractFunction[],
  contractName: string,
  compilerVersion: string
): string => {
  return `
    pragma solidity ${compilerVersion};

    contract ${contractName} {
      ${fns
        .map((fn) => {
          return `
        function ${fn.functionName}(${fn.inputTypes
            .map((inputType, idx) => {
              return `${inputType} _${idx}`
            })
            .join(', ')})
            public
        {
            return;
        }
        `
        })
        .join('\n')}
    }
  `
}

/**
 * Checks that a mock contract function definition is valid.
 * @param fn Mock contract function definition.
 * @returns Whether or not the function is valid.
 */
const isValidMockContractFunction = (fn: MockContractFunction): boolean => {
  return (
    fn.inputTypes &&
    fn.outputTypes &&
    fn.returnValues &&
    (!Array.isArray(fn.returnValues) ||
      fn.outputTypes.length === fn.returnValues.length)
  )
}

/**
 * Basic sanitization for mock function definitions
 * @param fn Mock contract function definition to sanitize.
 * @returns Sanitized definition.
 */
export const sanitizeMockContractFunction = (
  fn: MockContractFunction
): MockContractFunction => {
  const sanitized = {
    functionName: fn.functionName,
    inputTypes: fn.inputTypes || [],
    outputTypes: fn.outputTypes || [],
    returnValues: fn.returnValues || [],
  }

  if (!isValidMockContractFunction(sanitized)) {
    throw new Error(
      'Provided MockContract function is invalid. Please check your mock definition.'
    )
  }

  return sanitized
}

/**
 * Basic sanitization for mock function definitions
 * @param fns Mock contract function definitions to sanitize.
 * @returns Sanitized definitions.
 */
const sanitizeMockContractFunctions = (
  fns: MockContractFunction[]
): MockContractFunction[] => {
  return fns.map((fn) => {
    return sanitizeMockContractFunction(fn)
  })
}

/**
 * Gets mock return values for a set of output types.
 * @param outputTypes Output types as ethers param types.
 * @returns Mock return values.
 */
const getMockReturnValues = (outputTypes: ParamType[]): string[] => {
  return outputTypes.map((outputType) => {
    return outputType.type === outputType.baseType
      ? '0x' + '00'.repeat(32)
      : '0x' + '00'.repeat(64)
  })
}

/**
 * Converts an ethers function fragment to a mock function.
 * @param fn Function fragment to convert.
 * @returns Generated mock function.
 */
const getMockFunctionFromFragment = (
  fn: FunctionFragment
): MockContractFunction => {
  return {
    functionName: fn.name,
    inputTypes: [],
    outputTypes: [],
    returnValues: getMockReturnValues(fn.outputs),
  }
}

/**
 * Generates mock functions from a contract spec.
 * @param spec Contract or factory used as the spec.
 * @returns Array of mock functions.
 */
const getFnsFromContractSpec = (
  spec: Contract | ContractFactory
): MockContractFunction[] => {
  return Object.values(spec.interface.functions)
    .filter((fn) => {
      return fn.type === 'function'
    })
    .map((fn) => {
      return getMockFunctionFromFragment(fn)
    })
}

/**
 * Generates a mock contract for testing.
 * @param spec Mock contract function definitions or contract to base on.
 * @param signer Signer to use to deploy the mock.
 * @param compiler Optional compiler instance to use.
 * @returns Generated mock contract instance.
 */
export const getMockContract = async (
  spec: MockContractFunction[] | Contract | ContractFactory | string,
  signer?: Signer,
  compiler?: SolidityCompiler
): Promise<MockContract> => {
  if (typeof spec === 'string') {
    spec = await ethers.getContractFactory(spec)
  }

  if (!Array.isArray(spec)) {
    signer = signer || spec.signer
  }

  if (!signer) {
    throw new Error('You must provide a signer.')
  }

  compiler = compiler || (await getDefaultCompiler())

  const fns = Array.isArray(spec)
    ? sanitizeMockContractFunctions(spec)
    : getFnsFromContractSpec(spec)

  const contractName = 'MockContract'
  const contractPath = contractName + '.sol'
  const contractCode = getSolidityMockContractCode(
    fns,
    contractName,
    '^' + compiler.version().split('+')[0]
  )

  const compilerOutput = await compile(
    [
      {
        path: contractPath,
        content: contractCode,
      },
    ],
    compiler
  )

  const MockContractJSON = compilerOutput.contracts[contractPath][contractName]

  const MockContractFactory = new ethers.ContractFactory(
    MockContractJSON.abi,
    MockContractJSON.evm.bytecode.object,
    signer
  )

  const originalDefinePropertyFn = Object.defineProperty
  Object.defineProperty = (object: any, name: string, props: any): void => {
    if (props.writable === false) {
      props.writable = true
    }

    originalDefinePropertyFn(object, name, props)
  }

  const mock = (await MockContractFactory.deploy()) as MockContract

  Object.defineProperty = originalDefinePropertyFn

  bindMockWatcherToVM()
  bindMockContractToVM(mock, fns, spec)

  return mock
}
