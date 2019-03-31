/**
 * EventBus broadcasts events to listener services.
 */
export interface EventBus {
  /**
   * Emits an event.
   * @param namespace to emit to.
   * @param event name to emit.
   * @param args to emit along with the event.
   */
  emit(namespace: string, event: string, ...args: any[]): Promise<void>
}
