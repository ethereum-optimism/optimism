/* External Imports */
import { Contract } from 'ethers'

export interface MockContractFunction {
  functionName: string
  inputTypes?: string[]
  outputTypes?: string[]
  returnValues?: any[] | ((...params: any[]) => any[] | Promise<any>)
}

export interface MockContract extends Contract {
  getCallCount: (functionName: string) => number
  getCallData: (functionName: string, callIndex: number) => any[]
  setReturnValues: (
    functionName: string,
    returnValues: any[] | ((...params: any[]) => any[])
  ) => void
  __fns: {
    [sighash: string]: MockContractFunction
  }
}
