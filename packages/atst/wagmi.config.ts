import { defineConfig } from '@wagmi/cli'
import { hardhat, react } from '@wagmi/cli/plugins'
import * as chains from 'wagmi/chains'

export const ATTESTATION_STATION_ADDRESS =
  '0xEE36eaaD94d1Cc1d0eccaDb55C38bFfB6Be06C77'

export default defineConfig({
  out: 'src/react.ts',
  plugins: [
    hardhat({
      project: '../contracts-periphery',
      include: ['AttestationStation.json'],
      deployments: {
        AttestationStation: {
          [chains.optimism.id]: ATTESTATION_STATION_ADDRESS,
          [chains.optimismGoerli.id]: ATTESTATION_STATION_ADDRESS,
          [chains.foundry.id]: ATTESTATION_STATION_ADDRESS,
        },
      },
    }),
    react(),
  ],
})
