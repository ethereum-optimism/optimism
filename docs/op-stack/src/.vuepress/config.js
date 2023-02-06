const { description } = require('../../package')
const path = require('path')

module.exports = {
  title: 'OPStack Docs',
  description: description,

  head: [ 
    ['link', { rel: 'manifest', href: '/manifest.json' }],
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'apple-mobile-web-app-capable', content: 'yes' }],
    ['meta', { name: 'apple-mobile-web-app-status-bar-style', content: 'black' }],
    ['link', { rel: "icon", type: "image/png", sizes: "32x32", href: "/assets/logos/favicon.png"}],
  ],

//  cache: false,

  theme: path.resolve(__dirname, './theme'),
  themeConfig: {
    contributor: false,
    hostname: 'https://community.optimism.io',
    logo: '/assets/logos/logo.png',
    docsDir: 'src',
    docsRepo: 'https://github.com/ethereum-optimism/opstack-docs',
    docsBranch: 'main',
    lastUpdated: false,
    darkmode: 'disable',
    themeColor: false,
    blog: false,
    iconPrefix: 'far fa-',
    pageInfo: false,
    pwa: {
      cacheHTML: false,
    },
    activeHash: {
      offset: -200,
    },
    algolia: {
      appId: '8LQU4WGQXA',
      apiKey: '2c1a86142192f96dab9a5066ad0c1d50',
      indexName: 'optimism'
    },
    nav: [
      /* When you update here, don't forget to update the tiles
         in src/README.md */ 
      {
        text: 'How Optimism Works',
        link: '/docs/how-optimism-works/'
      },
      {
        text: 'Protocol Specs',
        link: '/docs/protocol/'
      },
      {
        text: 'Security',
        link: '/docs/security-model/',
      },
      {
        text: 'Developer Docs',
        link: '/docs/developers/',
      },
      {
        text: 'Contribute',
        link: '/docs/contribute/',
      },
      {
        text: 'Community',
        items: [
          {
            icon: 'discord',
            iconPrefix: 'fab fa-',
            iconClass: 'color-discord',
            text: 'Discord',
            link: 'https://discord.optimism.io',
          },
          {
            icon: 'github',
            iconPrefix: 'fab fa-',
            iconClass: 'color-github',
            text: 'GitHub',
            link: 'https://github.com/ethereum-optimism/optimism',
          },
          {
            icon: 'twitter',
            iconPrefix: 'fab fa-',
            iconClass: 'color-twitter',
            text: 'Twitter',
            link: 'https://twitter.com/optimismFND',
          },
          {
            icon: 'twitch',
            iconPrefix: 'fab fa-',
            iconClass: 'color-twitch',
            text: 'Twitch',
            link: 'https://www.twitch.tv/optimismpbc'
          },
          {
            icon: 'medium',
            iconPrefix: 'fab fa-',
            iconClass: 'color-medium',
            text: 'Blog',
            link: 'https://optimismpbc.medium.com/'
          },
          {
            icon: 'computer-classic',
            iconClass: 'color-ecosystem',
            text: 'Ecosystem',
            link: 'https://www.optimism.io/apps/all',
          },
          {
            icon: 'globe',
            iconClass: 'color-optimism',
            text: 'optimism.io',
            link: 'https://www.optimism.io/',
          }
        ]
      }
    ],
    searchPlaceholder: 'Search the docs',
    sidebar: {
      '/docs/how-optimism-works': [
        '/docs/how-optimism-works/design-philosophy.md',
        '/docs/how-optimism-works/rollup-protocol.md',
      ],
      '/docs/protocol/': [
        '/docs/protocol/contract-overview.md',
      ],
      '/docs/security-model/': [
        '/docs/security-model/optimism-security-model.md',
        '/docs/security-model/bounties.md',
      ],          
      '/docs/developers/': [
        {
          title: "OP Stack: Bedrock",
          children: [
            '/docs/developers/opstack/explainer.md',
            '/docs/developers/opstack/differences.md',
            '/docs/developers/opstack/public-testnets.md', 
            '/docs/developers/opstack/node-operator-guide.md',    
            '/docs/developers/opstack/upgrade-guide.md',   
            '/docs/developers/opstack/metrics.md'    
          ]
        },
        '/docs/developers/releases.md'
      ],
      '/docs/contribute/': [
        '/docs/contribute/README.md'
      ]
    }
  },

  plugins: [
    "@vuepress/pwa",
    [
      '@vuepress/plugin-medium-zoom',
      {
        // When an image is inside a link, it means we don't to expand it
        // when clicked
        selector: ':not(a) > img'
      }
    ],
    "plausible-analytics"
  ]
}
  
module.exports.themeConfig.sidebar["/docs/useful-tools/"] = module.exports.themeConfig.sidebar["/docs/developers/"]
