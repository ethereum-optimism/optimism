/* Imports: External */
import * as fs from 'fs'
import * as path from 'path'
import fetch from 'node-fetch'
import { ethers } from 'ethers'
import { subtask, extendEnvironment } from 'hardhat/config'
import {
  HardhatNetworkHDAccountsConfig,
  HardhatNetworkAccountUserConfig,
} from 'hardhat/types/config'
import { getCompilersDir } from 'hardhat/internal/util/global-dir'
import { Artifacts } from 'hardhat/internal/artifacts'
import {
  TASK_COMPILE_SOLIDITY_RUN_SOLCJS,
  TASK_COMPILE_SOLIDITY_RUN_SOLC,
} from 'hardhat/builtin-tasks/task-names'

/* Imports: Internal */
import './type-extensions'

const OPTIMISM_SOLC_VERSION_URL =
  'https://api.github.com/repos/ethereum-optimism/solc-bin/git/refs/heads/gh-pages'

const OPTIMISM_SOLC_BIN_URL =
  'https://raw.githubusercontent.com/ethereum-optimism/solc-bin/gh-pages/bin'

// I figured this was a reasonably modern default, but not sure if this is too new. Maybe we can
// default to 0.6.X instead?
const DEFAULT_OVM_SOLC_VERSION = '0.7.6'

// Poll the node every 50ms, to override ethers.js's default 4000ms causing OVM
// tests to be slow.
const OVM_POLLING_INTERVAL = 50

/**
 * Find or generate an OVM soljson.js compiler file and return the path of this file.
 * We pass the path to this file into hardhat.
 * @param version Solidity compiler version to get a path for in the format `X.Y.Z`.
 * @return Path to the downloaded soljson.js file.
 */
const getOvmSolcPath = async (version: string): Promise<string> => {
  // If __DANGEROUS_OVM_IGNORE_ERRORS__ env var is not undefined we append the -no-errors suffix to the solc version.
  if (process.env.__DANGEROUS_OVM_IGNORE_ERRORS__) {
    console.log('\n\n__DANGEROUS_OVM_IGNORE_ERRORS__ IS ENABLED!\n\n')
    version += '-no_errors'
  }

  // First, check to see if we've already downloaded this file. Hardhat gives us a folder to use as
  // a compiler cache, so we'll just be nice and use an `ovm` subfolder.
  const ovmCompilersCache = path.join(await getCompilersDir(), 'ovm')

  // Need to create the OVM compiler cache folder if it doesn't already exist.
  if (!fs.existsSync(ovmCompilersCache)) {
    fs.mkdirSync(ovmCompilersCache, { recursive: true })
  }

  // Pull information about the latest commit in the solc-bin repo. We'll use this to invalidate
  // our compiler cache if necessary.
  const remoteCompilerVersion = await (
    await fetch(OPTIMISM_SOLC_VERSION_URL)
  ).text()

  // Pull the locally stored info about the latest commit. If this differs from the remote info
  // then we know to invalidate our cache.
  let cachedCompilerVersion = ''
  const cachedCompilerVersionPath = path.join(
    ovmCompilersCache,
    `version-info-${version}.json`
  )
  if (fs.existsSync(cachedCompilerVersionPath)) {
    cachedCompilerVersion = fs
      .readFileSync(cachedCompilerVersionPath)
      .toString()
  }

  // Check to see if we already have this compiler version downloaded. We store the cached files at
  // `X.Y.Z.js`. If it already exists, just return that instead of downloading a new one.
  const cachedCompilerPath = path.join(ovmCompilersCache, `${version}.js`)
  if (
    remoteCompilerVersion === cachedCompilerVersion &&
    fs.existsSync(cachedCompilerPath)
  ) {
    return cachedCompilerPath
  }

  console.log(`Downloading OVM compiler version ${version}`)

  // We don't have a cache, so we'll download this file from GitHub. Currently stored at
  // ethereum-optimism/solc-bin.
  const compilerContentResponse = await fetch(
    OPTIMISM_SOLC_BIN_URL + `/soljson-v${version}.js`
  )

  // Throw if this request failed, e.g., 404 because of an invalid version.
  if (!compilerContentResponse.ok) {
    throw new Error(
      `Unable to download OVM compiler version ${version}. Are you sure that version exists?`
    )
  }

  // Otherwise, write the content to the cache. We probably want to do some sort of hash
  // verification against these files but it's OK for now. The real "TODO" here is to instead
  // figure out how to properly extend and/or hack Hardat's CompilerDownloader class.
  const compilerContent = await compilerContentResponse.text()
  fs.writeFileSync(cachedCompilerPath, compilerContent)
  fs.writeFileSync(cachedCompilerVersionPath, remoteCompilerVersion)

  return cachedCompilerPath
}

// TODO: implement a more elegant fix for this.
// Hardhat on M1 Macbooks does not run TASK_COMPILE_SOLIDITY_RUN_SOLC, but instead this runs one,
// TASK_COMPILE_SOLIDITY_RUN_SOLCJS.  We reroute this task back to the solc task in the case
// of OVM compilation.
subtask(TASK_COMPILE_SOLIDITY_RUN_SOLCJS, async (args, hre, runSuper) => {
  const argsAny = args as any
  if (argsAny.real || hre.network.ovm !== true) {
    for (const file of Object.keys(argsAny.input.sources)) {
      // Ignore any contract that has this tag or in ignore list
      if (
        argsAny.input.sources[file].content.includes('// @unsupported: evm') &&
        hre.network.ovm !== true
      ) {
        delete argsAny.input.sources[file]
      }
    }
    return runSuper(args)
  } else {
    return hre.run(TASK_COMPILE_SOLIDITY_RUN_SOLC, {
      ...argsAny,
      solcPath: argsAny.solcJsPath,
    })
  }
})

