/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { EventHandlerService } from './event-handler.service'
import { EventWatcherService } from './event-watcher.service'

@Module({
  services: [EventHandlerService, EventWatcherService],
})
export class EventModule {}
