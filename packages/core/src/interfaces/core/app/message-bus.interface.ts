/**
 * MessageBus broadcasts events to listener services.
 */
export interface MessageBus {
  /**
   * Emits an event.
   * @param event Event name to emit.
   * @param args Extra data to emit along with the event.
   */
  emit(event: string, ...args: any[]): void

  /**
   * Listens to an event.
   * @param event Event to listen to.
   * @param listener Function to call when the event is triggered.
   */
  on(event: string, listener: (...args: any[]) => any): void

  /**
   * Stops listening for an event.
   * @param event Event to stop listening for.
   * @param listener Function that was used to listen.
   */
  off(event: string, listener: (...args: any[]) => any): void
}
