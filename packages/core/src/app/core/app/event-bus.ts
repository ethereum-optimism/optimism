import { EventBus } from '../../../interfaces'

export class DefaultEventBus implements EventBus {
  public async emit(
    namespace: string,
    event: string,
    ...args: any[]
  ): Promise<void> {}
}
