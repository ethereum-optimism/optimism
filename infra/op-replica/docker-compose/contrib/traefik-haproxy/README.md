# Overview

These files allow you to expose the l2geth RPC/WS ports to the Internet, TLS-encrypted,
via a traefik reverse proxy and Let's Encrypt Certs.

Instructions for the CloudFlare and AWS options are kept up-to-date at https://eth-docker.net/docs/Usage/ReverseProxy

Please open issues for this contribution in the fork at https://github.com/CryptoManufaktur-io/op-replica

## Usage

The `.env` in the main project directory needs to contain traefik-specific variables. Make a backup copy with
`cp .env .env.bak`, then bring in the `default.env` from this directory: `cp contrib/traefik-haproxy/default.env .env`.
Adjust replica variables to match what they were and add either `contrib/traefik-haproxy/traefik-cf.yml` or 
`contrib/traefik-haproxy/traefik-aws.yml` to `COMPOSE_FILE`.

Then edit traefik-specific variables for CloudFlare or AWS as per above-linked instructions.

Alternatively, if you have traefik running in its own stack, you can add `contrib/traefik-haproxy/ext-network.yml`
and adjust it for the network traefik runs in.

The haproxy files are examples taken from a docker swarm mode installation. They should work
with minor modifications in k8s via kompose or Portainer.

`optimism-haproxy.cfg` is the configuration file for haproxy, adjust the host and domain names you'll
use in there

`haproxy-docker-sample.yml` is an example docker-compose style deployment in docker swarm.

For example, you may have two replicas called `optimism-a.example.com` and `optimism-b.example.com`.
Both are configured in their `traefik.env` to respond to `optimism-lb.example.com`. Haproxy has
a local alias `optimism-lb.example.com` and will forward traffic to a and b servers, which know to respond
to the `optimism-lb` hostname.

`check-ecsync-optimism.sh` is an external check script to take a server out of rotation when it is not
in sync. It relies on the sequencer-health metrics being available via traefik, and assumes that
servers have `-a`, `-b`, `-c` etc suffixes. The maximum slot distance and name of the healthcheck host
without suffix are configured in the script.
