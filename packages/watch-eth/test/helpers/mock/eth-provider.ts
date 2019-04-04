import { instance, mock, when, anything } from 'ts-mockito'
import { DefaultEthProvider } from '../../../src/eth-provider/default-eth-provider'
import { EventLog } from '../../../src/models'

const mockEthProvider = mock(DefaultEthProvider)

when(mockEthProvider.connected()).thenCall(() => {
  return true
})

let block = 1
when(mockEthProvider.getCurrentBlock()).thenCall(() => {
  return block
})
export const setBlock = (newBlock: number) => {
  block = newBlock
}

let events: EventLog[] = []
when(mockEthProvider.getEvents(anything())).thenCall(() => {
  return events
})
export const setEvents = (newEvents: EventLog[]) => {
  events = newEvents
}

export const reset = () => {
  block = 1
  events = []
}

export const eth = instance(mockEthProvider)
