import 'dotenv/config'
import { z } from 'zod'
import { Gauge, Pushgateway, Registry } from 'prom-client'

const env = z
  .object({
    PROMETHEUS_SERVER_URL: z.string().url(),
    PROMETHEUS_PUSHGATEWAY_URL: z.string().url(),
  })
  .parse(process.env)

const selfSendTransactionMetricName = 'metamask_self_send'
const feeEstimateLowMetricName = 'metamask_self_send_fee_estimation_low'
const feeEstimateMediumMetricName = 'metamask_self_send_fee_estimation_medium'
const feeEstimateHighMetricName = 'metamask_self_send_fee_estimation_high'
const feeEstimateActualMetricName = 'metamask_self_send_fee_estimation_actual'

const selfSendRegistry = new Registry()
const feeEstimateLowRegistry = new Registry()
const feeEstimateMediumRegistry = new Registry()
const feeEstimateHighRegistry = new Registry()
const feeEstimateActualRegistry = new Registry()

const selfSendGauge = new Gauge({
  name: selfSendTransactionMetricName,
  help: 'A gauge signifying the number of transactions sent with Metamask',
  registers: [selfSendRegistry]
})
const feeEstimateLowGauge = new Gauge({
  name: feeEstimateLowMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for Low transaction speed',
  registers: [feeEstimateLowRegistry]
})
const feeEstimateMediumGauge = new Gauge({
  name: feeEstimateMediumMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for Medium transaction speed',
  registers: [feeEstimateMediumRegistry]
})
const feeEstimateHighGauge = new Gauge({
  name: feeEstimateHighMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for High transaction speed',
  registers: [feeEstimateHighRegistry]
})
const feeEstimateActualGauge = new Gauge({
  name: feeEstimateActualMetricName,
  help: 'A gauge signifying the latest actual transaction fee',
  registers: [feeEstimateActualRegistry]
})

export const getSelfSendGaugeValue = async () => {
  const prometheusMetricQuery = `${env.PROMETHEUS_SERVER_URL}/api/v1/query?query=${selfSendTransactionMetricName}`

  const response = await fetch(prometheusMetricQuery)
  if (!response.ok) {
    console.error(response.status)
    console.error(response.statusText)
    throw new Error(`Failed to fetch metric from: ${prometheusMetricQuery}`)
  }

  // The following is an example of the expect response from prometheusMetricQuery
  // for response.json().data.result[0]:
  // [
  //   {
  //     metric: {
  //       __name__: 'metamask_self_send',
  //       exported_job: 'metamask_self_send_tx_count',
  //       instance: 'pushgateway:9091',
  //       job: 'pushgateway'
  //     },
  //     value: [ 1695847795.646, '-1' ]
  //   }
  // ]
  try {
    const responseJson = z
      .object({
        data: z.object({
          result: z.array(
            z.object({
              value: z.tuple([
                z.number(),
                z.number().or(z.string().transform((value) => parseInt(value))),
              ]),
            })
          ),
        }),
      })
      .parse(await response.json())

    return responseJson.data.result[0].value[1]
  } catch (error) {
    if (
      error.message === "Cannot read properties of undefined (reading 'value')"
    ) {
      console.warn(
        `No data found for metric ${selfSendTransactionMetricName} in Prometheus`
      )
      return undefined
    }

    throw error
  }
}

export const setSelfSendTxGauge = async (valueToSetTo: number) => {
  console.log(`Setting ${selfSendTransactionMetricName} to ${valueToSetTo}...`)
  selfSendGauge.set(valueToSetTo)

  const pushGateway = new Pushgateway(env.PROMETHEUS_PUSHGATEWAY_URL, undefined, selfSendRegistry)
  await pushGateway.pushAdd({ jobName: 'metamask_self_send_tx_count' })
}

export const incrementSelfSendTxGauge = async (isSuccess: boolean) => {
  const currentMetricValue = (await getSelfSendGaugeValue()) ?? 0

  let newMetricValue: number
  if (isSuccess) {
    newMetricValue = currentMetricValue >= 0 ? currentMetricValue + 1 : 1
  } else {
    newMetricValue = currentMetricValue < 0 ? currentMetricValue - 1 : -1
  }

  console.log(
    `Current value of ${selfSendTransactionMetricName} is ${currentMetricValue}, incrementing to ${newMetricValue}...`
  )
  await setSelfSendTxGauge(newMetricValue)
}

export const setFeeEstimationGauge = async (txSpeed: 'low' | 'medium' | 'high' | 'actual', fee: number) => {
  console.log(
    txSpeed !== 'actual'
    ? `Setting Metamask fee estimation for ${txSpeed} to ${fee}...`
    : `Setting actual transaction fee to ${fee}`
  )

  let prometheusRegistry: Registry
  switch (txSpeed) {
    case 'low':
      feeEstimateLowGauge.set(fee)
      prometheusRegistry = feeEstimateLowRegistry
      break;
    case 'medium':
      feeEstimateMediumGauge.set(fee)
      prometheusRegistry = feeEstimateMediumRegistry
      break;
    case 'high':
      feeEstimateHighGauge.set(fee)
      prometheusRegistry = feeEstimateHighRegistry
      break;
    case 'actual':
      feeEstimateActualGauge.set(fee)
      prometheusRegistry = feeEstimateActualRegistry
      break;
    default:
      throw new Error(`unsupported transaction speed given: ${txSpeed}`)
  }


  const pushGateway = new Pushgateway(env.PROMETHEUS_PUSHGATEWAY_URL, undefined, prometheusRegistry)
  await pushGateway.pushAdd({ jobName: `metamask_self_send_tx_fee_estimation_${txSpeed}` })
}
