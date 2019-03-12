/* tslint:disable:no-any */
import { instance, mock, when } from 'ts-mockito'

import { PlasmaApp } from '../../src/plasma'

const createApp = (services: any = {}) => {
  const mockApp = mock(PlasmaApp)
  when(mockApp.services).thenReturn(services)
  const app = instance(mockApp)
  return { app, mockApp }
}

export { createApp }
