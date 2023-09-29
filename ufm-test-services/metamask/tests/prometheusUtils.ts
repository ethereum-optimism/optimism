import 'dotenv/config'
import { z } from 'zod'
import { Gauge, Pushgateway } from 'prom-client'

const env = z
  .object({
    PROMETHEUS_SERVER_URL: z.string().url(),
    PROMETHEUS_PUSHGATEWAY_URL: z.string().url(),
  })
  .parse(process.env)

const selfSendTransactionMetricName = 'metamask_self_send'

const selfSendGauge = new Gauge({
  name: selfSendTransactionMetricName,
  help: 'A gauge signifying the number of transactions sent with Metamask',
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
  console.log(`Setting ${selfSendTransactionMetricName} to ${valueToSetTo}`)
  selfSendGauge.set(valueToSetTo)

  const pushGateway = new Pushgateway(env.PROMETHEUS_PUSHGATEWAY_URL)
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
    `Current value of ${selfSendTransactionMetricName} is ${currentMetricValue}, incrementing to ${newMetricValue}`
  )
  await setSelfSendTxGauge(newMetricValue)
}
