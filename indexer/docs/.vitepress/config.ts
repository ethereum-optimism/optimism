import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
	title: 'OP Stack Bridge Indexer',
	description: 'Index deposits and withdrawals from the OP Stack Bridge',
	themeConfig: {
		nav: [
			{ text: 'Home', link: '/' },
			{ text: 'API', link: '/reference/api' },
		],
		footer: {
			message: 'Released under the MIT License.',
		},
		editLink: {
			pattern: 'https://github.com/ethereum-optimism/optimism/edit/develop/indexer/docs/:path',
			text: 'Edit this page on GitHub',
		},
		sidebar: [
			{ text: 'Home', link: '/' },
			{ text: 'Configuration', link: '/api/configuration' },
			{
				text: 'api',
				collapsed: false,
				items: [
					{ text: 'Deposits', link: '/api/deposits' },
					{ text: 'Withdrawals', link: '/api/withdrawals' },
				],
			},
			{ text: 'README', link: '/README' },
		],
		socialLinks: [
			{ icon: 'github', link: 'https://github.com/ethereum-optimism/optimism' },
		],
	},
})
