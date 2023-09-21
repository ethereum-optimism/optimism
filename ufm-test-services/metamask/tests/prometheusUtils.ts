import 'dotenv/config'
import { z } from 'zod'
import { Counter, Pushgateway } from 'prom-client'

const env = z
  .object({
    PROMETHEUS_SERVER_URL: z.string().url(),
    PROMETHEUS_PUSHGATEWAY_URL: z.string().url(),
  })
  .parse(process.env)

const txSuccessMetricName = 'metamask_tx_success'
const txFailureMetricName = 'metamask_tx_failuree'

const txSuccessCounter = new Counter({
  name: txSuccessMetricName,
  help: 'A counter signifying the number of successful transactions sent with Metamask since last failure',
})
const txFailureCounter = new Counter({
  name: txFailureMetricName,
  help: 'A counter signifying the number of failed transactions sent with Metamask since last successful transaction',
})

export const getMetamaskTxCounterValue = async (isSuccess: boolean) => {
  const metricName = isSuccess ? txSuccessMetricName : txFailureMetricName
  const prometheusMetricQuery = `${env.PROMETHEUS_SERVER_URL}/api/v1/query?query=${metricName}`

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
  //       __name__: 'metamask_tx_success',
  //       exported_job: 'metamask_tx_count',
  //       instance: 'pushgateway:9091',
  //       job: 'pushgateway'
  //     },
  //     value: [ 1695250414.474, '0' ]
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
      console.warn(`No data found for metric ${metricName} in Prometheus`)
      return undefined
    }

    throw error
  }
}

export const setMetamaskTxCounter = async (
  isSuccess: boolean,
  valueToSetTo: number
) => {
  const metricName = isSuccess ? txSuccessMetricName : txFailureMetricName
  const txCounter = isSuccess ? txSuccessCounter : txFailureCounter

  txCounter.reset()
  console.log(`Setting ${metricName} to ${valueToSetTo}`)
  txCounter.inc(valueToSetTo)

  const pushGateway = new Pushgateway(env.PROMETHEUS_PUSHGATEWAY_URL)
  await pushGateway.pushAdd({ jobName: 'metamask_tx_count' })
}

export const incrementMetamaskTxCounter = async (isSuccess: boolean) => {
  const metricName = isSuccess ? txSuccessMetricName : txFailureMetricName
  const currentMetricValue = (await getMetamaskTxCounterValue(true)) ?? 0
  console.log(
    `Current value of ${metricName} is ${currentMetricValue}, incrementing to ${
      currentMetricValue + 1
    }`
  )
  await setMetamaskTxCounter(isSuccess, currentMetricValue + 1)
}
