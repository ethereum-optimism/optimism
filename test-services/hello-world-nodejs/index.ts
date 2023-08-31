import Axios from 'axios'
import { Counter, register, Pushgateway } from 'prom-client';

const metricName = 'nodejs_counter';
const prometheusServerURL = 'http://prometheus:9090';
const pushGatewayURL = 'http://pushgateway:9091';

async function getMetricValueFromPrometheus(metricName: string) {
  try {
      const response = await Axios.get(`${prometheusServerURL}/api/v1/query?query=${metricName}`);
      const data = response.data;

      if (data && data.data && data.data.result && data.data.result.length > 0) {
          return parseFloat(data.data.result[0].value[1]);
      } else {
          console.warn(`No data found for metric ${metricName} in Prometheus.`);
          return 0;
      }
  } catch (error) {
      console.error(`Error querying Prometheus for metric ${metricName}:`, error);
      return 0;
  }
}


async function main() {
    const currentValue = await getMetricValueFromPrometheus(metricName);

    // Initialize the counter with the current value from Prometheus
    const counter = new Counter({
        name: metricName,
        help: 'Description of the counter'
    });

    // Increment the counter
    counter.inc(currentValue + 1);

    register.registerMetric(counter);

    const pushGateway = new Pushgateway(pushGatewayURL, undefined, register);

    // Push the updated metric to the Push Gateway
    pushGateway.pushAdd({ jobName: 'job_name', groupings: { instance: 'localhost' }});
}

main();
