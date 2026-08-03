import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const routerPath = resolve(dirname(fileURLToPath(import.meta.url)), '../index.ts')
const routerSource = readFileSync(routerPath, 'utf8')

describe('usage guide route', () => {
  it('registers an authenticated built-in Markdown page', () => {
    expect(routerSource).toContain("path: '/guide'")
    expect(routerSource).toContain("name: 'UsageGuide'")
    expect(routerSource).toContain("markdownUrl: '/tutorial/usage-guide.md'")
    expect(routerSource).toContain("assetBaseUrl: '/tutorial'")
  })
})
