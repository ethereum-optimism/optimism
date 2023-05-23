---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Optimism Bridge Indexer"
  text: "Service for indexing and retrieving data related to the optimism bridge"
  tagline: Proof of concept for base
  actions:
    - theme: brand
      text: Configuration
      link: api/configuration
    - theme: alt
      text: Deposits endpoint
      link: api/deposits.md
    - theme: alt
      text: Withdrawals endpoint
      link: api/withdrawals.md

features:
  - title: Easy configurations
    details: Configure with indexer.toml.  Known chains like Base can use presets
  - title: Easy to deploy
    details: Distributed as a Docker container
  - title: Simple API for bridging needs
    details: Offers endpoints to retrieve deposits and withdrawals
---
