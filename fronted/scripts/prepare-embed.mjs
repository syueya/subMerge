import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const projectDir = path.dirname(path.dirname(fileURLToPath(import.meta.url)))
const outputDir = path.resolve(projectDir, "../backend/internal/webui/dist")

fs.mkdirSync(outputDir, { recursive: true })
for (const entry of fs.readdirSync(outputDir)) {
  if (entry === "README.txt") continue
  fs.rmSync(path.join(outputDir, entry), { recursive: true, force: true })
}
