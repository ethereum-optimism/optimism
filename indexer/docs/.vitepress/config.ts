import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
	title: 'EVMts docs',
	description: 'Execute solidity scripts in the browser',
	themeConfig: {
		// https://vitepress.dev/reference/default-theme-config
		nav: [
			{ text: 'Home', link: '/' },
			{ text: 'API', link: '/reference/api' },
		],
		footer: {
			message: 'Released under the MIT License.',
		},
		editLink: {
			pattern: 'https://github.com/evmts/evmts-monorepo/edit/main/docs/:path',
			text: 'Edit this page on GitHub',
		},
		sidebar: [
			{
				text: 'Introduction',
				link: '/introduction/intro',
				items: [
					{ text: 'Home', link: '/' },
					{ text: 'Why EVMts', link: '/introduction/intro' },
					{ text: 'Get started', link: '/introduction/get-started' },
					{ text: 'Quick start', link: '/introduction/quick-start' },
					{ text: 'Installation', link: '/introduction/installation' },
					{
						text: 'Plugin configuration',
						link: '/introduction/plugin-configuration',
					},
					{ text: 'Hello world', link: '/introduction/hello-world' },
				],
			},
			{
				text: 'EVMts Core',
				items: [
					{
						text: 'Clients and Transports',
						items: [
							{ text: 'PublicClient', link: '/reference/public-client' },
							{ text: 'WalletClient', link: '/reference/wallet-client' },
							{ text: 'HttpFork', link: '/reference/http-fork' },
						],
					},
					{
						text: 'Contracts and Scripts',
						items: [
							{
								text: 'Script',
								link: '/reference/script',
							},
							{ text: 'Contract', link: '/reference/contract' },
							{ text: 'HttpFork', link: '/reference/http-fork' },
						],
					},
				],
			},
			{
				text: 'EVMts Build Plugins',
				collapsed: true,
				items: [
					{ text: 'Typescript Plugin', link: '/plugin-reference/typescript' },
					{
						text: 'Rollup Plugin',
						link: '/plugin-reference/rollup',
						collapsed: true,
						items: [{ text: 'Forge', link: '/plugin-reference/forge' }],
					},
					{ text: 'Webpack', link: '/plugin-reference/webpack' },
				],
			},
			{
				text: 'Guides',
				collapsed: true,
				items: [
					{ text: 'Configuring configuring forge', link: '/guide/forge' },
					{ text: 'Writing solidity scripts', link: '/guide/scripting' },
					{ text: 'Testing scripts', link: '/guide/testing' },
				],
			},
		],
		socialLinks: [
			{ icon: 'github', link: 'https://github.com/evmts/evmts-monorepo' },
			{ icon: 'twitter', link: 'https://twitter.com/FUCORY' },
		],
	},
})
