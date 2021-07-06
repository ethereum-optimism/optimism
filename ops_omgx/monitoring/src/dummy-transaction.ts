import dotenv from 'dotenv'
dotenv.config()
import { fastOnRamp } from './3_fast-onramp'
import { fastExit } from './4_fast-exit'
import logger from './logger'

const dummyDelayMins = parseInt(process.env.DUMMY_DELAY_MINS, 10) || 5
const delayTime = 1000 * 60 * dummyDelayMins

fastOnRamp().catch((err) => {
  logger.error(err.message)
})
.then(fastExit).catch((err) => {
  logger.error(err.message)
})

setInterval(() => {
  fastOnRamp().catch((err) => {
    logger.error(err.message)
  }).then(fastExit).catch((err) => {
    logger.error(err.message)
  })
}, delayTime)
