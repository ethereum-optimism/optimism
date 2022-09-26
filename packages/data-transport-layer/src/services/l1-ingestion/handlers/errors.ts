export type EventName =
  | 'TransactionEnqueued'
  | 'SequencerBatchAppended'
  | 'StateBatchAppended'
  | 'SequencerBatchAppendedTransaction'

export class MissingElementError extends Error {
  constructor(public name: EventName) {
    super(`missing event: ${name}`)
  }
}
