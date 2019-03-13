/* External Imports */
import { Module } from '@nestd/core'

/* Services */
import { ProofVerificationService } from './proof-verification.service'

@Module({
  services: [ProofVerificationService],
})
export class ProofModule {}
