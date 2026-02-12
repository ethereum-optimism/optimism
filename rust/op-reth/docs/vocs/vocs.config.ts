import { defineConfig } from 'vocs'
import { sidebar } from './sidebar'

export default defineConfig({
  title: 'op-reth',
  iconUrl: '/logo.png',
  sidebar,
  search: {
    fuzzy: true
  },
  topNav: [
    { text: 'Run', link: '/run/opstack' },
    { text: 'CLI', link: '/cli/op-reth' },
    { text: 'GitHub', link: 'https://github.com/ethereum-optimism/optimism/tree/develop/op-reth' },
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
    pattern: "https://github.com/ethereum-optimism/optimism/edit/develop/op-reth/docs/vocs/docs/pages/:path",
  },
})
