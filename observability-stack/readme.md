# Observability Stack

Prometheus scrapers and Grafana dashboards powering observability for OP Stack
services driven by Flashbots block building.

The stack emphasises providing insights into the health and behavior of
block building and sequencing. Alerting based on these metrics will be added in
the future.

## Dashboards

- [x] L2<>L2 Proposer (aka `op-geth`)
- [ ] L2 Builder (aka `builder-op-geth`)

## Usage

1. `cp .env.example .env` and set your desired Grafana password

2. Override job ports in `prometheus.yml` if required (prom doesn't support env)

3. Start the services `docker compose up --build -d`

4. Visit `http://localhost:3000`, login with username `admin` and password
set in `.env`
