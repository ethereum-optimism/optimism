/* External Imports */
import { Contract } from 'ethers'
import { Interface, AbiCoder } from 'ethers/lib/utils'
import * as path from 'path'
import { REVERT_FLAGS, encodeRevertData, decodeRevertData } from '../codec'

/* Internal Imports */
import { TestStep } from './test.types'

const abi = new AbiCoder()

const getContractDefinition = (name: string): any => {
  return require(path.join(__dirname, '../../../artifacts', `${name}.json`))
}

export const getInitcode = (name: string): string => {
  return getContractDefinition(name).bytecode
}

export const getBytecode = (name: string): string => {
  return getContractDefinition(name).deployedBytecode
}

export interface TestCallGenerator {
  getCalldata(): string
  shouldCallSucceed(): boolean
  getReturnData(): string
  getFunctionName(): string
  interpretActualReturnData(data: string, succeeded: boolean): string
}

export class DefaultTestGenerator implements TestCallGenerator {
  constructor(
    protected ovmExecutionManager: Contract,
    protected ovmCallHelper: Contract,
    protected ovmCreateStorer: Contract,
    protected ovmCreateHelper: Interface,
    protected ovmRevertHelper: Contract,
    protected ovmInvalidHelper: Contract,
    protected step: TestStep
  ) {}

  getFunctionParams(): any[] {
    return this.step.functionParams
  }

  getReturnValues(): any[] {
    return this.step.expectedReturnValues
  }

  getCalldata(): string {
    return this.ovmExecutionManager.interface.encodeFunctionData(
      this.step.functionName,
      this.getFunctionParams()
    )
  }

  shouldCallSucceed(): boolean {
    return this.step.expectedReturnStatus
  }

  getFunctionName(): string {
    return this.step.functionName
  }

  getReturnData(): string {
    let expectedReturnData
    if (this.step.expectedReturnStatus) {
      expectedReturnData = this.ovmExecutionManager.interface.encodeFunctionResult(
        this.step.functionName,
        this.getReturnValues()
      )
    } else {
      expectedReturnData = encodeRevertData(
        this.step.expectedReturnValues[0],
        this.step.expectedReturnValues[1],
        this.step.expectedReturnValues[2],
        this.step.expectedReturnValues[3]
      )
    }
    return expectedReturnData
  }

  interpretActualReturnData(data: string, succeeded: boolean): string {
    let interpretation: string =
      'call to EM.' + this.step.functionName + ' returned values:'
    interpretation += succeeded
      ? this.ovmExecutionManager.interface.decodeFunctionResult(
          this.step.functionName,
          data
        )
      : decodeRevertData(data)
    return interpretation
  }
}

export class ovmCALLGenerator extends DefaultTestGenerator {
  getCalleeGenerators(): TestCallGenerator[] {
    return (this.step.functionParams[2] as TestStep[]).map((step) => {
      return getTestGenerator(
        step,
        this.ovmExecutionManager,
        this.ovmCallHelper,
        this.ovmCreateStorer,
        this.ovmCreateHelper,
        this.ovmRevertHelper,
        this.ovmInvalidHelper
      )
    })
  }

  getFunctionParams(): any[] {
    return [
      this.step.functionParams[0],
      this.step.functionParams[1],
      this.ovmCallHelper.interface.encodeFunctionData('runSteps', [
        this.getCalleeGenerators().map((calleeGenerator) => {
          return calleeGenerator.getCalldata()
        }),
        !this.step.expectedReturnStatus,
        this.ovmCreateStorer.address,
      ]),
    ]
  }

  getReturnValues(): any[] {
    if (this.step.expectedReturnValues.length <= 1) {
      return [
        !this.step.expectedReturnValues[0],
        this.ovmCallHelper.interface.encodeFunctionResult('runSteps', [
          this.getCalleeGenerators().map((calleeGenerator) => {
            return {
              success: calleeGenerator.shouldCallSucceed(),
              data: calleeGenerator.getReturnData(),
            }
          }),
        ]),
      ]
    } else {
      return this.step.expectedReturnValues
    }
  }

