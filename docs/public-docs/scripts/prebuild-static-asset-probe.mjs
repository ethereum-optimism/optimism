import { mkdir, writeFile } from "node:fs/promises"
import path from "node:path"

const outputPath = path.join(
  process.cwd(),
  "public",
  "mintlify-prebuild-probe.png"
)

const onePixelPng = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
  "base64"
)

await mkdir(path.dirname(outputPath), { recursive: true })
await writeFile(outputPath, onePixelPng)

console.log(`Generated ${outputPath}`)
