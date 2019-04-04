import { NestdFactory } from '@nestd/core'

import './setup'
import { CoreAppModule } from '../src/app/core/app.module'

const loop = () => {
  setTimeout(() => {
    loop()
  }, 100)
}

const main = async () => {
  const app = await NestdFactory.create(CoreAppModule)
  await app.start()
}

main()