  interpretActualReturnData(data: string, success: boolean): string {
    if (!success) {
      return 'ovmCALL-type reverted with flag:' + decodeRevertData(data)
    }

    if (this.step.expectedReturnValues.length > 1) {
      return (
        'ovmCALL-type returned successfully with overridden return data: ' +
        data
      )
    }

    const resultOfOvmCALL = this.ovmExecutionManager.interface.decodeFunctionResult(
      this.step.functionName,
      data
    )
    const resultOfSubcalls = this.ovmCallHelper.interface.decodeFunctionResult(
      'runSteps',
      resultOfOvmCALL[1]
    )[0]

    const calleeGenerators = this.getCalleeGenerators()
    const interpretedResults = resultOfSubcalls.map((result, i) => {
      const generator = calleeGenerators[i]
      const EMsuccess = result[0]
      const EMdata = result[1]
      return (
        'subcall ' +
        i +
        '(' +
        generator.getFunctionName() +
        ') had return status: ' +
        EMsuccess +
        ' and appears to have done: ' +
        generator.interpretActualReturnData(EMdata, EMsuccess)
      )
    })

    return (
      'ovmCALL returned ' +
      resultOfOvmCALL[0] +
      ' \n      with subcalls:' +
      JSON.stringify(interpretedResults) +
      '\n'
    )
  }
}

export class ovmCREATEGenerator extends DefaultTestGenerator {
  getInitcodeGenerators(): TestCallGenerator[] {
    return (this.step.functionParams[2] as TestStep[]).map((step) => {
      return getTestGenerator(
        step,
        this.ovmExecutionManager,
        this.ovmCallHelper,
        this.ovmCreateStorer,
        this.ovmCreateHelper,
        this.ovmRevertHelper,
        this.ovmInvalidHelper
      )
    })
  }

  getFunctionParams(): any[] {
    return [
      getInitcode('Helper_CodeContractForCreates') +
        this.ovmCreateHelper
          .encodeDeploy([
            this.getInitcodeGenerators().map((initcodeGenerator) => {
              return initcodeGenerator.getCalldata()
            }),
            !this.step.expectedReturnStatus,
            this.step.functionParams[0],
            this.ovmCreateStorer.address,
          ])
          .slice(2),
    ]
  }

  getReturnData(): string {
    const expectedDirectEMReturnData = this.step.expectedReturnStatus
      ? this.ovmExecutionManager.interface.encodeFunctionResult(
          this.step.functionName,
          this.getReturnValues()
        )
      : encodeRevertData(
          this.step.expectedReturnValues[0],
          this.step.expectedReturnValues[1],
          this.step.expectedReturnValues[2],
          this.step.expectedReturnValues[3]
        )

    const responsesShouldBeReverted = !this.step.functionParams[1]
    const expectedStoredValues = responsesShouldBeReverted
      ? []
      : this.getInitcodeGenerators().map((initcodeGenerator) => {
          return {
            success: initcodeGenerator.shouldCallSucceed(), // TODO: figure out if we need this and expose in generator interface if so.
            data: initcodeGenerator.getReturnData(),
          }
        })

    const expectedReturnData = abi.encode(
      ['bytes', 'bytes'],
      [
        expectedDirectEMReturnData,
        this.ovmCreateStorer.interface.encodeFunctionResult(
          'getLastResponses',
          [expectedStoredValues]
        ),
      ]
    )
    return expectedReturnData
  }

  interpretActualReturnData(data: string, success: boolean): string {
    const ovmCREATEDataAndInitcodeResults = abi.decode(['bytes', 'bytes'], data)
    const ovmCREATEData = ovmCREATEDataAndInitcodeResults[0]

    if (!success) {
      return (
        'ovmCREATE reverted with: ' + decodeRevertData(ovmCREATEData.toString())
      )
    }

    // const decodedDataFromOvmCREATE = this.ovmExecutionManager.interface.decodeFunctionResult(this.step.functionName, ovmCREATEData)

    const initcodeResultsRaw = ovmCREATEDataAndInitcodeResults[1]
    const initcodeResults = this.ovmCreateStorer.interface.decodeFunctionResult(
      'getLastResponses',
      initcodeResultsRaw
    )[0]

    const interpretedInitcodeResults = initcodeResults.map((result, i) => {
      return (
        '\n       initcode subcall ' +
        i +
        ' had response status ' +
        result[0] +
        ' and appears to have done: ' +
        this.getInitcodeGenerators()[i].interpretActualReturnData(
          result[1],
          result[0]
        )
      )
    })
    return JSON.stringify(interpretedInitcodeResults)
  }
}

