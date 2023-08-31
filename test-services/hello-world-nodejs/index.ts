import { Counter, register, Pushgateway } from 'prom-client';

// Create a new counter
const counter = new Counter({
  name: 'nodejs_counter',
  help: 'Description of my counter',
});

// Increment the counter
counter.inc();

// Register the metric
register.registerMetric(counter);

// Push the metric to the Pushgateway
const pushGateway = new Pushgateway('http://pushgateway:9091', undefined, register);

pushGateway.pushAdd({ jobName: 'job_name', groupings: { instance: 'localhost' }});
