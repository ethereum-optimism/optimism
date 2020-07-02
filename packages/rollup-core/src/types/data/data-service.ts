import { L1DataService } from './l1-data-service'
import { L2DataService } from './l2-data-service'
import { VerifierDataService } from './verifier-data-service'

// Catch all data service for non-specialized Data Service needs.
export interface DataService
  extends L1DataService,
    L2DataService,
    VerifierDataService {}
