const { description } = require('../../package')
const path = require('path')

module.exports = {
  title: 'OP Stack Docs',
  description: description,
  head: [
    ['link', { rel: 'manifest', href: '/manifest.json' }],
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
    ['meta', { name: 'apple-mobile-web-app-capable', content: 'yes' }],
    ['meta', { name: 'apple-mobile-web-app-status-bar-style', content: 'black' }],
    ['meta', { property: 'og:image', content: 'https://stack.optimism.io/assets/logos/twitter-logo.png' }],
    ['meta', { name: 'twitter:image', content: 'https://stack.optimism.io/assets/logos/twitter-logo.png' }],
    ['meta', { name: 'twitter:title', content: 'OP Stack Docs' }],
    ['meta', { property: 'og:title', content: 'OP Stack Docs' }],
    ['meta', { name: 'twitter:card', content: 'summary' } ],
    ['link', { rel: "icon", type: "image/png", sizes: "32x32", href: "/assets/logos/favicon.png"}],
  ],
  theme: path.resolve(__dirname, './theme'),
  themeConfig: {
    "twitter:card": "summary",
    contributor: false,
    hostname: 'https://stack.optimism.io',
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
      appId: '7Q6XITDI0Z',
      apiKey: '9d55a31a04b210cd26f97deabd161705',
      indexName: 'optimism'
    },
    nav: [
      {
        text: 'Home',
        link: 'https://www.optimism.io/'
      },
      {
        text: 'OP Stack Docs',
        link: '/'
      },
      {
        text: 'Optimism Docs',
        link: 'https://community.optimism.io/'
      },
      {
        text: 'Governance',
        link: 'https://community.optimism.io/docs/governance/'
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
    sidebar: [
      {
        title: "OP Stack",
        collapsable: false,
        children: [        
          '/',
          [
            '/docs/understand/design-principles.md',
            'Design Principles'
          ],
          '/docs/understand/landscape.md',
          '/docs/understand/explainer.md'
        ]
      }, 
      {
        title: "Releases",
        collapsable: false,
        children: [
          '/docs/releases/',
          {
            title: "Bedrock",
            collapsable: true,
            children: [
              '/docs/releases/bedrock/',
              '/docs/releases/bedrock/explainer.md',
              '/docs/releases/bedrock/differences.md'
            ]
          }
        ]
      },
      {
        title: "Building OP Stack Rollups",
        collapsable: false,
        children: [
          '/docs/build/getting-started.md',
          '/docs/build/conf.md',
          '/docs/build/explorer.md',
          {
            title: "OP Stack Hacks",
            collapsable: true,
            children: [
              '/docs/build/hacks.md',
              '/docs/build/featured.md',
              '/docs/build/data-avail.md',
              '/docs/build/derivation.md',
              '/docs/build/execution.md',
              '/docs/build/settlement.md',
              {
                title: "Sample Hacks",
                children: [
                  "/docs/build/tutorials/add-attr.md",
                  "/docs/build/tutorials/new-precomp.md",                
                ]
              }  // End of tutorials                      
            ], 
          },    // End of OP Stack hacks
        ],
      },      // End of Building OP Stack Rollups
      {
        title: "Contributing",
        collapsable: false,
        children: [
          '/docs/contribute.md',
        ]
      },
      {
        title: "Security",
        collapsable: false,
        children: [
          '/docs/security/faq.md',
          '/docs/security/policy.md',
        ]
      },        
    ],  // end of sidebar
  plugins: [
    "@vuepress/pwa",
    [
      '@vuepress/plugin-medium-zoom',
      {
        selector: ':not(a) > img'
      }
    ],
    "plausible-analytics"
  ]
}
}

// module.exports.themeConfig.sidebar["/docs/useful-tools/"] = module.exports.themeConfig.sidebar["/docs/developers/"]
