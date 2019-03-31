export interface EventFilterOptions {
  event: string
  indexed?: { [key: string]: any }
}

export interface EventWatcherOptions {
  address: string
  abi: any
  finalityDepth?: number
  pollInterval?: number
}

export interface EventWatcher {
  subscribe(
    options: EventFilterOptions | string,
    listener: (...args: any) => any
  ): void
  unsubscribe(
    options: EventFilterOptions | string,
    listener: (...args: any) => any
  ): void
}
