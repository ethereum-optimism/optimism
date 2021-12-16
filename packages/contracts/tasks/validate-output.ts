'use strict'

import { task, extendConfig } from 'hardhat/config'

import { TASK_COMPILE } from 'hardhat/builtin-tasks/task-names'
import chalk from 'chalk'
import 'hardhat/types/config'
import {
  HardhatConfig,
  HardhatUserConfig,
  CompilerOutputContract,
} from 'hardhat/types'
import { BuildInfo } from 'hardhat/src/types/artifacts'

export interface Checks {
  title?: boolean // default: true,
  details?: boolean // default: true,
  compilationWarnings?: boolean // default: true,
  missingUserDoc?: boolean // default: true,
  missingDevDoc?: boolean // default: true,
}

declare module 'hardhat/types/config' {
  export interface HardhatUserConfig {
    outputChecks?: {
      include?: string[]
      exclude?: string[]
      runOnCompile?: boolean
      errorMode?: boolean
      checks?: Checks
    }
  }

  export interface HardhatConfig {
    outputChecks: {
      include: string[]
      exclude: string[]
      runOnCompile: boolean
      errorMode: boolean
      checks: Checks
    }
  }
}

extendConfig(
  (config: HardhatConfig, userConfig: Readonly<HardhatUserConfig>) => {
    config.outputChecks = {
      errorMode: userConfig.outputChecks?.errorMode || false,
      checks: {
        title: true,
        details: true,
        compilationWarnings: true,
        missingUserDoc: true,
        missingDevDoc: true,
        ...(userConfig.outputChecks?.checks || {}),
      },
      include: userConfig.outputChecks?.include || [],
      exclude: userConfig.outputChecks?.exclude || [],
      runOnCompile: userConfig.outputChecks?.runOnCompile || false,
    }
  }
)

interface ErrorInfo {
  type: ErrorType
  text: string
  at: string
  filePath: string
  fileName: string
}

enum ErrorType {
  MissingTitle,
  MissingDetails,
  CompilationWarning,
  // User Docs
  MissingUserDoc,
  // Dev Docs
  MissingDevDoc,
}

declare interface ErrorUserdocArrayItem {
  notice?: string
}

export interface ErrorDevdocArrayItem {
  details?: string
  params?: {
    [key: string]: string
  }
}

export interface CompilerOutputContractWithDocumentation
  extends CompilerOutputContract {
  devdoc?: {
    author?: string
    details?: string
    title?: string
    errors?: {
      [key: string]: ErrorDevdocArrayItem[]
    }
    events?: {
      [key: string]: {
        details: string
        params: {
          [key: string]: string
        }
      }
    }
    methods?: {
      [key: string]: {
        details?: string
        params: {
          [key: string]: string
        }
        returns: {
          [key: string]: string
        }
      }
    }
    returns?: {
      [key: string]: {
        details?: string
        params: {
          [key: string]: string
        }
      }
    }
    stateVariables?: {
      [key: string]: {
        details?: string
        params: {
          [key: string]: string
        }
        returns: {
          [key: string]: string
        }
      }
    }
  }
  userdoc?: {
    errors?: {
      [key: string]: ErrorUserdocArrayItem[]
    }
    events?: {
      [key: string]: {
        notice: string
      }
    }
    methods?: {
      [key: string]: {
        notice: string
      }
    }
    notice?: string
  }
}

export interface CompilerOutputWithDocsAndPath
  extends CompilerOutputContractWithDocumentation {
  filePath: string
  fileName: string
}

const setupErrors =
  (fileSource: string, fileName: string) =>
  (errorType: ErrorType, extraData?: any) => {
    const typeToMessage = () => {
      switch (errorType) {
        case ErrorType.MissingTitle:
          return 'Contract is missing title'
        case ErrorType.MissingDetails:
          return 'Contract is missing details'
        case ErrorType.CompilationWarning:
          return `Compilation warnings: \n ${extraData} `

        // User DOCS
        case ErrorType.MissingUserDoc:
          return `${extraData} is missing @notice`

        // DEV DOCS
        case ErrorType.MissingDevDoc:
          return `${extraData} is missing @notice`

        default:
          return undefined
      }
    }

    return `${
      errorType !== ErrorType.CompilationWarning ? 'Comments Error' : ''
    }: ${typeToMessage()}\n   @ ${fileName} \n   --> ${fileSource}\n`
  }