class ovmCALLToRevertGenerator extends DefaultTestGenerator {
  getCalldata(): string {
    return this.ovmExecutionManager.interface.encodeFunctionData('ovmCALL', [
      this.step.functionParams[0],
      this.step.functionParams[1],
      this.ovmRevertHelper.interface.encodeFunctionData('doRevert', [
        encodeRevertData(
          this.step.functionParams[2][0],
          this.step.functionParams[2][1],
          this.step.functionParams[2][2],
          this.step.functionParams[2][3]
        ),
      ]),
    ])
  }

  getReturnData(): string {
    let expectedReturnData
    if (this.step.expectedReturnStatus) {
      expectedReturnData = this.ovmExecutionManager.interface.encodeFunctionResult(
        'ovmCALL',
        this.step.expectedReturnValues
      )
    } else {
      expectedReturnData = encodeRevertData(
        this.step.expectedReturnValues[0],
        this.step.expectedReturnValues[1],
        this.step.expectedReturnValues[2],
        this.step.expectedReturnValues[3]
      )
    }
    return expectedReturnData
  }

  interpretActualReturnData(data: string, success: boolean): string {
    if (success) {
      return (
        'ovmCALL to revert heler, which succeeded with return params:' +
        JSON.stringify(
          this.ovmExecutionManager.interface.decodeFunctionResult(
            'ovmCALL',
            data
          )
        )
      )
    } else {
      return (
        'ovmCALL to revert helper did itself revert with flag:' +
        JSON.stringify(decodeRevertData(data))
      )
    }
  }
}

class ovmSTATICCALLToRevertGenerator extends DefaultTestGenerator {
  getCalldata(): string {
    return this.ovmExecutionManager.interface.encodeFunctionData(
      'ovmSTATICCALL',
      [
        this.step.functionParams[0],
        this.step.functionParams[1],
        this.ovmRevertHelper.interface.encodeFunctionData('doRevert', [
          encodeRevertData(
            this.step.functionParams[2][0],
            this.step.functionParams[2][1],
            this.step.functionParams[2][2],
            this.step.functionParams[2][3]
          ),
        ]),
      ]
    )
  }

  getReturnData(): string {
    let expectedReturnData
    if (this.step.expectedReturnStatus) {
      expectedReturnData = this.ovmExecutionManager.interface.encodeFunctionResult(
        'ovmCALL',
        this.step.expectedReturnValues
      )
    } else {
      expectedReturnData = encodeRevertData(
        this.step.expectedReturnValues[0],
        this.step.expectedReturnValues[1],
        this.step.expectedReturnValues[2],
        this.step.expectedReturnValues[3]
      )
    }
    return expectedReturnData
  }

  interpretActualReturnData(data: string, success: boolean): string {
    if (success) {
      return (
        'ovmCALL to revert heler, which succeeded with return params:' +
        JSON.stringify(
          this.ovmExecutionManager.interface.decodeFunctionResult(
            'ovmCALL',
            data
          )
        )
      )
    } else {
      return (
        'ovmCALL to revert helper did itself revert with flag:' +
        JSON.stringify(decodeRevertData(data))
      )
    }
  }
}

class ovmCALLToInvalidGenerator extends DefaultTestGenerator {
  getCalldata(): string {
    return this.ovmExecutionManager.interface.encodeFunctionData('ovmCALL', [
      this.step.functionParams[0],
      this.step.functionParams[1],
      this.ovmInvalidHelper.interface.encodeFunctionData('doInvalid', []),
    ])
  }

