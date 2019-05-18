import { RuntimeException } from './runtime.exception'
import { SERVER_ALREADY_STARTED_MESSAGE } from './messages'

export class ServerAlreadyStartedException extends RuntimeException {
  constructor() {
    super(SERVER_ALREADY_STARTED_MESSAGE)
  }
}
