import { testWithSynpress } from './testWithSynpress'
import { test } from '@playwright/test'

testWithSynpress('should be able to read', async ({ page }) => {
	await page.goto('http://localhost:9011')
})
