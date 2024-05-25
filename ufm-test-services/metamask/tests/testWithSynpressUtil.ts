import 'dotenv/config'
import {
	type BrowserContext,
	chromium,
	expect,
	test as base,
} from '@playwright/test'
import metamask from '@synthetixio/synpress/commands/metamask.js'
import helpers from '@synthetixio/synpress/helpers.js'

const { initialSetup } = metamask
const { prepareMetamask } = helpers

export const testWithSynpress = base.extend<{
	context: BrowserContext
}>({
	context: async ({}, use) => {
		// required for synpress
		global.expect = expect
		// download metamask
		const metamaskPath = await prepareMetamask(
			process.env.METAMASK_VERSION || '10.25.0',
		)
		// prepare browser args
		const browserArgs = [
			`--disable-extensions-except=${metamaskPath}`,
			`--load-extension=${metamaskPath}`,
			'--remote-debugging-port=9222',
		]
		if (process.env.CI) {
			browserArgs.push('--disable-gpu')
		}
		if (process.env.HEADLESS_MODE) {
			browserArgs.push('--headless=new')
		}
		// launch browser
		const context = await chromium.launchPersistentContext('', {
			headless: false,
			args: browserArgs,
		})
		// wait for metamask
		await context.pages()[0].waitForTimeout(3000)
		// setup metamask
		await initialSetup(chromium, {
			secretWordsOrPrivateKey: process.env.METAMASK_SECRET_WORDS_OR_PRIVATEKEY,
			network: process.env.METAMASK_NETWORK,
			password: process.env.METAMASK_PASSWORD,
			enableAdvancedSettings: true,
		})
		await use(context)
	},
})

export { expect }
