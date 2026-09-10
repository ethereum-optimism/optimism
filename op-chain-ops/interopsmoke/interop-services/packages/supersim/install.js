const fs = require('fs')
const { https } = require('follow-redirects')
const path = require('path')
const os = require('os')
const { exec } = require('child_process')

const PLATFORM = os.platform()
const ARCH = os.arch()
const BIN_PATH = path.join(__dirname, 'bin')
const SUPERSIM_VERSION = '0.1.0-alpha.53'

const archiveFilename = {
  darwin: {
    arm64: 'supersim_Darwin_arm64.tar.gz',
    x64: 'supersim_Darwin_x86_64.tar.gz',
  },
  linux: {
    arm64: 'supersim_Linux_arm64.tar.gz',
    x64: 'supersim_Linux_x86_64.tar.gz',
  },
  win32: {
    arm64: 'supersim_Windows_arm64.zip',
    x64: 'supersim_Windows_x86_64.zip',
  },
}

function extractRelease(filename, outputPath) {
  const cmd = filename.endsWith('.tar.gz')
    ? `tar -xzf ${outputPath} -C ${BIN_PATH}`
    : `unzip ${outputPath} -d ${BIN_PATH}`

  exec(cmd, (err) => {
    if (err) {
      console.error('Error extracting', err)
    } else {
      console.log('Successfully extracted release')
    }
  })
}

async function main() {
  const filename = archiveFilename[PLATFORM][ARCH]
  if (!filename) {
    console.error('Unsupported platform/architecture')
    process.exit(1)
  }

  const downloadUrl = `https://github.com/ethereum-optimism/supersim/releases/download/${SUPERSIM_VERSION}/${filename}`
  const outputPath = path.join(BIN_PATH, filename)

  if (!fs.existsSync(BIN_PATH)) {
    fs.mkdirSync(BIN_PATH)
  }

  console.log(`Attempting to fetch ${filename}`)

  https
    .get(downloadUrl, (res) => {
      const fileStream = fs.createWriteStream(outputPath)
      res.pipe(fileStream)

      fileStream.on('finish', () => {
        console.log(`Downloaded ${filename}`)
        fileStream.on('close', () => extractRelease(filename, outputPath))
      })
    })
    .on('error', (err) => {
      console.error('Error downloading supersim', err)
      process.exit(1)
    })
}

;(async () => {
  try {
    await main()
  } catch (err) {
    console.error('Error setting up supersim', err)
    process.exit(1)
  }
})()
