/**
 * Implements a time-based counter that can keep an accurate rolling count
 * of how many per X milliseconds.
 *
 * It does this by breaking X milliseconds into Y time-buckets and
 * 1) Incrementing the appropriate time bucket based on the current time
 * 2) Clearing time buckets that are more than X milliseconds old
 *
 * EXAMPLE:
 * Parameters: Time period is 1000 millis and we have 5 buckets.
 *
 * We start with [0,0,0,0,0]
 * Say we get 2 requests in the first 200 millis.
 * Our counter is now [2,0,0,0,0].
 *
 * Say we get our next request at the 450 millisecond mark.
 * Our counter is now [1,0,2,0,0]
 *
 * Say our next request is at the 850 millisecond mark
 * Our counter is now [1,0,1,0,2]
 *
 * If our next 4 requests come at the 1050 millisecond mark,
 * our initial bucket is now out of range, so it gets cleared and moved to the front
 * Our counter is now [4,1,0,1,0].
 *
 * This allows us to keep a moving counter of number per unit time. The more buckets, the more accurate.
 */
export class TimeBucketedCounter {
  private total: number
  private currentBucketStartTime: number
  private bucketIndex: number
  private readonly timeBuckets: number[]
  private readonly bucketDurationMillis: number

  constructor(
    private readonly periodInMillis: number,
    numBuckets: number = 10
  ) {
    this.total = 0
    this.bucketIndex = 0
    this.timeBuckets = new Array(numBuckets).fill(0)
    this.bucketDurationMillis = Math.floor(periodInMillis / numBuckets)
  }

  public increment(): number {
    this.cycleThroughBuffer()
    this.timeBuckets[this.bucketIndex]++
    this.total++
    return this.total
  }

  public getTotal(): number {
    this.cycleThroughBuffer()
    return this.total
  }

  /**
   * Cycles through the bucket buffer, purging values from timeBuckets that are older than our time window
   * and updating our index in our circular buffer accordingly.
   *
   * @param currentMillis The current time in millis, which combined with the last write time tells us how many
   *  time timeBuckets have gone out of range
   */
  private cycleThroughBuffer(
    currentMillis: number = new Date().getTime()
  ): void {
    if (!this.currentBucketStartTime) {
      this.currentBucketStartTime = currentMillis
      return
    }

    const bucketsToReset = Math.floor(
      (currentMillis - this.currentBucketStartTime) / this.bucketDurationMillis
    )
    if (bucketsToReset >= this.timeBuckets.length) {
      this.timeBuckets.fill(0)
      this.total = 0
      this.currentBucketStartTime = currentMillis
    } else if (bucketsToReset > 0) {
      for (let i = 0; i < bucketsToReset; i++) {
        this.bucketIndex = ++this.bucketIndex % this.timeBuckets.length
        this.total -= this.timeBuckets[this.bucketIndex]
        this.timeBuckets[this.bucketIndex] = 0
      }
      this.currentBucketStartTime += bucketsToReset * this.bucketDurationMillis
    }
  }
}
