import { copyFile, mkdir } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const target = resolve(root, 'apps/miniprogram/miniprogram/domain')
await mkdir(target, { recursive: true })

for (const file of ['meal-editor.ts', 'phone.ts', 'profile-validation.ts']) {
  await copyFile(resolve(root, 'packages/domain/src', file), resolve(target, file))
}
