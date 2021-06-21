import * as compose from 'docker-compose'
import * as shell from 'shelljs'
import * as path from 'path'

type ServiceNames =
  | 'batch_submitter'
  | 'dtl'
  | 'l2geth'
  | 'relayer'
  | 'verifier'
  | 'replica'

const OPS_DIRECTORY = path.join(process.cwd(), '../ops')
const DEFAULT_SERVICES: ServiceNames[] = [
  'batch_submitter',
  'dtl',
  'l2geth',
  'relayer',
]

export class DockerComposeNetwork {
  constructor(private readonly services: ServiceNames[] = DEFAULT_SERVICES) {}

  async up(options?: compose.IDockerComposeOptions) {
    const out = await compose.upMany(this.services, {
      cwd: OPS_DIRECTORY,
      ...options,
    })

    const { err, exitCode } = out

    if (!err || exitCode) {
      console.error(err)
      throw new Error(
        'Unexpected error when starting docker-compose network, dumping output'
      )
    }

    if (err.includes('Creating')) {
      console.info(
        '🐳 Tests required starting containers. Waiting for sequencer to ready.'
      )
      shell.exec(`${OPS_DIRECTORY}/scripts/wait-for-sequencer.sh`, {
        cwd: OPS_DIRECTORY,
      })
    }

    return out
  }

  async logs() {
    return compose.logs(this.services, { cwd: OPS_DIRECTORY })
  }

  async stop(service: ServiceNames) {
    return compose.stopOne(service, { cwd: OPS_DIRECTORY })
  }

  async rm() {
    return compose.rm({ cwd: OPS_DIRECTORY })
  }
}
