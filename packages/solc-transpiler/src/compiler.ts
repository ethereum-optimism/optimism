/* External Imports */
import {
  ErroredTranspilation,
  OpcodeReplacerImpl,
  OpcodeWhitelistImpl,
  SuccessfulTranspilation,
  TranspilationError,
  Transpiler,
  TranspilerImpl,
} from '@eth-optimism/rollup-dev-tools'
import {
  bufToHexString,
  getLogger,
  hexStrToBuf,
  isValidHexAddress,
  Logger,
  remove0x,
} from '@eth-optimism/core-utils'
import * as solc from 'solc'
import { execSync } from 'child_process'
import * as requireFromString from 'require-from-string'

const log: Logger = getLogger('solc-transpiler')

interface TranspilationOutput {
  bytecode?: any
  deployedBytecode?: any
  errors?: any[]
}

/**
 * Solc-compatible entrypoint into compilation. This function expects Solc-formatted input (with the addition of the
 * `executionManagerAddress` field), and it will compile and transpile any contracts provided, resulting in
 * Solc-compatible output.
 *
 * @param configJsonString The Solc input as a string.
 * @param callbacks Optional callback(s) that are just passed to solc's compile function.
 * @returns The Solc output as a string.
 */
export const compile = (configJsonString: string, callbacks?: any): string => {
  const compiler = getCompiler(process.env.SOLC_VERSION)
  log.debug(`Trying to transpile with config: ${configJsonString}`)
  let json: any
  try {
    json = JSON.parse(configJsonString)
  } catch (e) {
    log.debug(`Cannot parse JSON: ${JSON.stringify(e)}`)
    // todo: populate some errors
    return compiler.compile(configJsonString)
  }

  const inputErrors: string = getInputErrors(json)
  if (!!inputErrors) {
    return inputErrors
  }

  const solcConfig: string = getSolcConfig(json)
  const resString = !!callbacks
    ? compiler.compile(solcConfig, callbacks)
    : compiler.compile(solcConfig)

  const res = JSON.parse(resString)
  if (
    'errors' in res &&
    !!res.errors &&
    !!res.errors.filter((x) => x.severity === 'error').length
  ) {
    log.debug(`ERROR: ${JSON.stringify(res)}`)
    return resString
  }

  const transpiler: Transpiler = new TranspilerImpl(
    new OpcodeWhitelistImpl(),
    new OpcodeReplacerImpl(getExecutionManagerAddress(json))
  )

  for (const [filename, fileJson] of Object.entries(res.contracts)) {
    log.debug(`Transpiling file: ${filename}`)
    for (const [contractName, contractJson] of Object.entries(fileJson)) {
      log.debug(`Transpiling contract: ${contractName}`)
      const output = transpileContract(
        transpiler,
        contractJson,
        filename,
        contractName
      )

      log.debug(`Transpiled output ${JSON.stringify(output)}`)

      res.contracts[filename][contractName].evm.bytecode.object = remove0x(
        output.bytecode || ''
      )
      res.contracts[filename][
        contractName
      ].evm.deployedBytecode.object = remove0x(output.deployedBytecode || '')
      res.contracts[filename][contractName].evm.bytecode.object = remove0x(
        output.bytecode || ''
      )
      res.contracts[filename][
        contractName
      ].evm.deployedBytecode.object = remove0x(output.deployedBytecode || '')

      if (!!output.errors) {
        if (!res.errors) {
          res.errors = []
        }

        res.errors.push(...output.errors)
      }
    }
  }

  return formatOutput(res, json)
}

/**
 * Gets the requested version of the solc module.
 * 
 * solc-js provides downloadable versions of itself which can be downloaded and
 * used to compile contracts that require different compiler versions. This
 * function must be synchronous so it can be used in the compilation process
 * which is also synchronous. To achieve this we construct a string of
 * JavaScript which downloads the latest version of solc and run that code using
 * `execSync`
 *
 * @param versionString The requested version of solc
 * @returns The requested version of the `solc` module or the latest version
 */
const getCompiler = (versionString: string): typeof solc => {
  if (!versionString) {
    return solc
  }

  const getCompilerString = `
    function httpsRequest(params, postData) {
      return new Promise(function(resolve, reject) {
          var req = https.request(params, function(res) {
              var body = [];
              res.on("data", function(chunk) {
                  body.push(chunk);
              });
              res.on("end", function() {
                  try {
                      body = Buffer.concat(body).toString();
                  } catch(e) {
                      reject(e);
                  }
                  resolve(body);
              });
          });
          req.on("error", function(err) {
              reject(err);
          });
          req.end();
      });
    }

    async function getSolcVersion(versionString) {
      const listUrl = "https://ethereum.github.io/solc-bin/bin/list.json";
      const {releases} = JSON.parse(await httpsRequest(listUrl))
      const solcUrl = "https://ethereum.github.io/solc-bin/bin/" + releases[versionString];
      return await httpsRequest(solcUrl);
    }

    (async () => {
      await process.stdout.write(await getSolcVersion("${versionString}"));
    })();
    `
  return solc.setupMethods(
    requireFromString(
      execSync(`${process.argv[0]} -e '${getCompilerString}'`).toString()
    )
  )
}

