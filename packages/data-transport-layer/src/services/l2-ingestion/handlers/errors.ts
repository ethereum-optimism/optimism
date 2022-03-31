export type EventName = 'SequencerTransaction'

export class MissingElementError extends Error {
  constructor(public name: EventName) {
    super(`missing event: ${name}`)
  }
}
