import dotenv from 'dotenv'
dotenv.config()
import { fastOnRamp } from './3_fast-onramp'
import { fastExit } from './4_fast-exit'
import logger from './logger'

const dummyDelayMins = parseInt(process.env.DUMMY_DELAY_MINS, 10) || 5
const delayTime = 1000 * 60 * dummyDelayMins

const sleep = () => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(delayTime)
    }, delayTime)
  })
}

const doTransaction = () => {
  return fastOnRamp().catch((err) => {
    logger.error(err.message)
  })
  .then(fastExit).catch((err) => {
    logger.error(err.message)
  })
  .then(sleep).then(doTransaction)
}

doTransaction()