const getExecutionManagerAddress = (configObject: any): string => {
  return (
    configObject.settings.executionManagerAddress ||
    process.env.EXECUTION_MANAGER_ADDRESS
  )
}

/**
 * Validates the input jsonObject by checking to see that it's formatted properly. If not, it will return
 * an errors as properly-formatted solc output.
 *
 * @param configObject The config object being checked for validity
 * @returns undefined if valid, formatted error(s) as a string if invalid.
 */
const getInputErrors = (configObject: any): string => {
  if (!configObject.settings || typeof configObject.settings !== 'object') {
    return getFormattedSolcErrorOutput(
      'Input must include "settings" object in top-level object.'
    )
  }

  const executionManagerAddress: string = getExecutionManagerAddress(
    configObject
  )
  if (!executionManagerAddress || !isValidHexAddress(executionManagerAddress)) {
    return getFormattedSolcErrorOutput(
      'Input must include "executionManagerAddress" field in the "settings" object or there must be an "EXECUTION_MANAGER_ADDRESS" environment variable, and it must be a valid Ethereum address as a hex string (case insensitive).'
    )
  }
  log.info(`Compiling with executionManagerAddress ${executionManagerAddress}`)

  if (
    !configObject.settings.outputSelection ||
    typeof configObject.settings.outputSelection !== 'object' ||
    !Object.entries(configObject.settings.outputSelection).length
  ) {
    return getFormattedSolcErrorOutput(
      'Input must include a populated "outputSelection" object in "settings"'
    )
  }

  for (const [filename, fileConfig] of Object.entries(
    configObject.settings.outputSelection
  )) {
    if (typeof fileConfig !== 'object' || Array.isArray(fileConfig)) {
      return getFormattedSolcErrorOutput(
        '"outputSelection" configuration in "settings" must be of the form: { "filename": { "contractName": [] }, ... }'
      )
    }
    for (const contractConfig of Object.values(
      configObject.settings.outputSelection[filename]
    )) {
      if (!Array.isArray(contractConfig)) {
        return getFormattedSolcErrorOutput(
          '"outputSelection" configuration in "settings" must be of the form: { "filename": { "contractName": [] }, ... }'
        )
      }
    }
  }
}

/**
 * Takes the provided config and converts it into solc-js input.
 * This mainly entails:
 *   * removing the `executionManagerAddress` from settings
 *   * making sure `evm.legacyAssembly` is listed as an output selection
 * @param config The config object
 * @returns the formatted solc config
 */
const getSolcConfig = (config: any): string => {
  // Just deep cloning the config json
  const solcConfig = JSON.parse(JSON.stringify(config))
  delete solcConfig.settings.executionManagerAddress

  for (const [filename, fileConfig] of Object.entries(
    solcConfig.settings.outputSelection
  )) {
    for (const [contractName, contractConfig] of Object.entries(fileConfig)) {
      const lowerConfig = contractConfig.map((x) => x.toLowerCase())
      if (!('evm.legacyAssembly' in lowerConfig) && !('*' in lowerConfig)) {
        solcConfig.settings.outputSelection[filename][contractName].push(
          'evm.legacyAssembly'
        )
      }
    }
  }

  return JSON.stringify(solcConfig)
}

/**
 * Takes the transpilation output object and formats it to be returned.
 * This mainly entails:
 *    * Removing `legacyAssemby` output if it was not specified on the input config
 *    * Formatting the resulting object as a string
 *
 * @param transpiledOutput The transpilation result.
 * @params inputConfig The input config that indicates how the output should be formatted.
 * @returns The formatted output.
 */
const formatOutput = (transpiledOutput: any, inputConfig: any): string => {
  for (const [filename, fileObj] of Object.entries(
    transpiledOutput.contracts
  )) {
    for (const [contractName, contractObj] of Object.entries(fileObj)) {
      const filenameConfig =
        filename in inputConfig.settings.outputSelection ? filename : '*'
      const contractNameConfig =
        contractName in inputConfig.settings.outputSelection[filenameConfig]
          ? contractName
          : '*'
      const outputConfig =
        inputConfig.settings.outputSelection[filenameConfig][contractNameConfig]
      if (!('evm.legacyAssembly' in outputConfig) && !('*' in outputConfig)) {
        delete transpiledOutput.contracts[filename][contractName].evm
          .legacyAssembly
      }
    }
  }

  return JSON.stringify(transpiledOutput)
}

