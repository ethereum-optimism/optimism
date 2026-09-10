#!/usr/bin/env tsx

import { writeFileSync, readFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

// Get the directory name of the current module
const __dirname = dirname(fileURLToPath(import.meta.url))

// Function to get current version from package.json
function getCurrentVersion(): string {
  const packageJsonPath = resolve(
    __dirname,
    '../packages/supersim/package.json',
  )
  const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf-8'))
  return packageJson.version
}

// Function to get the latest release version from GitHub
async function getLatestVersion(): Promise<string> {
  try {
    const response = await fetch(
      'https://api.github.com/repos/ethereum-optimism/supersim/releases/latest',
    )
    if (!response.ok) {
      throw new Error(`Failed to fetch latest release: ${response.statusText}`)
    }
    const data = await response.json()
    return data.tag_name
  } catch (error) {
    console.error('Error fetching latest release:', error)
    process.exit(1)
  }
}

// Function to check if a specific release version exists on GitHub
async function checkVersionExists(version: string): Promise<boolean> {
  const testUrl = `https://github.com/ethereum-optimism/supersim/releases/download/${version}/supersim_Darwin_arm64.tar.gz`

  try {
    const response = await fetch(testUrl)
    return response.status === 200
  } catch (error) {
    console.error(
      `❌ Version ${version} does not exist on GitHub releases`,
      error,
    )
    return false
  }
}

async function updateSupersimVersion(version: string) {
  const currentVersion = getCurrentVersion()
  if (currentVersion === version) {
    console.log(`✨ Already at version ${version}, no update needed`)
    return
  }

  // First check if the version exists
  console.log(`Checking if version ${version} exists...`)
  const exists = await checkVersionExists(version)
  if (!exists) {
    console.error(`❌ Version ${version} does not exist on GitHub releases`)
    process.exit(1)
  }

  // 1. Update install.js
  const installJsPath = resolve(__dirname, '../packages/supersim/install.js')
  let installJsContent = readFileSync(installJsPath, 'utf-8')

  installJsContent = installJsContent.replace(
    /const SUPERSIM_VERSION = ['"].*['"]/,
    `const SUPERSIM_VERSION = '${version}'`,
  )

  writeFileSync(installJsPath, installJsContent)

  // 2. Update package.json version
  const packageJsonPath = resolve(
    __dirname,
    '../packages/supersim/package.json',
  )
  const packageJson = JSON.parse(readFileSync(packageJsonPath, 'utf-8'))
  packageJson.version = version
  writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, '  ') + '\n')

  // 3. Update mise.toml
  const miseTomlPath = resolve(__dirname, '../mise.toml')
  let miseTomlContent = readFileSync(miseTomlPath, 'utf-8')

  miseTomlContent = miseTomlContent.replace(
    /^"ubi:ethereum-optimism\/supersim"=.*$/m,
    `"ubi:ethereum-optimism/supersim"="${version}"`,
  )

  writeFileSync(miseTomlPath, miseTomlContent)

  console.log(
    `✅ Updated supersim version from ${currentVersion} to ${version} in:`,
  )
  console.log(`  - install.js`)
  console.log(`  - package.json`)
  console.log(`  - mise.toml`)
}

// Get version from command line argument or fetch latest
const version = process.argv[2]
;(async () => {
  try {
    let targetVersion: string
    if (!version) {
      console.log('No version specified, fetching latest release...')
      targetVersion = await getLatestVersion()
      console.log(`Latest version is ${targetVersion}`)
    } else {
      targetVersion = version
    }
    await updateSupersimVersion(targetVersion)
  } catch (err) {
    console.error('Error updating supersim version:', err)
    process.exit(1)
  }
})()
