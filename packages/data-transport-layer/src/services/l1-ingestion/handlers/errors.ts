export type EventName = 'TransactionEnqueued' | 'SequencerBatchAppended'

export class MissingElementError extends Error {
  constructor(public name: EventName) {
    super(`missing event: ${name}`)
  }
}
