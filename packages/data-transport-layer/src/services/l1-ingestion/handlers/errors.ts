export type EventName =
  | 'TransactionEnqueued'
  | 'SequencerBatchAppended'
  | 'StateBatchAppended'

export class MissingElementError extends Error {
  constructor(public name: EventName) {
    super(`missing event: ${name}`)
  }
}
