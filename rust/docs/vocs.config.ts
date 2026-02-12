import { defineConfig } from 'vocs'
import { sidebar } from './sidebar'

export default defineConfig({
  title: 'OP Stack Rust',
  description: 'Rust implementations for the OP Stack: Kona, op-reth, and op-alloy',
  logoUrl: '/logo.png',
  iconUrl: '/logo.png',
  sidebar,
  search: {
    fuzzy: true
  },
  topNav: [
    {
      text: 'Kona',
      items: [
        { text: 'Overview', link: '/kona/intro/overview' },
        { text: 'Run a Node', link: '/kona/node/run/overview' },
        { text: 'SDK', link: '/kona/sdk/overview' },
        { text: 'Rustdocs', link: 'https://docs.rs/kona-node/latest/' },
      ]
    },
    {
      text: 'op-reth',
      items: [
        { text: 'Overview', link: '/op-reth/' },
        { text: 'Run', link: '/op-reth/run/opstack' },
        { text: 'CLI Reference', link: '/op-reth/cli/op-reth' },
      ]
    },
    {
      text: 'op-alloy',
      items: [
        { text: 'Overview', link: '/op-alloy/intro' },
        { text: 'Getting Started', link: '/op-alloy/starting' },
        { text: 'Building', link: '/op-alloy/building' },
      ]
    },
    { text: 'GitHub', link: 'https://github.com/ethereum-optimism/optimism/tree/develop/rust' },
  ],
  socials: [
    {
      icon: 'github',
      link: 'https://github.com/ethereum-optimism/optimism',
    },
  ],
  theme: {
    accentColor: {
      light: '#ff0420',
      dark: '#ff0420',
    }
  },
  editLink: {
    pattern: "https://github.com/ethereum-optimism/optimism/edit/develop/rust/docs/:path",
  },
  sponsors: [
    {
      name: 'Supporters',
      height: 120,
      items: [
        [
          {
            name: 'OP Labs',
            link: 'https://oplabs.co',
            image: 'https://avatars.githubusercontent.com/u/109625874?s=200&v=4',
          }
        ]
      ]
    }
  ]
})
