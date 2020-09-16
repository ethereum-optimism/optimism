import * as path from 'path'

import { ethers } from "@nomiclabs/buidler"
import { ContractFactory } from 'ethers'
import { Interface } from 'ethers/lib/utils'

const getContractDefinition = (name: string): any => {
    return require(path.join(__dirname, '../../../artifacts', `${name}.json`))
}
  
export const getContractInterface = (name: string): Interface => {
    const definition = getContractDefinition(name)
    return new ethers.utils.Interface(definition.abi)
}

// const ExecutionManager: ContractFactory = await ethers.getContractFactory('iOVM_ExecutionManager')
export const iEM = getContractInterface('iOVM_ExecutionManager')

// const CodeContract: ContractFactory = await ethers.getContractFactory('Helper_CodeContractForCalls')
const iCC = getContractInterface('Helper_CodeContractForCalls')

export interface TestCallGenerator {
  generateCalldata(): string
  generateExpectedReturnData(): string
  parseActualReturnData(returned: string): any
}


export abstract class baseOVMCallTest implements TestCallGenerator {
  executionManagerMethodName: string = 'EM METHOD NAME NOT SET'
  arguments: any[] = []
  expectedReturnValues: any[] = []

  generateCalldata(): string {
      return iEM.encodeFunctionData(
          this.executionManagerMethodName,
          this.arguments
      )
  }

  generateExpectedReturnData(): string {
      return iEM.encodeFunctionResult(
          this.executionManagerMethodName,
          this.expectedReturnValues
      )
  }

  parseActualReturnData(returned: string) {
      const decodedResult = iEM.decodeFunctionResult(
          this.executionManagerMethodName,
          returned
      )
      return 'call to ExeMgr.' + this.executionManagerMethodName + ' returned: \n' + decodedResult
  }
}

export class ovmADDRESSTest extends baseOVMCallTest {
  constructor(
      expectedAddress: string
  ) {
      super()
      this.executionManagerMethodName = 'ovmADDRESS'
      this.expectedReturnValues = [expectedAddress]
  }
}

export class ovmCALLERTest extends baseOVMCallTest {
  constructor(
      expectedMsgSender: string
  ) {
      super()
      this.executionManagerMethodName = 'ovmCALLER'
      this.expectedReturnValues = [expectedMsgSender]
  }
}

const DEFAULT_GAS_LIMIT = 1_000_000

export class ovmCALLTest extends baseOVMCallTest {
  constructor(
      callee: string,
      calleeTests: Array<TestCallGenerator>,
      shouldCalleeSucceed: boolean = true,
      gasLimit: number = DEFAULT_GAS_LIMIT
  ) {
      super()
      this.executionManagerMethodName = 'ovmCALL'
      this.arguments = [
          gasLimit,
          callee,
          iCC.encodeFunctionData(
              'runSteps',
              [{
                  callsToEM: calleeTests.map((testGenerator) => {
                      return testGenerator.generateCalldata()
                  }),
                  shouldRevert: !shouldCalleeSucceed
              }]
          )
      ]
      this.expectedReturnValues = [
          shouldCalleeSucceed,
          iCC.encodeFunctionResult(
              'runSteps',
              [calleeTests.map((testGenerator) => {
                  return {
                      success: true, //TODO: figure out if we need this not to happen
                      data: testGenerator.generateExpectedReturnData()
                  }
              })]
          )
      ]
  }
}
