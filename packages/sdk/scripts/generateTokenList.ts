import fs from 'fs'

import { fetchTokenList } from '../src/utils/token-list-utils'

fetchTokenList().then((list) => {
  fs.writeFileSync('generated/optimismTokenList.json', list)
})
