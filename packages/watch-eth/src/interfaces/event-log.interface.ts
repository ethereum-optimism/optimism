import {EventLogData} from "./event-log-data.interface";

export interface EventLog {
  data: EventLogData
  getHash(): string
}
