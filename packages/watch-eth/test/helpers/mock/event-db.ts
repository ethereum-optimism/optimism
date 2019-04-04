import { instance, mock, spy } from 'ts-mockito'
import { DefaultEventDB } from '../../../src/event-db/default-event-db'

const mockEventDB = mock(DefaultEventDB)

export const db = instance(mockEventDB)
export const dbSpy = spy(db)
