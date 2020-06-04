/**********************************
 * Byte String Generation Helpers *
 *********************************/

// Create a byte string of some length in bytes. It repeats the value provided until the
// string hits that length
export function makeRepeatedBytes(value: string, length: number): string {
  const repeated = value.repeat((length * 2) / value.length + 1)
  const sliced = repeated.slice(0, length * 2)
  return '0x' + sliced
}

export function makeRandomBlockOfSize(blockSize: number): string[] {
  const block = []
  for (let i = 0; i < blockSize; i++) {
    block.push(makeRepeatedBytes('' + Math.floor(Math.random() * 500 + 1), 32))
  }
  return block
}

export function makeRandomBatchOfSize(batchSize: number): string[] {
  return makeRandomBlockOfSize(batchSize)
}
