/* External Imports */
import { utils, ContractFactory } from 'ethers'

/* Internal Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'

const EMContract = new ContractFactory(
  ExecutionManager.abi,
  ExecutionManager.bytecode
)
const EMEvents = EMContract.interface.events
let topics = []
for (const eventKey of Object.keys(EMEvents)) {
  topics.push(EMEvents[eventKey].topic)
}

export const ALL_EXECUTION_MANAGER_EVENT_TOPICS = topics
