import 'dotenv/config'

import { execSync } from 'node:child_process'
import * as path from 'node:path'

import { PonderInteropClient } from '@eth-optimism/ponder-interop/client'
import { App } from '@eth-optimism/utils-app'
import { Option } from 'commander'
import type { Hono } from 'hono'
import type { Logger } from 'pino'
import type { Address, Hex, PublicClient, WalletClient } from 'viem'
import {
  createPublicClient,
  createWalletClient,
  getAddress,
  http,
  isHex,
} from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { z } from 'zod'

import { attachBudgetRoute } from '@/admin/budget.js'
import { attachRelayerBalanceRoute } from '@/admin/relayerBalance.js'
import { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'
import { RelayerMetrics } from '@/metrics.js'
import { RelayFailureRegistry } from '@/relay/relayFailureRegistry.js'
import type { RelayerConfig } from '@/relayer.js'
import { Relayer } from '@/relayer.js'
import { ClientManager } from '@/services/clientManager.js'
import { CallbackShareModule } from '@/strategies/callbackShareModule.js'
import { EthBridgeModule } from '@/strategies/ethBridgeModule.js'
import { GeneralRelayModule } from '@/strategies/generalRelayModule.js'
import { PromiseModule } from '@/strategies/promiseModule.js'
import type { RelayModule, RunContext } from '@/strategies/types.js'

// `import attributes` (the `with { type: 'json' }` form) trip prettier's
// parser when it appears in a sorted import group; pinning this to the top
// in its own opt-out region keeps both prettier and simple-import-sort happy.
import packageFile from '../package.json' with { type: 'json' };

const ChainSchema = z.array(
  z.object({
    id: z.number(),
    name: z.string(),
    url: z.string(),
  }),
)

const MODULE_NAMES = [
  'eth-bridge',
  'general-relay',
  'promise',
  'callback-share',
] as const

/**
 * All four modules are on by default (R1): fresh installs must relay
 * promises and callbacks without extra configuration — the 6/08 partner-demo
 * stall was a fresh orchestrator install that silently relayed nothing but
 * eth-bridge messages. Deployments that want less set ENABLED_MODULES
 * explicitly. Referenced by BOTH declaration sites (ConfigSchema and the
 * commander Option) so they can't drift apart again.
 */
const DEFAULT_ENABLED_MODULES = [...MODULE_NAMES]

export const ConfigSchema = z.object({
  loopIntervalMs: z.coerce.number().min(2000),
  ponderInteropApi: z.string().url(),
  sponsoredEndpoint: z.string().url(),
  chainEndpointOverrides: z.array(z.string().url()).optional(),
  senderPrivateKeys: z.array(
    z.string().refine((key) => {
      if (!key) return true
      return isHex(key) && privateKeyToAccount(key) !== undefined
    }, 'private key must be a valid hex string'),
  ),
  gasTankAddress: z.custom<Address>().optional(),
  promiseAddress: z.custom<Address>().optional(),
  enabledModules: z
    .array(z.enum(MODULE_NAMES))
    .min(1)
    .default(DEFAULT_ENABLED_MODULES),
})

type Chains = z.infer<typeof ChainSchema>
type Config = z.infer<typeof ConfigSchema>

interface BuildInfo {
  version: string
  sha: string
  dirty: string
  node: string
}

/**
 * Resolves build metadata for the `relayer_build_info` gauge.
 *
 * Env vars take precedence (so container images can bake sha/dirty at build
 * time without shipping a .git directory). Falls back to `git` for local dev.
 * Any value that can't be determined is reported as 'unknown' rather than
 * throwing — build_info must never block startup.
 */
function detectBuildInfo(): BuildInfo {
  const version = packageFile.version ?? 'unknown'
  const node = process.version

  let sha = process.env.GIT_SHA?.trim() || ''
  if (!sha) {
    try {
      sha = execSync('git rev-parse HEAD', {
        stdio: ['ignore', 'pipe', 'ignore'],
      })
        .toString()
        .trim()
    } catch {
      sha = 'unknown'
    }
  }

  let dirty = process.env.GIT_DIRTY?.trim() || ''
  if (!dirty) {
    try {
      execSync('git diff --quiet', { stdio: 'ignore' })
      dirty = 'false'
    } catch {
      // `git diff --quiet` exits non-zero when the tree is dirty OR when git
      // itself isn't available. Disambiguate by re-checking whether git works.
      try {
        execSync('git rev-parse HEAD', { stdio: 'ignore' })
        dirty = 'true'
      } catch {
        dirty = 'unknown'
      }
    }
  }

  return { version, sha, dirty, node }
}

/**
 * Optional positive-integer env override; undefined (use the code default)
 * when unset or unparsable. Used for the Ponder pagination caps —
 * PONDER_PAGE_LIMIT, PONDER_MAX_PER_CYCLE, PONDER_MAX_SCAN_PER_CYCLE.
 */
function parsePositiveIntEnv(name: string): number | undefined {
  const raw = process.env[name]
  if (!raw) return undefined
  const n = parseInt(raw, 10)
  return Number.isFinite(n) && n > 0 ? n : undefined
}

// Parse private keys from .env, supporting both singular and comma-separated plural.
function parseSenderPrivateKeys(senderPrivateKey?: string): string[] {
  const keys = (process.env.SENDER_PRIVATE_KEYS || '')
    .split(',')
    .map((k) => k.trim())
    .filter(Boolean)

  if (keys.length) return keys

  const singleKey = senderPrivateKey || process.env.SENDER_PRIVATE_KEY
  return singleKey ? [singleKey] : []
}

class RelayerApp extends App {
  private relayer!: Relayer
  // Stored on the app so the admin-API override (called by App.run after
  // preMain returns) can read them. preMain is the only writer.
  private relayFunds!: RelayFundsDeposited
  private ponderClient!: PonderInteropClient
  private clientManager!: ClientManager
  private relayerChains!: Chains
  private relayerMode!: 'local' | 'sponsored'

  constructor() {
    super({
      name: packageFile.name,
      version: packageFile.version,
      description: packageFile.description,
    })
  }

  /**
   * Creates a Relayer instance with the given configuration
   * @param log - The logger to use for the relayer
   * @param relayerConfig - The configuration for the relayer
   * @param modules - The relay modules to register
   * @returns A configured Relayer instance
   */
  protected createRelayer(
    log: Logger,
    relayerConfig: RelayerConfig,
    modules: RelayModule[],
  ): Relayer {
    return new Relayer(log, relayerConfig, modules)
  }

  protected additionalOptions(): Option[] {
    return [
      new Option('--loop-interval-ms <ms>', 'interval to run the relayer')
        .default(2000)
        .env('LOOP_INTERVAL_MS'),
      new Option('--ponder-interop-api <url>', 'url to the interop ponder api')
        .default('http://127.0.0.1:42069')
        .env('PONDER_INTEROP_API_URL'),
      new Option(
        '--sponsored-endpoint <url>',
        'sponsored endpoint to use for the relayer',
      )
        .default('http://127.0.0.1:3000')
        .env('SPONSORED_ENDPOINT_URL'),
      new Option(
        '--chain-endpoint-overrides <endpoints>',
        'comma separated list of chain rpc urls',
      )
        .env('CHAIN_ENDPOINT_OVERRIDES')
        .argParser((val) => val.split(',')),
      new Option(
        '--sender-private-key <key>',
        'local private key to use for the relayer',
      )
        .conflicts('sponsoredEndpoint')
        .env('SENDER_PRIVATE_KEY'),
      new Option(
        '--gas-tank-address <address>',
        'address of the gas tank to use for the relayer',
      )
        .env('GAS_TANK_ADDRESS')
        .argParser((val) => getAddress(val)),
      new Option(
        '--promise-address <address>',
        'address of the Promise contract (required for the callback-share module)',
      )
        // Env name matches the orchestrator's render.mjs, which already writes
        // PROMISE_CONTRACT_ADDRESS into the autorelayer .env.
        .env('PROMISE_CONTRACT_ADDRESS')
        .argParser((val) => getAddress(val)),
      new Option(
        '--enabled-modules <modules>',
        `comma separated list of modules to enable (${MODULE_NAMES.join(
          ', ',
        )})`,
      )
        .default(DEFAULT_ENABLED_MODULES)
        .env('ENABLED_MODULES')
        .argParser((val) => val.split(',')),
    ]
  }

  protected async preMain(): Promise<void> {
    const config = this.parseConfig()
    const ponderClient = new PonderInteropClient(config.ponderInteropApi)
    const chains = await this.fetchChains(config, ponderClient)
    const clients = this.buildClients(chains, config)
    const relayFunds = new RelayFundsDeposited(
      process.env.RELAY_FUNDS_DB_PATH ??
        path.join(process.cwd(), '.relay-funds.sqlite'),
    )
    const failureRegistry = new RelayFailureRegistry(
      process.env.RELAY_FAILURE_REGISTRY_DB_PATH ??
        path.join(process.cwd(), '.relay-failure-registry.sqlite'),
    )
    const modules = this.buildModules(config, relayFunds, failureRegistry)
    this.relayer = this.wireRelayer(
      config,
      ponderClient,
      clients,
      modules,
      failureRegistry,
    )
    this.relayFunds = relayFunds
    this.ponderClient = ponderClient
    this.clientManager = clients
    this.relayerChains = chains
    this.relayerMode = config.senderPrivateKeys.length > 0 ? 'local' : 'sponsored'
  }

  protected initializeAdminApi(): Hono {
    const adminApi = super.initializeAdminApi()
    attachBudgetRoute(adminApi, {
      relayFunds: this.relayFunds,
      ponderClient: this.ponderClient,
      logger: this.logger,
    })
    attachRelayerBalanceRoute(adminApi, {
      clients: this.clientManager,
      chains: this.relayerChains
        .map((c) => ({ id: c.id, name: c.name }))
        .sort((a, b) => a.id - b.id),
      mode: this.relayerMode,
      logger: this.logger,
    })
    return adminApi
  }

  private parseConfig(): Config {
    const senderPrivateKeys = parseSenderPrivateKeys(
      this.options.senderPrivateKey,
    )
    const { data, error } = ConfigSchema.safeParse({
      ...this.options,
      senderPrivateKeys,
    })
    if (error) throw new Error(`invalid configuration: ${error}`)
    return data
  }

  private async fetchChains(
    config: Config,
    ponderClient: PonderInteropClient,
  ): Promise<Chains> {
    this.logger.debug('fetching chain config from %s', config.ponderInteropApi)
    let chains: Chains
    try {
      chains = await ponderClient.getChains()
      if (chains.length === 0) throw new Error('no chains found')
    } catch (error) {
      throw new Error(`failed to fetch chains: ${error}`)
    }

    const overrides = config.chainEndpointOverrides ?? []
    const overriddenIds = new Set<number>()
    for (const url of overrides) {
      try {
        const client = createPublicClient({ transport: http(url) })
        const chainId = await client.getChainId()
        const chain = chains.find((c) => c.id === chainId)
        if (chain) {
          chain.url = url
          overriddenIds.add(chainId)
        }
      } catch (error) {
        throw new Error(`failed to configure chain endpoint override: ${error}`)
      }
    }

    // When explicit RPC overrides are given, scope the relayer to exactly those
    // chains. Ponder's /chains returns every known chain (60+ public networks);
    // building clients for all of them makes the per-cycle EOA-balance snapshot
    // block for ~40s on unreachable RPCs. With overrides set we know precisely
    // which chains this deployment targets. No overrides → keep all chains.
    if (overrides.length > 0) {
      const scoped = chains.filter((c) => overriddenIds.has(c.id))
      this.logger.info(
        { chains: scoped.map((c) => c.id) },
        'scoped to chains with RPC overrides',
      )
      return scoped
    }
    return chains
  }

  private buildClients(chains: Chains, config: Config): ClientManager {
    const clients: Record<number, PublicClient> = {}
    const walletClients: Record<number, WalletClient[]> = {}
    for (const chain of chains) {
      this.logger.debug('configuring clients for chain %d', chain.id)
      const transport = http(chain.url)
      clients[chain.id] = createPublicClient({ transport })

      if (config.senderPrivateKeys.length > 0) {
        const accounts = config.senderPrivateKeys.map((k) =>
          privateKeyToAccount(k as Hex),
        )
        walletClients[chain.id] = accounts.map((account) =>
          createWalletClient({ account, transport }),
        )
        this.logger.debug(
          { senders: accounts.map((a) => a.address) },
          'configured local senders',
        )
      } else {
        const url = `${config.sponsoredEndpoint}/${chain.id}`
        walletClients[chain.id] = [createWalletClient({ transport: http(url) })]
        this.logger.debug('configured sponsored endpoint %s', url)
      }
    }
    return new ClientManager(clients, walletClients)
  }

  /**
   * Whether ENABLED_MODULES came from the operator (env var or CLI flag)
   * rather than the code default. Determines error strictness for module
   * prerequisites: explicit configuration fails loudly, defaults degrade
   * gracefully.
   */
  private modulesExplicitlyConfigured(): boolean {
    return (
      process.env.ENABLED_MODULES !== undefined ||
      process.argv.includes('--enabled-modules')
    )
  }

  private buildModules(
    config: Config,
    relayFunds: RelayFundsDeposited,
    failureRegistry: RelayFailureRegistry,
  ): RelayModule[] {
    const enabled = new Set(config.enabledModules)
    const modules: RelayModule[] = []
    if (enabled.has('eth-bridge'))
      modules.push(new EthBridgeModule(relayFunds, failureRegistry))

    if (enabled.has('promise'))
      modules.push(new PromiseModule(failureRegistry))

    if (enabled.has('callback-share')) {
      if (!config.promiseAddress) {
        // callback-share is enabled by default (R1) but needs the Promise
        // contract address. When the module set was NOT explicitly
        // configured, a missing address must not brick an otherwise-valid
        // eth-bridge deployment — warn and skip the module. When the
        // operator explicitly asked for callback-share, fail loudly.
        if (this.modulesExplicitlyConfigured()) {
          throw new Error(
            'callback-share module requires --promise-address / PROMISE_CONTRACT_ADDRESS',
          )
        }
        this.logger.warn(
          'callback-share module disabled: no --promise-address / PROMISE_CONTRACT_ADDRESS configured',
        )
      } else {
        modules.push(
          new CallbackShareModule(config.promiseAddress, failureRegistry),
        )
      }
    }

    // general-relay is the catch-all and must be built last: it skips any
    // message whose sender another enabled module already owns, so it needs the
    // union of every prior module's ownedSenders (lowercased) to avoid
    // double-relaying (Decision A).
    if (enabled.has('general-relay')) {
      const claimedSenders = new Set(
        modules
          .flatMap((m) => m.ownedSenders ?? [])
          .map((addr) => addr.toLowerCase()),
      )
      modules.push(new GeneralRelayModule(failureRegistry, claimedSenders))
    }

    this.logger.info(
      { enabledModules: modules.map((m) => m.name) },
      'enabled relay modules',
    )
    return modules
  }

  private wireRelayer(
    config: Config,
    ponderClient: PonderInteropClient,
    clients: ClientManager,
    modules: RelayModule[],
    failureRegistry: RelayFailureRegistry,
  ): Relayer {
    const relayerConfig: RelayerConfig = {
      ponderInteropApi: config.ponderInteropApi,
      ponderClient,
      clients,
      failureRegistry,
      gasTankAddress: config.gasTankAddress,
      pagination: {
        pageLimit: parsePositiveIntEnv('PONDER_PAGE_LIMIT'),
        maxAttemptablePerCycle: parsePositiveIntEnv('PONDER_MAX_PER_CYCLE'),
        maxScanPerCycle: parsePositiveIntEnv('PONDER_MAX_SCAN_PER_CYCLE'),
      },
    }
    const relayer = this.createRelayer(this.logger, relayerConfig, modules)
    const metrics = new RelayerMetrics(this.metricsRegistry)

    const build = detectBuildInfo()
    metrics.buildInfo.set(build, 1)
    this.logger.info(build, 'build info')

    // Two-phase init: ctx.relayMessage closes over relayer; relayer needs ctx.
    const ctx: RunContext = {
      ponderClient,
      clients,
      log: this.logger,
      pendingMessages: [],
      pendingPromises: [],
      unsharedResolvedPromises: [],
      metrics,
      relayMessage: (params) => relayer.submitRelayMessage(params),
    }
    relayer.setContext(ctx)
    return relayer
  }

  protected async main(): Promise<void> {
    this.logger.info('worker interval: %dms', this.options.loopIntervalMs)

    while (!this.isShuttingDown) {
      const start = Date.now()
      try {
        await this.relayer.run()
      } catch (error) {
        this.logger.error({ err: error }, 'failed run')
      }

      const elapsed = Date.now() - start
      const sleepMs = Math.max(0, this.options.loopIntervalMs - elapsed)
      await new Promise((res) => setTimeout(res, sleepMs))
    }
  }
}

export { Relayer, RelayerApp, type RelayerConfig }

// Re-export module types and implementations for downstream consumers
export { ClientManager } from '@/services/clientManager.js'
export { EthBridgeModule } from '@/strategies/ethBridgeModule.js'
export type { RelayModule, RunContext, RunResult } from '@/strategies/types.js'
