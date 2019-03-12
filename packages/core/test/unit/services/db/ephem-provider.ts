import '../../../setup'

/* Internal Imports */
import { EphemDBProvider } from '../../../../src/services/db/backends/ephem-provider'

describe('EphemDBProvider', async () => {
  const db = new EphemDBProvider()

  it('should add a new item to the database', async () => {
    const expected = 'value'
    await db.set('key', expected)
    const value = await db.get('key')

    value.should.equal(expected)
  })

  it('should remove an item from the database', async () => {
    const expected = 'value'
    await db.set('key', expected)
    await db.delete('key')

    await db.get('key').should.be.rejectedWith('Key not found in database')
  })

  it('should check if an item exists', async () => {
    const expected = 'value'
    await db.set('key', expected)
    const exists = await db.exists('key')

    exists.should.be.true
  })

  it('should have a key not exist if it was removed', async () => {
    const expected = 'value'
    await db.set('key', expected)
    await db.delete('key')
    const exists = await db.exists('key')

    exists.should.be.false
  })
})
