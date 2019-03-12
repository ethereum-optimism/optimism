import {
  anyFunction,
  anyString,
  anything,
  instance,
  mock,
  when,
} from 'ts-mockito'

import { EventHandler } from '../../../src/services'

const mockEventHandler = mock(EventHandler)
const listeners: { [key: string]: Function } = {}
const mockEmitterOn = (event: string, listener: Function) => {
  listeners[event] = listener
  return mockEventHandler
}
const mockEmitterEmit = (event: string, ...args: object[]) => {
  listeners[event](...args)
  return true
}
when(mockEventHandler.on(anyString(), anyFunction())).thenCall(mockEmitterOn)
when(mockEventHandler.emit(anyString(), anything())).thenCall(mockEmitterEmit)
when(mockEventHandler.started).thenReturn(true)

const eventHandler = instance(mockEventHandler)

export { mockEventHandler, eventHandler }