/**
 * Gets Solc-formatted errors from the provided transpilation errors.
 *
 * @param transpilationErrors.
 * @param file The file in which the errors occurred.
 * @param contract The contract in qhich the errors occurred
 * @param isDeployedBytecode.
 * @returns The Solc-formatted errors
 */
const getSolcErrorsFromTranspilerErrors = (
  transpilationErrors: TranspilationError[],
  file: string,
  contract: string,
  isDeployedBytecode: boolean = false
): any[] => {
  return transpilationErrors.map((x) => {
    const message: string = `${file}:${contract} error [${
      x.message
    }] at index ${x.index} of ${
      isDeployedBytecode ? 'deployed bytecode' : 'bytecode'
    }`
    return {
      component: 'general',
      formattedMessage: message,
      message,
      severity: 'error',
      type: 'CompilerError',
    }
  })
}

/**
 * Creates a formatted solc-js error output string from the provided params.
 *
 * @param message The error message.
 * @param severity The severity of the error.
 * @param component The component of the error
 * @param type The type of the error
 * @returns The formatted string.
 */
const getFormattedSolcErrorOutput = (
  message: string,
  severity: string = 'error',
  component: string = 'general',
  type: string = 'JSONError'
): string => {
  return JSON.stringify({
    errors: [
      {
        component,
        formattedMessage: message,
        message,
        severity,
        type,
      },
    ],
  })
}

/**
 * Gets the bytecode or the deployedBytecode from the solc output for a specific contract.
 *
 * @param contractSolcOutput The solc output for the contract in question.
 * @param isDeployedBytecode Whether we're getting the deployed bytecode or the bytecode.
 * @returns The bytecode if it exists in the solc output.
 */
const getBytecode = (
  contractSolcOutput: any,
  isDeployedBytecode: boolean
): string => {
  try {
    const code: string = isDeployedBytecode
      ? contractSolcOutput.evm.deployedBytecode.object
      : contractSolcOutput.evm.bytecode.object

    const auxData = getAuxData(contractSolcOutput)
    return !!auxData ? code.split(auxData)[0] : code
  } catch (e) {
    return undefined
  }
}

/**
 * Gets the auxdata from the compiler output.
 * This is the fingerprint of the bytecode that may depend on compiler and version and therefore should be removed
 * from bytecode if strictly analyzing the bytecode.
 *
 * @param contractSolcOutput The solc-js compile(...) output.
 * @returns The auxdata in question.
 */
const getAuxData = (contractSolcOutput: any): string => {
  try {
    return contractSolcOutput.evm.legacyAssembly['.data']['0']['.auxdata']
  } catch (e) {
    return undefined
  }
}

/**
 * Transpiles the provided solc output, overwriting the `bytecode` and `deployedBytecode`.
 *
 * @param transpiler The transpiler to use.
 * @param contractSolcOutput The contract solc output.
 * @param filename The file being transpiled.
 * @param contractName The contract being transpiled.
 * @returns The updated solc output where the bytecode and deployedBytecode are overwritten with transpiled bytecode if present.
 */
const transpileContract = (
  transpiler: Transpiler,
  contractSolcOutput: any,
  filename: string,
  contractName: string
): TranspilationOutput => {
  const originalBytecodeSize: number = hexStrToBuf(
    contractSolcOutput.evm.bytecode.object
  ).byteLength
  let bytecode: string = getBytecode(contractSolcOutput, false)
  let deployedBytecode: string = getBytecode(contractSolcOutput, true)

  if (!bytecode && !deployedBytecode) {
    return contractSolcOutput
  }

  if (!!bytecode) {
    const transpilationResult = transpiler.transpile(
      hexStrToBuf(bytecode),
      hexStrToBuf(deployedBytecode),
      originalBytecodeSize
    )
    if (!transpilationResult.succeeded) {
      const errorResult: ErroredTranspilation = transpilationResult as ErroredTranspilation
      return {
        errors: getSolcErrorsFromTranspilerErrors(
          errorResult.errors,
          filename,
          contractName
        ),
      }
    }
    bytecode = bufToHexString(
      (transpilationResult as SuccessfulTranspilation).bytecode
    )
  }

  if (!!deployedBytecode) {
    const transpilationResult = transpiler.transpileRawBytecode(
      hexStrToBuf(deployedBytecode)
    )
    if (!transpilationResult.succeeded) {
      const errorResult: ErroredTranspilation = transpilationResult as ErroredTranspilation
      return {
        errors: getSolcErrorsFromTranspilerErrors(
          errorResult.errors,
          filename,
          contractName,
          true
        ),
      }
    }
    deployedBytecode = bufToHexString(
      (transpilationResult as SuccessfulTranspilation).bytecode
    )
  }

  return {
    bytecode,
    deployedBytecode,
  }
}
