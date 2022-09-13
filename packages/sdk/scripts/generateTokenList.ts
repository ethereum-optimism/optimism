import fs, { rmdirSync } from 'fs'
import path from 'path'

import { fetchTokenList } from '../src/utils/token-list-utils'

const generatedDirectory = path.join(__dirname, '..', 'src', 'generated')

fetchTokenList()
  .then((list) => {
    if (fs.existsSync(generatedDirectory)) {
      rmdirSync(generatedDirectory, { recursive: true })
    }
    fs.mkdirSync(generatedDirectory)
    fs.writeFileSync(
      path.join(generatedDirectory, 'tokenList.json'),
      JSON.stringify(list, null, 2)
    )
  })
  .then(() => {
    console.log('successfully generated token list')
  })
  .catch((err) => {
    console.error('There was an error generating token list', err)
  })
