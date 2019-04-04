import { should } from '../../helpers/setup'
import { DefaultEventDB } from '../../../src/event-db/default-event-db'

describe('DefaultEventDB', () => {
  let db: DefaultEventDB
  beforeEach(() => {
    db = new DefaultEventDB()
  })

  describe('setLastLoggedBlock', () => {
    it('should allow a user to set the last logged block', async () => {
      should.not.Throw(async () => {
        await db.setLastLoggedBlock('RealEvent', 999)
      })
    })
  })

  describe('getLastLoggedBlock', () => {
    it('should return -1 if the event does not exist', async () => {
      const result = await db.getLastLoggedBlock('FakeEvent')
      result.should.equal(-1)
    })

    it('should return the value if the event exists', async () => {
      await db.setLastLoggedBlock('RealEvent', 999)
      const result = await db.getLastLoggedBlock('RealEvent')
      result.should.equal(999)
    })
  })

  describe('setEventSeen', () => {
    it('should allow a user to mark an event as seen', async () => {
      should.not.Throw(async () => {
        await db.setEventSeen('SeenEvent')
      })
    })
  })

  describe('getEventSeen', () => {
    it('should return false if the event has not been seen', async () => {
      const result = await db.getEventSeen('UnseenEvent')
      result.should.be.false
    })

    it('should return true if the event has been seen', async () => {
      await db.setEventSeen('SeenEvent')
      const result = await db.getEventSeen('SeenEvent')
      result.should.be.true
    })
  })
})
