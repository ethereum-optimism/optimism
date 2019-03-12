/* External Imports */
import { Service } from '@nestd/core'
import debug, { Debugger } from 'debug'

@Service()
export class LoggerService {
  private loggers: { [key: string]: Debugger }

  /**
   * Logs a message to the console via debug.
   * @param namespace Namespace to log under.
   * @param message Message to log.
   */
  public log(namespace: string, message: any): void {
    if (!(namespace in this.loggers)) {
      this.loggers[namespace] = debug(namespace)
    }
    this.loggers[namespace](message)
  }

  /**
   * Logs an error the to console.
   * @param namespace Namespace to log under.
   * @param message Message to log.
   * @param trace Error stack trace.
   */
  public error(namespace: string, message: any, trace?: any): void {
    message += trace ? `\n{trace}\n` : ''
    this.log(namespace, `ERROR: ${message}`)
  }

  /**
   * Logs a warning to the console.
   * @param namespace Namespace to log under.
   * @param message Message to log.
   */
  public warn(namespace: string, message: any): void {
    this.log(namespace, `WARNING: ${message}`)
  }
}
