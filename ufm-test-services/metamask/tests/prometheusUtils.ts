import 'dotenv/config'
import { z } from 'zod'
import { Gauge, Pushgateway, Registry } from 'prom-client'

const env = z
  .object({
    METRICS_READ_URL: z.string().url(),
    METRICS_READ_USERNAME: z.string().optional(),
    METRICS_READ_PASSWORD: z.string().optional(),
    METRICS_WRITE_URL: z.string().url(),
    METRICS_WRITE_TOOL: z.enum(['grafana', 'prometheus-pushgateway']),
    METRICS_WRITE_SOURCE: z.string().optional(),
    METRICS_WRITE_USERNAME: z.string().optional(),
    METRICS_WRITE_PASSWORD: z.string().optional(),
  })
  .refine(
    (data) => {
      if (
        (data.METRICS_READ_USERNAME && !data.METRICS_READ_PASSWORD) ||
        (data.METRICS_READ_PASSWORD && !data.METRICS_READ_USERNAME)
      ) {
        return false
      }

      if (
        (data.METRICS_WRITE_USERNAME && !data.METRICS_WRITE_PASSWORD) ||
        (data.METRICS_WRITE_PASSWORD && !data.METRICS_WRITE_USERNAME)
      ) {
        return false
      }

      return true
    },
    {
      message:
        'Both username and password must be provided together for read or write metrics',
    }
  )
  .refine(
    (data) => {
      if (
        data.METRICS_WRITE_TOOL === 'grafana' &&
        data.METRICS_WRITE_SOURCE === undefined
      )
        return false

      return true
    },
    {
      message:
        'Writing to Grafana requires a source, please specify one using METRICS_WRITE_SOURCE env',
    }
  )
  .parse(process.env)

const selfSendTransactionMetricName = 'metamask_self_send_metric'
const feeEstimateLowMetricName = 'metamask_self_send_fee_estimation_low_metric'
const feeEstimateMediumMetricName =
  'metamask_self_send_fee_estimation_medium_metric'
const feeEstimateHighMetricName =
  'metamask_self_send_fee_estimation_high_metric'
const feeEstimateActualMetricName =
  'metamask_self_send_fee_estimation_actual_metric'

const selfSendRegistry = new Registry()
const feeEstimateLowRegistry = new Registry()
const feeEstimateMediumRegistry = new Registry()
const feeEstimateHighRegistry = new Registry()
const feeEstimateActualRegistry = new Registry()

const selfSendGauge = new Gauge({
  name: selfSendTransactionMetricName,
  help: 'A gauge signifying the number of transactions sent with Metamask',
  registers: [selfSendRegistry],
})
const feeEstimateLowGauge = new Gauge({
  name: feeEstimateLowMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for Low transaction speed',
  registers: [feeEstimateLowRegistry],
})
const feeEstimateMediumGauge = new Gauge({
  name: feeEstimateMediumMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for Medium transaction speed',
  registers: [feeEstimateMediumRegistry],
})
const feeEstimateHighGauge = new Gauge({
  name: feeEstimateHighMetricName,
  help: 'A gauge signifying the latest fee estimation from Metamask for High transaction speed',
  registers: [feeEstimateHighRegistry],
})
const feeEstimateActualGauge = new Gauge({
  name: feeEstimateActualMetricName,
  help: 'A gauge signifying the latest actual transaction fee',
  registers: [feeEstimateActualRegistry],
})

const queryMetricsReadUrl = async (
  query: string = selfSendTransactionMetricName
) => {
  const metricsReadRequest = `${env.METRICS_READ_URL}?query=${query}`
  const response = await fetch(metricsReadRequest, {
    headers:
      env.METRICS_READ_USERNAME === undefined
        ? undefined
        : {
            Authorization: `Bearer ${env.METRICS_READ_USERNAME}:${env.METRICS_READ_PASSWORD}`,
          },
  })
  if (!response.ok) {
    console.error(response.status)
    console.error(response.statusText)
    throw new Error(`Failed to fetch metric from: ${metricsReadRequest}`)
  }
  return response
}

export const getSelfSendGaugeValue = async () => {
  const response = await queryMetricsReadUrl(selfSendTransactionMetricName)

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
        `No data found for metric ${selfSendTransactionMetricName} in ${env.METRICS_READ_URL}`
      )
      return undefined
    }

    throw error
  }
}

const pushMetricsGrafana = (metricName: string, valueToSetTo: number) =>
  pushMetricsWriteUrl(
    `${metricName},source=${
      env.METRICS_WRITE_SOURCE
    } metric=${valueToSetTo}`
  )

const pushMetricsPrometheusPushgateway = (registry: Registry) => {
  const pushGateway = new Pushgateway(env.METRICS_WRITE_URL, undefined, registry)
  return pushGateway.pushAdd({ jobName: 'ufm-metamask-metric-push'})
}

const pushMetricsWriteUrl = async (requestBody: string) => {
  const response = await fetch(env.METRICS_WRITE_URL, {
    method: 'POST',
    headers:
      env.METRICS_WRITE_USERNAME === undefined
        ? undefined
        : {
            Authorization: `Bearer ${env.METRICS_WRITE_USERNAME}:${env.METRICS_WRITE_PASSWORD}`,
          },
    body: requestBody,
  })
  if (!response.ok) {
    console.error(response.status)
    console.error(response.statusText)
    throw new Error(`Failed to push metric to: ${env.METRICS_WRITE_URL}`)
  }
  return response
}

export const setSelfSendTxGauge = async (valueToSetTo: number) => {
  console.log(`Setting ${selfSendTransactionMetricName} to ${valueToSetTo}...`)
  selfSendGauge.set(valueToSetTo)

  env.METRICS_WRITE_TOOL === 'grafana'
    ? await pushMetricsGrafana(selfSendTransactionMetricName.replace('_metric', ''), valueToSetTo)
    : await pushMetricsPrometheusPushgateway(selfSendRegistry)
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

export const setFeeEstimationGauge = async (
  txSpeed: 'low' | 'medium' | 'high' | 'actual',
  fee: number
) => {
  let metricNameGrafana: string
  let prometheusRegistry: Registry
  switch (txSpeed) {
    case 'low':
      feeEstimateLowGauge.set(fee)
      metricNameGrafana = feeEstimateLowMetricName
      prometheusRegistry = feeEstimateLowRegistry
      break
    case 'medium':
      feeEstimateMediumGauge.set(fee)
      metricNameGrafana = feeEstimateMediumMetricName
      prometheusRegistry = feeEstimateMediumRegistry
      break
    case 'high':
      feeEstimateHighGauge.set(fee)
      metricNameGrafana = feeEstimateHighMetricName
      prometheusRegistry = feeEstimateHighRegistry
      break
    case 'actual':
      feeEstimateActualGauge.set(fee)
      metricNameGrafana = feeEstimateActualMetricName
      prometheusRegistry = feeEstimateActualRegistry
      break
    default:
      throw new Error(`unsupported transaction speed given: ${txSpeed}`)
  }
  metricNameGrafana = metricNameGrafana.replace('_metric', '')

  console.log(`Setting ${metricNameGrafana} to ${fee}...`)

  env.METRICS_WRITE_TOOL === 'grafana'
    ? await pushMetricsGrafana(metricNameGrafana, fee)
    : await pushMetricsPrometheusPushgateway(prometheusRegistry)
}