  getReturnData(): string {
    let expectedReturnData
    if (this.step.expectedReturnStatus) {
      expectedReturnData = this.ovmExecutionManager.interface.encodeFunctionResult(
        'ovmCALL',
        this.step.expectedReturnValues
      )
    } else {
      expectedReturnData = encodeRevertData(
        this.step.expectedReturnValues[2][0],
        this.step.expectedReturnValues[2][1],
        this.step.expectedReturnValues[2][2],
        this.step.expectedReturnValues[2][3]
      )
    }
    return expectedReturnData
  }

  interpretActualReturnData(data: string, success: boolean): string {
    if (success) {
      return (
        'ovmCALL to InvalidJump/OutOfGas heler, which succeeded with return params:' +
        JSON.stringify(
          this.ovmExecutionManager.interface.decodeFunctionResult(
            'ovmCALL',
            data
          )
        )
      )
    } else {
      return (
        'ovmCALL to InvalidJump/OutOfGas helper did itself revert with flag:' +
        JSON.stringify(decodeRevertData(data))
      )
    }
  }
}

class ovmCREATEToInvalidGenerator extends DefaultTestGenerator {
  getCalldata(): string {
    return this.ovmExecutionManager.interface.encodeFunctionData('ovmCREATE', [
      getInitcode('Helper_CodeContractForInvalidInCreation'),
    ])
  }

  getReturnData(): string {
    let expectedEMResponse
    if (this.step.expectedReturnStatus) {
      expectedEMResponse = this.ovmExecutionManager.interface.encodeFunctionResult(
        'ovmCREATE',
        this.step.expectedReturnValues
      )
    } else {
      expectedEMResponse = encodeRevertData(
        this.step.expectedReturnValues[2][0],
        this.step.expectedReturnValues[2][1],
        this.step.expectedReturnValues[2][2],
        this.step.expectedReturnValues[2][3]
      )
    }
    return abi.encode(
      ['bytes', 'bytes'],
      [
        expectedEMResponse,
        this.ovmCreateStorer.interface.encodeFunctionResult(
          'getLastResponses',
          [[]]
        ),
      ]
    )
  }

  interpretActualReturnData(data: string, success: boolean): string {
    const EMResponse = abi.decode(['bytes', 'bytes'], data)[0]
    console.log(`EMResponse is ${EMResponse}, success is ${success}`)
    if (success) {
      return (
        'ovmCREATE to InvalidJump/OutOfGas IN CONSTRUCTOR heler, which succeeded returning address:' +
        JSON.stringify(
          this.ovmExecutionManager.interface.decodeFunctionResult(
            'ovmCREATE',
            EMResponse
          )
        )
      )
    } else {
      return (
        'ovmCALL to InvalidJump/OutOfGas IN CONSTRUCTOR heler did itself revert with flag:' +
        JSON.stringify(decodeRevertData(EMResponse))
      )
    }
  }
}

export const getTestGenerator = (
  step: TestStep,
  ovmExecutionManager: Contract,
  ovmCallHelper: Contract,
  ovmCreateStorer: Contract,
  ovmCreateHelper: Interface,
  ovmRevertHelper: Contract,
  ovmInvalidHelper: Contract
): TestCallGenerator => {
  switch (step.functionName) {
    case 'ovmCALL':
    case 'ovmDELEGATECALL':
    case 'ovmSTATICCALL':
      return new ovmCALLGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    case 'ovmCREATE':
      return new ovmCREATEGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    case 'ovmCALLToRevert':
      return new ovmCALLToRevertGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    case 'ovmCALLToInvalid':
      return new ovmCALLToInvalidGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    case 'ovmSTATICCALLToRevert':
      return new ovmSTATICCALLToRevertGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    case 'ovmCREATEToInvalid':
      return new ovmCREATEToInvalidGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
    default:
      return new DefaultTestGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        ovmCreateStorer,
        ovmCreateHelper,
        ovmRevertHelper,
        ovmInvalidHelper,
        step
      )
  }
}