subtask(
  TASK_COMPILE_SOLIDITY_RUN_SOLC,
  async (args: { input: any; solcPath: string }, hre, runSuper) => {
    const ignoreRxList = hre.network.config.ignoreRxList || []
    const ignore = (filename: string) =>
      ignoreRxList.reduce(
        (ignored: boolean, rx: string | RegExp) =>
          ignored || new RegExp(rx).test(filename),
        false
      )
    if (hre.network.ovm !== true) {
      // Separate the EVM and OVM inputs.
      for (const file of Object.keys(args.input.sources)) {
        // Ignore any contract that has this tag or in ignore list
        if (
          args.input.sources[file].content.includes('// @unsupported: evm') ||
          ignore(file)
        ) {
          delete args.input.sources[file]
        } else {
          //console.log(file + ' included');
        }
      }
      return runSuper(args)
    }

    // Just some silly sanity checks, make sure we have a solc version to download. Our format is
    // `X.Y.Z` (for now).
    let ovmSolcVersion = DEFAULT_OVM_SOLC_VERSION
    if (hre.config?.ovm?.solcVersion) {
      ovmSolcVersion = hre.config.ovm.solcVersion
    }

    // Get a path to a soljson file.
    const ovmSolcPath = await getOvmSolcPath(ovmSolcVersion)

    // These objects get fed into the compiler. We're creating two of these because we need to
    // throw one into the OVM compiler and another into the EVM compiler. Users are able to prevent
    // certain files from being compiled by the OVM compiler by adding "// @unsupported: ovm"
    // somewhere near the top of their file.
    const ovmInput = {
      language: 'Solidity',
      sources: {},
      settings: args.input.settings,
    }

    // Separate the EVM and OVM inputs.
    for (const file of Object.keys(args.input.sources)) {
      // Ignore any contract that has this tag or in ignore list
      if (
        !args.input.sources[file].content.includes('// @unsupported: ovm') &&
        !ignore(file)
      ) {
        ovmInput.sources[file] = args.input.sources[file]
      }
    }

    if (Object.keys(ovmInput.sources).length === 0) {
      return {}
    }

    // Build both inputs separately.
    const ovmOutput = await hre.run(TASK_COMPILE_SOLIDITY_RUN_SOLCJS, {
      input: ovmInput,
      solcJsPath: ovmSolcPath,
      real: true,
    })

    // Just doing this to add some extra useful information to any errors in the OVM compiler output.
    ovmOutput.errors = (ovmOutput.errors || []).map((error: any) => {
      if (error.severity === 'error') {
        error.formattedMessage = `OVM Compiler Error (insert "// @unsupported: ovm" if you don't want this file to be compiled for the OVM):\n ${error.formattedMessage}`
      }

      return error
    })

    return ovmOutput
  }
)

extendEnvironment(async (hre) => {
  if (hre.network.config.ovm) {
    hre.network.ovm = hre.network.config.ovm

    let artifactsPath = hre.config.paths.artifacts
    if (!artifactsPath.endsWith('-ovm')) {
      artifactsPath = artifactsPath + '-ovm'
    }

    let cachePath = hre.config.paths.cache
    if (!cachePath.endsWith('-ovm')) {
      cachePath = cachePath + '-ovm'
    }

    // Forcibly update the artifacts object.
    hre.config.paths.artifacts = artifactsPath
    hre.config.paths.cache = cachePath
    ;(hre as any).artifacts = new Artifacts(artifactsPath)

    // if typechain is present, send the typed bindings to an ovm-specific
    // directory
    if ((hre.config as any).typechain) {
      if (!(hre as any).config.typechain.outDir.endsWith('-ovm')) {
        ;(hre as any).config.typechain.outDir += '-ovm'
      }
    }

    // if ethers is present and the node is running, override the polling interval to not wait the full
    // duration in tests
    if ((hre as any).ethers) {
      const interval = hre.network.config.interval || OVM_POLLING_INTERVAL
      if ((hre as any).ethers.provider.pollingInterval === interval) {
        return
      }

      // override the provider polling interval
      const provider = new ethers.providers.JsonRpcProvider(
        (hre as any).ethers.provider.url || (hre as any).network.config.url
      )
      provider.pollingInterval = interval

      // the gas price is overriden to the user provided gasPrice or to 0.
      provider.getGasPrice = async () =>
        ethers.BigNumber.from(hre.network.config.gasPrice || 0)

      // if the node is up, override the getSigners method's signers
      try {
        let signers: ethers.Signer[]
        const accounts = hre.network.config.accounts as
          | HardhatNetworkHDAccountsConfig
          | HardhatNetworkAccountUserConfig[]
        if (Array.isArray(accounts)) {
          signers = (accounts as HardhatNetworkAccountUserConfig[]).map(
            (account) =>
              new ethers.Wallet(
                typeof account === 'string' ? account : account.privateKey
              ).connect(provider)
          )
        } else if (accounts) {
          const indices = Array.from(Array(20).keys()) // generates array of [0, 1, 2, ..., 18, 19]
          signers = indices.map((i) =>
            ethers.Wallet.fromMnemonic(
              accounts.mnemonic,
              `${accounts.path}/${i}`
            ).connect(provider)
          )
        } else {
          signers = await (hre as any).ethers.getSigners()
          signers = signers.map((s: any) => {
            s._signer.provider.pollingInterval = interval
            return s
          })
        }

        ;(hre as any).ethers.getSigners = () => signers
        /* tslint:disable:no-empty */
      } catch (e) {}

      // Update the provider at the very end to avoid any weird issues.
      ;(hre as any).ethers.provider = provider
    }
  }
})
