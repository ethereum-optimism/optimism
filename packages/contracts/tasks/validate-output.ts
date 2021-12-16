/* eslint-disable guard-for-in, max-len, no-await-in-loop, no-restricted-syntax */
import { task } from 'hardhat/config'
import { CompilerOutputContractWithDocumentation } from '@primitivefi/hardhat-dodoc/dist/src/dodocTypes'
import { TASK_COMPILE } from 'hardhat/builtin-tasks/task-names'

// import { CompilerOutputContractWithDocumentation, Doc } from './dodocTypes';
// import { decodeAbi } from './abiDecoder';
// import './type-extensions';

// TODO:
// extendConfig((config: HardhatConfig, userConfig: Readonly<HardhatUserConfig>) => {
//   // eslint-disable-next-line no-param-reassign
//   config.outputChecks = {
//     error: userConfig.outputChecks?.error || false,
//     checks: userConfig.outputChecks?.checks || [],
//     include: userConfig.outputChecks?.include || [],
//     exclude: userConfig.outputChecks?.exclude || [],
//     runOnCompile: userConfig.outputChecks?.runOnCompile !== undefined ? userConfig.outputChecks?.runOnCompile : true,
//   };
// });

interface ErrorInfo {
  type: ErrorInfo
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
          return ''

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

    return `Error in ${fileName} at path: ${fileSource}\n ---> ${typeToMessage()}`
  }

type CompilerOutputWithDocsAndPath = CompilerOutputContractWithDocumentation & {
  filePath: string
  fileName: string
}

// Custom task triggered when COMPILE is called
task(TASK_COMPILE, async (args, hre, runSuper) => {
  // const config = hre.config.outputChecks;

  // Updates the compiler settings
  for (const compiler of hre.config.solidity.compilers) {
    compiler.settings.outputSelection['*']['*'].push('devdoc')
    compiler.settings.outputSelection['*']['*'].push('userdoc')
  }

  // Calls the actual COMPILE task
  await runSuper()

  // if (!config.runOnCompile) {
  //   return;
  // }

  console.log('<<< Starting Output Checks >>> ')

  const allContracts = await hre.artifacts.getAllFullyQualifiedNames()
  // console.log("allContracts", allContracts);
  const qualifiedNames = allContracts.filter((str) =>
    str.startsWith('contracts')
  )
  console.log('qualifiedNames', qualifiedNames)
  // Loops through all the qualified names to get all the compiled contracts

  const getBuildInfo = async (
    qualifiedName: string
  ): Promise<CompilerOutputWithDocsAndPath | undefined> => {
    const [source, name] = qualifiedName.split(':')
    // TODO:
    // Checks if the documentation has to be generated for this contract
    // if (
    //   (config.include.length === 0 || config.include.includes(name))
    //   && !config.exclude.includes(name)
    // ) {
    const contractBuildInfo = await hre.artifacts.getBuildInfo(qualifiedName)
    const info: CompilerOutputContractWithDocumentation =
      contractBuildInfo?.output.contracts[source][name]

    return {
      ...info,
      filePath: source,
      fileName: name,
    } as CompilerOutputWithDocsAndPath
    // }
    return undefined
  }

  const checkForErrors = (info: CompilerOutputWithDocsAndPath): ErrorInfo[] => {
    const foundErrors = []

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
      const userDocEntry = findByName(info.userdoc.events, entity.name)

      if (!userDocEntry || !userDocEntry.notice) {
        addError(ErrorType.MissingUserDoc, `Event: (${entity.name})`)
      }

      const devDocEntry = findByName(info.devdoc.events, entity.name)

      // TODO: Extend with checks for params, returns
      if (!devDocEntry) {
        addError(ErrorType.MissingUserDoc, `Event: (${entity.name})`)
      }
    }

    const checkFunction = (entity) => {
      const userDocEntry = findByName(info.userdoc.methods, entity.name)

      if (!userDocEntry || !userDocEntry.notice) {
        addError(ErrorType.MissingUserDoc, `Function: (${entity.name})`)
      }

      const devDocEntryFunc = findByName(info.devdoc.methods, entity.name)
      const devDocEntryVar = findByName(info.devdoc.stateVariables, entity.name)

      // TODO: Extend with checks for params, returns
      if (!devDocEntryFunc && !devDocEntryVar) {
        addError(ErrorType.MissingUserDoc, `Function: (${entity.name})`)
      }
    }

    // ErrorInfo: Missing
    const getErrorText = setupErrors(info.filePath, info.fileName)

    if (!info.devdoc.title) {
      addError(ErrorType.MissingTitle)
    }
    if (!info.devdoc.details) {
      addError(ErrorType.MissingDetails)
    }

    if (Array.isArray(info.abi)) {
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

    // Loop through the abi and for each function/event/var check in the user/dev doc.

    return foundErrors
  }

  // 1. Setup
  const buildInfo: CompilerOutputWithDocsAndPath[] = (
    await Promise.all(qualifiedNames.map(getBuildInfo))
  ).filter((inf) => inf !== undefined)

  // 2. Check
  const errors = buildInfo.reduce((foundErrors, info) => {
    const docErrors = checkForErrors(info)
    console.log('DOC ERRORS: ', docErrors)

    if (docErrors && docErrors.length > 0) {
      foundErrors[info.filePath] = docErrors
    }

    return foundErrors
  }, {} as { [file: string]: ErrorInfo[] })

  // 3. Act
  const printErrors = (level: 'error' | 'warn' = 'warn') => {
    Object.keys(errors).forEach((file) => {
      const errorsInfo = errors[file]

      if (errorsInfo && errorsInfo.length > 0) {
        ;(errorsInfo as ErrorInfo[]).forEach((erIn) => {
          if ((level === 'error')) {
            console.error(erIn.text)
          } else {
            console.warn(erIn.text)
          }
        })
      }
    })
  }

  // if throwing is enabled -> throw
  if (false) {
    // Loop through and console.error
    // & throw
    printErrors('error')
    throw new Error('Missing Natspec Comments')
  }

  printErrors()

  console.log('✅ All Doc Checks are passing')
})
