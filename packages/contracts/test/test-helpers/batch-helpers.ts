export function makeRepeatedBytes(value: string, length: number): string {
  const repeated = value.repeat((length * 2) / value.length + 1)
  return '0x' + repeated.slice(0, length * 2)
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
