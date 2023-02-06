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
        link: '/docs/protocol/',
      },      
      {
        text: 'Support',
        link: '/docs/biz/'        
      },
      {
        text: 'Security',
        link: '/docs/security-model/',
      },
      {
        text: 'Dev Docs',
        link: '/docs/developers/',
      },
      {
        text: "Governance",
        link: "/docs/governance/"
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
      '/docs/governance/': [
        [
          '/docs/governance/',
          'What is the Optimism Collective?'
        ],
        [
          'https://www.optimism.io/vision',
          'The Optimistic Vision'
        ],
        {
          title: "OP Holders",
          children: [
            '/docs/governance/howto-delegate.md',
            '/docs/governance/economics.md',
            '/docs/governance/allocations.md',             
            '/docs/governance/airdrop-1.md'    
          ],
          collapsable: true,
        },                
        {
          title: "Delegates",
          children: [          
                '/docs/governance/delegate.md',
                '/docs/governance/existing-delegate.md',                
                '/docs/governance/delegate-info.md',
          ],
          collapsable: true,
        },
        {
          title: "Proposal Submitters",
          children: [
            '/docs/governance/proposals.md',
            [
              'https://gov.optimism.io/tags/c/proposals/38/passed',
              'Passed Proposals'
            ]
          ],
          collapsable: true,
        },
        {
          title: "Token House Governance",
          children: [
            '/docs/governance/token-house.md',
            '/docs/governance/token-house-history.md',
            '/docs/governance/gov-fund.md', 
            [
              "https://docs.google.com/spreadsheets/d/1Ul8iMTsOFUKUmqz6MK0zpgt8Ki8tFtoWKGlwXj-Op34",
              "Governance Fund Tracker"
            ],            
            [
              'https://gov.optimism.io/t/working-constitution-of-the-optimism-collective/55',
              "Working Constitution"
            ],
            [
              'https://github.com/ethereum-optimism/OPerating-manual/blob/main/manual.md',
              'Operating Manual'
            ]
          ],
          collapsable: true,
        },  
        {
          title: "Citizen House Governance",
          children: [
            '/docs/governance/citizens-house.md',
            '/docs/governance/citizenship.md',
            '/docs/governance/retropgf-2.md',                        
          ],
          collapsable: true
        },
        [
          'https://calendar.google.com/calendar/u/0?cid=Y19mbm10Z3VoNm5vbzZxZ2JuaTJncGVyaWQ0a0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t',
          'Governance Calendar'
        ],
        '/docs/governance/attestation-station.md' 
      ],
      '/docs/security-model/': [
        '/docs/security-model/optimism-security-model.md',
        '/docs/security-model/bounties.md',
        [
          'https://medium.com/ethereum-optimism/our-pragmatic-path-to-decentralization-cb5805ca43c1',
          'Decentralizing Optimism'
        ]
      ],
      '/docs/protocol/': [
        {
          title: 'How Optimism Works',
          children: [
            '/docs/protocol/1-design-philosophy.md',
            '/docs/protocol/2-rollup-protocol.md',
          ],
          collapsable: false,          
        },
        {
          title: 'Protocol Specs',
          children: [
            '/docs/protocol/protocol-2.0.md',
            '/docs/protocol/compressed-ctc.md'
          ],
          collapsable: false,  
        }
      ],
      '/docs/developers/': [
        '/docs/developers/releases.md',
        {
          title: 'Building on Optimism',
          children: [
            '/docs/developers/build/using-tools.md',
            '/docs/developers/build/transaction-fees.md',
            '/docs/developers/build/system-contracts.md',
            '/docs/developers/build/dev-node.md',
            '/docs/developers/build/run-a-node.md',
            '/docs/developers/build/differences.md',
            '/docs/developers/build/json-rpc.md',
            '/docs/developers/build/testing-dapps.md',
            '/docs/developers/build/cheap-dapp.md'

          ],
        },
        {
          title: 'Bridging L1 and L2',
          children: [
            '/docs/developers/bridge/basics.md',
            '/docs/developers/bridge/standard-bridge.md',
            '/docs/developers/bridge/messaging.md',
            '/docs/developers/bridge/comm-strategies.md'
          ],
          collapsable: true,
        },
        '/docs/developers/known-issues.md',
        { 
          title: "Useful Tools",
          children: [
            '/docs/useful-tools/networks.md',
            '/docs/useful-tools/debugging.md',
            '/docs/useful-tools/faucets.md',
            '/docs/useful-tools/monitoring.md',
            '/docs/useful-tools/explorers.md',
            '/docs/useful-tools/providers.md',
            '/docs/useful-tools/oracles.md',
            '/docs/useful-tools/meta-tx.md',
            ['https://www.optimism.io/apps/tools', 'Third Party Tools'],
          ],
        },
        {
          title: "SDK",
          children: [
            '/docs/sdk/js-client.md',
            [
              'https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts/docs',
              'Optimism Contracts'
            ],
            '/docs/sdk/alchemy-sdk.md'           
          ]
        },
        '/docs/developers/media.md',
        {
          title: "OP Stack: Bedrock",
          children: [
            '/docs/developers/bedrock/explainer.md',
            '/docs/developers/bedrock/differences.md',
            '/docs/developers/bedrock/public-testnets.md',
            '/docs/developers/bedrock/node-operator-guide.md',
            '/docs/developers/bedrock/upgrade-guide.md',
            '/docs/developers/bedrock/metrics.md'
          ]
        }
      ],
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
