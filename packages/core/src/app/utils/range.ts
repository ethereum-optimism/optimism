/* Internal Imports */
import BigNum = require('bn.js')
import { bnMax, bnMin } from './misc'
import { BlockRange, Range } from 'src/interfaces'
import { bufferUtils } from 'src/app'

/**
 * Checks if two ranges intersect, eg. [1,10) & [8,11) would return true.
 * @param start1 The start of the first range.
 * @param end1 The end of the first range.
 * @param start2 The start of the second range.
 * @param end2 The end of the second range.
 * @returns true if max(start1, start2) < min(end1, end2), false otherwise.
 */
export const intersects = (
  start1: Buffer,
  end1: Buffer,
  start2: Buffer,
  end2: Buffer
): boolean => {
  const maxStart = bufferUtils.max(start1, start2)
  const minEnd = bufferUtils.min(end1, end2)
  return bufferUtils.lt(maxStart, minEnd)
}

/**
 * Gets the intersection of two Ranges, eg. [1,10) & [8,11) would return [8,10).
 * @param range1 the first range
 * @param range2 the second range
 * @returns true if ranges overlap, false otherwise
 */
export const getOverlappingRange = (
  range1: Range,
  range2: Range
): Range | undefined => {
  const start: BigNum = bnMax(range1.start, range2.start)
  const end: BigNum = bnMin(range1.end, range2.end)

  return start.lt(end) ? { start, end } : undefined
}

/**
 * Checks if two ranges intersect, eg. [1,10) & [8,11) would return true.
 * @param range1 the first range
 * @param range2 the second range
 * @returns true if ranges overlap, false otherwise
 */
export const rangesIntersect = (range1: Range, range2: Range): boolean => {
  return !!getOverlappingRange(range1, range2)
}

/**
 * RangeStore makes it easy to store ranges.
 * When ranges are added, only the sections with
 * a higher block number than existing ranges
 * that they overlap with will be inserted.
 */
export class DefaultBlockRange<T extends BlockRange> {
  public ranges: T[]

  /**
   * Creates the store.
   * @param ranges Initial ranges to store.
   */
  constructor(ranges: T[] = []) {
    this.ranges = ranges
  }

  /**
   * Merges the ranges of another RangeStore into this one.
   * @param other The other RangeStore.
   */
  public merge(other: DefaultBlockRange<T>): void {
    for (const range of other.ranges) {
      this.addRange(range)
    }
  }

  /**
   * Returns the sections of existing ranges
   * that overlap with the given range.
   * @param range Range to overlap with.
   * @returns a list of existing ranges.
   */
  public getOverlapping(range: Range): T[] {
    return this.ranges.reduce((overlap, existing) => {
      const overlapping: Range | undefined = getOverlappingRange(
        existing,
        range
      )
      if (!!overlapping) {
        overlap.push({
          ...existing,
          start: overlapping.start,
          end: overlapping.end,
        })
      }
      return overlap
    }, [])
  }

  /**
   * Adds a range to the range store.
   * Will pieces of the range with a higher
   * block number than the existing ranges
   * they overlap with.
   * @param range Range to add.
   */
  public addRange(range: T): void {
    if (range.start.gte(range.end)) {
      throw new Error('Invalid range')
    }

    const toAdd = new DefaultBlockRange([range])
    for (const overlap of this.getOverlapping(range)) {
      if (overlap.block.gt(range.block)) {
        // Existing range has a greater block number,
        // don't add this part of the new range.
        toAdd.removeRange(overlap)
      } else {
        // New range has a greater block number,
        // remove this part of the old range.
        this.removeRange(overlap)
      }
    }

    this.ranges = this.ranges.concat(toAdd.ranges)
    this.sortRanges()
  }

  /**
   * Removes a range from the store.
   * @param range Range to remove.
   */
  public removeRange(range: Range): void {
    for (const overlap of this.getOverlapping(range)) {
      // Remove the old range entirely.
      let removed: T
      this.ranges = this.ranges.filter((r) => {
        if (r.start.lte(overlap.start) && r.end.gte(overlap.end)) {
          removed = r
          return false
        }
        return true
      })

      // Add back any of the left or right
      // portions of the old snapshot that didn't
      // overlap with the piece being removed.
      // For visual intuition:
      //
      // [-----------]   old snapshot
      //     [---]       removed range
      // |xxx|           left remainder
      //         |xxx|   right remainder

      // Add left remainder.
      if (removed.start.lt(overlap.start)) {
        this.ranges.push({
          ...removed,
          end: overlap.start,
        })
      }

      // Add right remainder.
      if (removed.end.gt(overlap.end)) {
        this.ranges.push({
          ...removed,
          start: overlap.end,
        })
      }
    }

    this.sortRanges()
  }

  /**
   * Increments the block number of any parts of ranges
   * that intersect with the given range.
   * @param range Range to increment.
   */
  public incrementBlocks(range: Range): void {
    if (range.start.gte(range.end)) {
      throw new Error('Invalid range')
    }

    for (const existing of this.ranges) {
      const overlap: Range = {
        start: bnMax(existing.start, range.start),
        end: bnMin(existing.end, range.end),
      }

      // No overlap, can skip.
      if (overlap.start.gte(overlap.end)) {
        continue
      }

      this.addRange({
        ...existing,
        ...overlap,
        ...{
          block: existing.block.addn(1),
        },
      })
    }
  }

  /**
   * Sorts ranges by start.
   */
  private sortRanges(): void {
    this.ranges = this.ranges.sort((a, b) => {
      return a.start.sub(b.start).toNumber()
    })
  }
}
