/**
 * Logger is used to generate logs.
 */
export interface Logger {
  readonly namespace: string

  /**
   * Logs a message.
   * @param message to log.
   */
  log(message: string): void

  /**
   * Logs an error.
   * @param message to log.
   * @param [trace] for the error.
   */
  error(message: string, trace?: string): void

  /**
   * Logs a warning.
   * @param message to log.
   */
  warn(message: string): void
}
