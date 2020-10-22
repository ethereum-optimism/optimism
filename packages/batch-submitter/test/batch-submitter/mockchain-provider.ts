import { JsonRpcProvider, TransactionResponse, Block } from '@ethersproject/providers'

interface L2Block extends Block {
    queueOrigin: string
    stateRoot: string
}

export class MockchainProvider extends JsonRpcProvider {
    mockBlockNumber: number = 0
    mockBlocks: L2Block[] = []

    async getBlockNumber(): Promise<number> {
        // Increment our mock block number
        if (this.mockBlockNumber < this.mockBlocks.length) {
            this.mockBlockNumber += 5
        }
        return this.mockBlockNumber
    }
}