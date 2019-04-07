export interface BaseEventDB {
  getLastLoggedBlock(event: string): Promise<number>
  setLastLoggedBlock(event: string, block: number): Promise<void>
  getEventSeen(event: string): Promise<boolean>
  setEventSeen(event: string): Promise<void>
}
