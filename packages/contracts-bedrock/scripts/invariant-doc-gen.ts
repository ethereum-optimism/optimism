import fs from 'fs'

const BASE_INVARIANTS_DIR = `${__dirname}/../contracts/test/invariants`
const BASE_DOCS_DIR = `${__dirname}/../invariant-docs`
const BASE_GH_URL =
  'https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/invariant-docs/'

type Contract = {
  name: string
  docs: InvariantDoc[]
}

type InvariantDoc = {
  header?: string
  desc?: string
  lineNo?: number
}

/**
 * Lazy-parses all test files in the `contracts/test/invariants` directory to generate documentation
 * on all invariant tests.
 */
const docGen = (): void => {
  // Grab all files within the invariants test dir
  const files = fs.readdirSync(BASE_INVARIANTS_DIR)

  // Array to store all found invariant documentation comments.
  const docs: Contract[] = []

  // TODO: Handle multiple contracts per file (?)
  for (const file of files) {
    // Read the contents of the invariant test file.
    const fileContents = fs
      .readFileSync(`${BASE_INVARIANTS_DIR}/${file}`)
      .toString()

    // Split the file into individual lines and trim whitespace.
    const lines = fileContents.split('\n').map((line: string) => line.trim())

    // Create an object to store all invariant test docs for the current contract
    const contract: Contract = { name: file.replace('.t.sol', ''), docs: [] }

    let currentDoc: InvariantDoc

    // Loop through all lines to find comments.
    for (let i = 0; i < lines.length; i++) {
      let line = lines[i]

      if (line.startsWith('/**')) {
        // We are at the beginning of a new doc comment. Reset the currentDoc array.
        currentDoc = {}

        // Move on to the next line
        line = lines[++i]

        // We have an invariant doc
        if (line.startsWith('* INVARIANT:')) {
          // TODO: Handle ambiguous case for `INVARIANT: ` prefix.
          // Assign the header of the invariant doc.
          currentDoc = {
            header: line.replace('* INVARIANT:', '').trim(),
            desc: '',
          }

          // Process the description
          while ((line = lines[++i]).startsWith('*')) {
            line = line.replace(/\*(\/)?/, '').trim()

            if (line.length > 0) {
              currentDoc.desc += `${line}\n`
            }
          }

          // Set the line number of the test
          currentDoc.lineNo = i + 1

          // Add the doc to the contract
          contract.docs.push(currentDoc)
        }
      }
    }

    // Add the contract to the array of docs
    docs.push(contract)
  }

  for (const contract of docs) {
    fs.writeFileSync(
      `${BASE_DOCS_DIR}/${contract.name}.md`,
      renderContractDoc(contract)
    )
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

/**
 * Render a `Contract` object into valid markdown.
 */
const renderContractDoc = (contract: Contract): string => {
  const header = `# ${contract.name} Invariants`
  const docs = contract.docs
    .map((doc: InvariantDoc) => {
      return `## ${doc.header}\n**Test:** [\`L${doc.lineNo}\`](${BASE_GH_URL}${contract.name}.t.sol)\n\n${doc.desc}`
    })
    .join('\n\n')

  return `${header}\n\n${docs}`
}

// Generate the docs
docGen()
