import { readFile, writeFile, mkdir } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const source = JSON.parse(await readFile(resolve(root, 'packages/design-tokens/src/tokens.json'), 'utf8'))

function render(selector, platform) {
  const declarations = Object.entries(source)
    .map(([name, values]) => `  ${name}: ${values[platform]};`)
    .join('\n')
  return `${selector} {\n${declarations}\n}\n`
}

const outputs = [
  ['packages/design-tokens/src/tokens.css', render(':root', 'web')],
  ['apps/miniprogram/miniprogram/styles/tokens.wxss', render('page', 'miniprogram')],
]

for (const [relativePath, contents] of outputs) {
  const path = resolve(root, relativePath)
  await mkdir(dirname(path), { recursive: true })
  await writeFile(path, contents)
}
