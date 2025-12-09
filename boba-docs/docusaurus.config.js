// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import {themes as prismThemes} from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Boba Docs',
  tagline: 'For boba users and developers',
  favicon: 'img/boba_B.ico',

  // Set the production url of your site here
  url: 'https://docs.boba.network',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'facebook', // Usually your GitHub org/user name.
  projectName: 'docusaurus', // Usually your repo name.

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },
  themes: ['@docusaurus/theme-mermaid'],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          path: 'user-docs',
          routeBasePath: 'user-docs',
          sidebarPath: undefined, //'./sidebars.js',
          versions: {
            current: {
              label: 'v1.0.0',
            },
          },
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   'https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/',
        },
        blog: {
          showReadingTime: true,
          feedOptions: {
            type: ['rss', 'atom'],
            xslt: true,
          },
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          // editUrl:
          //   'https://github.com/facebook/docusaurus/tree/main/packages/create-docusaurus/templates/shared/',
          // Useful options to enforce blogging best practices
          onInlineTags: 'warn',
          onInlineAuthors: 'warn',
          onUntruncatedBlogPosts: 'warn',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  plugins: [
    [
      '@docusaurus/plugin-content-docs',
      {
        id: 'dev',
        path: 'dev-docs',
        routeBasePath: '/', // when implementing landing page, change this back to dev-docs
        sidebarPath: undefined, //require.resolve('./sidebarsSDK.js'),
        lastVersion: 'current',
          versions: {
            current: {
              label: 'v1.0.0',
            },
          },
      },
    ],

    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            to: '/dev-docs',
            from: '/',
          },
        ],
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: 'img/boba_preview.jpg',
      navbar: {
        title: 'Boba Docs',
        logo: {
          alt: 'For all your Boba needs',
          src: 'img/boba_B.png',
        },
        items: [
          { to: '/', label: 'Developer', position: 'left' }, //if implementing landing page, change back to dev-docs/index
          { to: 'user-docs/index', label: 'User', position: 'left' },
          // {
          //   type: 'docSidebar',
          //   sidebarId: 'tutorialSidebar',
          //   position: 'left',
          //   label: 'Docs',
          // },
          // // {to: '/blog', label: 'Blog', position: 'left'},
          // {
          //   href: 'https://github.com/bobanetwork',
          //   label: 'GitHub',
          //   position: 'right',
          // },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Community',
            items: [
              {
                label: 'Discord',
                href: 'https://discord.com/invite/Hvu3zpFwWd',
              },
              {
                label: 'X',
                href: 'https://x.com/bobanetwork',
              },
              {
                label: 'Reddit',
                href: 'https://www.reddit.com/r/bobanetwork/',
              },
              {
                label: 'Github',
                href: 'https://github.com/bobanetwork',
              },
            ],
          },
          // {
          //   title: 'More',
          //   items: [
          //     {
          //       label: 'Blog',
          //       to: '/blog',
          //     },

          //   ],
          // },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} My Project, Inc. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

export default config;
