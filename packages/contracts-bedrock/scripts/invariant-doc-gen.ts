import fs from 'fs'
import path from 'path'

const BASE_INVARIANTS_DIR = path.join(__dirname, '..', 'test', 'invariants')
const BASE_DOCS_DIR = path.join(__dirname, '..', 'invariant-docs')
const BASE_INVARIANT_GH_URL = '../test/invariants/'
const NATSPEC_INV = '@custom:invariant'

// Represents an invariant test contract
type Contract = {
  name: string
  fileName: string
  docs: InvariantDoc[]
}

// Represents the documentation of an invariant
type InvariantDoc = {
  header?: string
  desc?: string
  lineNo?: number
}

const writtenFiles = []

// Lazy-parses all test files in the `test/invariants` directory
// to generate documentation on all invariant tests.
const docGen = (dir: string): void => {
  // Grab all files within the invariants test dir
  const files = fs.readdirSync(dir)

  // Array to store all found invariant documentation comments.
  const docs: Contract[] = []

  for (const fileName of files) {
    // Read the contents of the invariant test file.
    const fileContents = fs.readFileSync(path.join(dir, fileName)).toString()

    // Split the file into individual lines and trim whitespace.
    const lines = fileContents.split('\n').map((line: string) => line.trim())

    // Create an object to store all invariant test docs for the current contract
    const name = fileName.replace('.t.sol', '')
    const contract: Contract = { name, fileName, docs: [] }

    let currentDoc: InvariantDoc

    // Loop through all lines to find comments.
    for (let i = 0; i < lines.length; i++) {
      let line = lines[i]

      // We have an invariant doc
      if (line.startsWith(`/// ${NATSPEC_INV}`)) {
        // Assign the header of the invariant doc.
        // TODO: Handle ambiguous case for `INVARIANT: ` prefix.
        currentDoc = {
          header: line.replace(`/// ${NATSPEC_INV}`, '').trim(),
          desc: '',
        }

        // If the header is multi-line, continue appending to the `currentDoc`'s header.
        line = lines[++i]
        while (line.startsWith(`///`) && line.trim() !== '///') {
          currentDoc.header += ` ${line.replace(`///`, '').trim()}`
          line = lines[++i]
        }

        // Process the description
        while ((line = lines[++i]).startsWith('///')) {
          line = line.replace('///', '').trim()

          // If the line has any contents, insert it into the desc.
          // Otherwise, consider it a linebreak.
          currentDoc.desc += line.length > 0 ? `${line} ` : '\n'
        }

        // Set the line number of the test
        currentDoc.lineNo = i + 1

        // Add the doc to the contract
        contract.docs.push(currentDoc)
      }
    }

    // Add the contract to the array of docs
    docs.push(contract)
  }

  for (const contract of docs) {
    const fileName = path.join(BASE_DOCS_DIR, `${contract.name}.md`)
    const alreadyWritten = writtenFiles.includes(fileName)

    // If the file has already been written, append the extra docs to the end.
    // Otherwise, write the file from scratch.
    fs.writeFileSync(
      fileName,
      alreadyWritten
        ? `${fs.readFileSync(fileName)}\n${renderContractDoc(contract, false)}`
        : renderContractDoc(contract, true)
    )

    // If the file was just written for the first time, add it to the list of written files.
    if (!alreadyWritten) {
      writtenFiles.push(fileName)
    }
  }

  console.log(
    `Generated invariant test documentation for:\n - ${
      docs.length
    } contracts\n - ${docs.reduce(
      (acc: number, contract: Contract) => acc + contract.docs.length,
      0
    )} invariant tests\nsuccessfully!`
  )
}

//  Generate a table of contents for all invariant docs and place it in the README.
const tocGen = (): void => {
  const autoTOCPrefix = '<!-- START autoTOC -->\n'
  const autoTOCPostfix = '<!-- END autoTOC -->\n'

  // Grab the name of all markdown files in `BASE_DOCS_DIR` except for `README.md`.
  const files = fs
    .readdirSync(BASE_DOCS_DIR)
    .filter((fileName: string) => fileName !== 'README.md')

  // Generate a table of contents section.
  const tocList = files
    .map(
      (fileName: string) => `- [${fileName.replace('.md', '')}](./${fileName})`
    )
    .join('\n')
  const toc = `${autoTOCPrefix}\n## Table of Contents\n${tocList}\n${autoTOCPostfix}`

  // Write the table of contents to the README.
  const readmeContents = fs
    .readFileSync(path.join(BASE_DOCS_DIR, 'README.md'))
    .toString()
  const above = readmeContents.split(autoTOCPrefix)[0]
  const below = readmeContents.split(autoTOCPostfix)[1]
  fs.writeFileSync(
    path.join(BASE_DOCS_DIR, 'README.md'),
    `${above}${toc}${below}`
  )
}

// Render a `Contract` object into valid markdown.
const renderContractDoc = (contract: Contract, header: boolean): string => {
  const _header = header ? `# \`${contract.name}\` Invariants\n` : ''
  const docs = contract.docs
    .map((doc: InvariantDoc) => {
      const line = `${contract.fileName}#L${doc.lineNo}`
      return `## ${doc.header}\n**Test:** [\`${line}\`](${BASE_INVARIANT_GH_URL}${line})\n\n${doc.desc}`
    })
    .join('\n\n')
  return `${_header}\n${docs}`
}

// Generate the docs

// Forge
console.log('Generating docs for forge invariants...')
docGen(BASE_INVARIANTS_DIR)

// New line
console.log()

// Generate an updated table of contents
tocGen()