task(TASK_COMPILE, async (args, hre, runSuper) => {
  const config = hre.config.outputChecks

  // Updates the compiler settings
  for (const compiler of hre.config.solidity.compilers) {
    compiler.settings.outputSelection['*']['*'].push('devdoc')
    compiler.settings.outputSelection['*']['*'].push('userdoc')
  }

  // Calls the actual COMPILE task
  await runSuper()

  if (!config.runOnCompile) {
    return
  }

  const getBuildInfo = async (
    qualifiedName: string
  ): Promise<BuildInfo | undefined> => {
    return hre.artifacts.getBuildInfo(qualifiedName)
  }

  // Loops through all the qualified names to get all the compiled contracts
  const getContractBuildInfo = async (
    qualifiedName: string
  ): Promise<CompilerOutputWithDocsAndPath | undefined> => {
    const [source, name] = qualifiedName.split(':')

    const build = await getBuildInfo(qualifiedName)
    const info: CompilerOutputContractWithDocumentation =
      build?.output.contracts[source][name]

    return {
      ...info,
      filePath: source,
      fileName: name,
    } as CompilerOutputWithDocsAndPath

    return undefined
  }

  const checkForErrors = (info: CompilerOutputWithDocsAndPath): ErrorInfo[] => {
    const foundErrors = []
    const getErrorText = setupErrors(info.filePath, info.fileName)

    const addError = (errorType: ErrorType, extraData?: any) => {
      const text = getErrorText(errorType, extraData)
      foundErrors.push({
        text,
        type: errorType,
        at: '',
        filePath: info.filePath,
        fileName: info.fileName,
      })
    }

    const findByName = (searchInObj: object, entityName: string) => {
      if (!searchInObj || !entityName) {
        return
      }

      const key = Object.keys(searchInObj).find((methodSigniture) => {
        const name = methodSigniture.split('(')[0]
        return name === entityName
      })

      if (!key) {
        return
      }
      return searchInObj[key]
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const checkConstructor = (entity) => {
      // TODO:
      return
    }

    const checkEvent = (entity) => {
      if (config.checks.missingUserDoc) {
        const userDocEntry = findByName(info.userdoc.events, entity.name)

        if (!userDocEntry || !userDocEntry.notice) {
          addError(ErrorType.MissingUserDoc, `Event: (${entity.name})`)
        }
      }
      if (config.checks.missingDevDoc) {
        const devDocEntry = findByName(info.devdoc.events, entity.name)

        // TODO: Extend with checks for params, returns
        if (!devDocEntry) {
          addError(ErrorType.MissingUserDoc, `Event: (${entity.name})`)
        }
      }
    }

    const checkFunction = (entity) => {
      if (config.checks.missingUserDoc) {
        const userDocEntry = findByName(info.userdoc.methods, entity.name)

        if (!userDocEntry || !userDocEntry.notice) {
          addError(ErrorType.MissingUserDoc, `Function: (${entity.name})`)
        }
      }
      if (config.checks.missingDevDoc) {
        const devDocEntryFunc = findByName(info.devdoc.methods, entity.name)
        const devDocEntryVar = findByName(
          info.devdoc.stateVariables,
          entity.name
        )

        // TODO: Extend with checks for params, returns
        if (!devDocEntryFunc && !devDocEntryVar) {
          addError(ErrorType.MissingUserDoc, `Function: (${entity.name})`)
        }
      }
    }

    if (config.checks.title && !info.devdoc.title) {
      addError(ErrorType.MissingTitle)
    }
    if (config.checks.details && !info.devdoc.details) {
      addError(ErrorType.MissingDetails)
    }

    if (Array.isArray(info.abi)) {
      // Loops through the abi and for each function/event/var check in the user/dev doc.
      info.abi.forEach((entity) => {
        if (entity.type === 'constructor') {
          checkConstructor(entity)
        } else if (entity.type === 'event') {
          checkEvent(entity)
        } else if (entity.type === 'function') {
          checkFunction(entity)
        }
      })
    }

    // TODO: check for userDoc.errors

    return foundErrors
  }

  console.log('<<< Starting Output Checks >>> ')

  const allContracts = await hre.artifacts.getAllFullyQualifiedNames()
  // console.log("allContracts", allContracts);
  const qualifiedNames = allContracts
    .filter((str) => str.startsWith('contracts'))
    .filter((path) => {
      // Checks if this contact is included
      const includesPath = config.include.some((str) => path.includes(str))
      const excludesPath = config.exclude.some((str) => path.includes(str))

      return (config.include.length === 0 || includesPath) && !excludesPath
    })
  console.log('qualifiedNames', qualifiedNames)

  // 1. Setup
  const buildInfo: BuildInfo[] = (
    await Promise.all(qualifiedNames.map(getBuildInfo))
  ).filter((inf) => inf !== undefined)

  const contractBuildInfo: CompilerOutputWithDocsAndPath[] = (
    await Promise.all(qualifiedNames.map(getContractBuildInfo))
  ).filter((inf) => inf !== undefined)

  // 2. Check
  const errors = contractBuildInfo.reduce((foundErrors, info) => {
    const docErrors = checkForErrors(info)

    if (docErrors && docErrors.length > 0) {
      foundErrors[info.filePath] = docErrors
    }

    return foundErrors
  }, {} as { [file: string]: ErrorInfo[] })

  // Check for CompilationWarning
  if (config.checks.compilationWarnings) {
    for (const bi of buildInfo) {
      const outputErrors = (bi.output as any).errors
      if (outputErrors && outputErrors.length > 0) {
        outputErrors.forEach((err) => {
          if (!errors[err.sourceLocation.file]) {
            errors[err.sourceLocation.file] = []
          }
          const filePath = err.sourceLocation.file
          const fileComponents = filePath.split('/')
          const fileName = fileComponents[fileComponents.length - 1]

          errors[err.sourceLocation.file].push({
            text: setupErrors(filePath, fileName)(
              ErrorType.CompilationWarning,
              err.formattedMessage
            ),
            type: ErrorType.CompilationWarning,
            at: '',
            filePath,
            fileName,
          })
        })
        break
      }
    }
  }

  // 3. Act
  const printErrors = (level: 'error' | 'warn' = 'warn') => {
    Object.keys(errors).forEach((file) => {
      const errorsInfo = errors[file]

      if (errorsInfo && errorsInfo.length > 0) {
        errorsInfo.forEach((erIn) => {
          if (level === 'error') {
            console.error(chalk.red(erIn.text))
          } else {
            console.warn(chalk.yellow(erIn.text))
          }
        })
      }
    })
  }

  if (config.errorMode) {
    printErrors('error')
    throw new Error('Missing Natspec Comments')
  }

  printErrors()

  console.log('✅ All Contracts have been checked for missing Natspec comments')
})
