/* External Imports */
import { ethers } from 'ethers'

/* Contract Imports */

import * as ExecutionManager from '../build/ExecutionManager.json'
import * as L2ExecutionManager from '../build/L2ExecutionManager.json'
import * as ContractAddressGenerator from '../build/ContractAddressGenerator.json'
import * as L2ToL1MessageReceiver from '../build/L2ToL1MessageReceiver.json'
import * as L2ToL1MessagePasser from '../build/L2ToL1MessagePasser.json'
import * as L1ToL2TransactionPasser from '../build/L1ToL2TransactionPasser.json'
import * as RLPEncode from '../build/RLPEncode.json'
import * as SafetyChecker from '../build/SafetyChecker.json'

// Contract Exports
export const ExecutionManagerContractDefinition = ExecutionManager
export const L2ExecutionManagerContractDefinition = L2ExecutionManager
export const ContractAddressGeneratorContractDefinition = ContractAddressGenerator
export const L2ToL1MessageReceiverContractDefinition = L2ToL1MessageReceiver
export const L2ToL1MessagePasserContractDefinition = L2ToL1MessagePasser
export const L1ToL2TransactionPasserContractDefinition = L1ToL2TransactionPasser
export const RLPEncodeContractDefinition = RLPEncode
export const SafetyCheckerContractDefinition = SafetyChecker

export const executionManagerInterface = new ethers.utils.Interface(
  ExecutionManager.interface
)

export const l2ExecutionManagerInterface = new ethers.utils.Interface(
  L2ExecutionManager.interface
)
export const l2ToL1MessagePasserInterface = new ethers.utils.Interface(
  L2ToL1MessagePasser.interface
)
