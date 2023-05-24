---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "@eth-optimism/indexer"
  text: "Persistance layer for OP-Stack bridges"
  tagline: /api/v0/withdrawals/:address 
  actions:
    - theme: brand
      text: Configuration
      link: /api/configuration
    - theme: alt
      text: Deposits endpoint
      link: /api/deposits.md
    - theme: alt
      text: Withdrawals endpoint
      link: /api/withdrawals.md

features:
  - title: Easy deployment
    details: Distributed as a Docker container
  - title: Easy configurations
    details: Configure with indexer.toml.  Known chains like Base can use presets
  - title: Simple REST API 
    details: Offers endpoints to retrieve deposits and withdrawals
---
