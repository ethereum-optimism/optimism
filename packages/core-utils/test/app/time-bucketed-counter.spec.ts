import { sleep, TimeBucketedCounter } from '../../src/app'

describe('Time Bucketed Counter', () => {
  let counter: TimeBucketedCounter

  beforeEach(() => {
    counter = new TimeBucketedCounter(2000, 5)
  })

  it('accurately increments within time period, no sleeping', () => {
    let res: number = counter.increment()
    res.should.equal(1, 'Incorrect counter after increment 1')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    res = counter.increment()
    res.should.equal(2, 'Incorrect counter after increment 2')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    res = counter.increment()
    res.should.equal(3, 'Incorrect counter after increment 3')
    res.should.equal(counter.getTotal(), 'Incorrect total!')
  })

  it('accurately increments within time period', async () => {
    let res: number = counter.increment()
    res.should.equal(1, 'Incorrect counter after increment 1')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(400)
    res = counter.increment()
    res.should.equal(2, 'Incorrect counter after increment 2')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(800)
    res = counter.increment()
    res.should.equal(3, 'Incorrect counter after increment 3')
    res.should.equal(counter.getTotal(), 'Incorrect total!')
  })

  it('accurately increments within time period', async () => {
    let res: number = counter.increment()
    res.should.equal(1, 'Incorrect counter after increment 1')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(400)
    res = counter.increment()
    res.should.equal(2, 'Incorrect counter after increment 2')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(1200)
    res = counter.increment()
    res.should.equal(3, 'Incorrect counter after increment 3')
    res.should.equal(counter.getTotal(), 'Incorrect total!')
  })

  it('cycles out old counts, one increment', async () => {
    let res = counter.increment()
    res.should.equal(1, 'Incorrect counter after increment 1')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(2000)
    res = counter.getTotal()
    res.should.equal(0, 'Incorrect count!')
  })

  it('accurately cycles out old counts, multiple increments', async () => {
    counter.increment()
    counter.increment()
    let res: number = counter.increment()
    res.should.equal(3, 'Incorrect counter after increment 1')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(400)
    res = counter.increment()
    res.should.equal(4, 'Incorrect counter after increment 2')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(1200)
    res = counter.increment()
    res.should.equal(5, 'Incorrect counter after increment 3')
    res.should.equal(counter.getTotal(), 'Incorrect total!')

    await sleep(401)
    counter
      .getTotal()
      .should.equal(2, 'First 3 increments did not cycle out properly!')
    res = counter.increment()
    res.should.equal(3, 'Incorrect counter after increment 3')
    res.should.equal(counter.getTotal(), 'Incorrect total!')
  })

  it('accurately cycles out old counts, over time', async () => {
    for (let totalIncrements = 0; totalIncrements < 15; totalIncrements++) {
      const res = counter.increment()
      const expected = totalIncrements >= 3 ? 4 : totalIncrements + 1
      res.should.equal(
        expected,
        `Incorrect increment result! Total increments: ${totalIncrements}`
      )
      await sleep(500)
    }
  }).timeout(10_000)
})
