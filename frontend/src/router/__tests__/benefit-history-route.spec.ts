import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(testDirectory, '../index.ts'), 'utf8')
const sidebarSource = readFileSync(
  resolve(testDirectory, '../../components/layout/AppSidebar.vue'),
  'utf8'
)
const redeemViewSource = readFileSync(resolve(testDirectory, '../../views/user/RedeemView.vue'), 'utf8')

describe('benefit grant history navigation', () => {
  it('merges grant history into redeem and preserves the old URL as a redirect', () => {
    expect(redeemViewSource).toContain('<BenefitGrantHistory')
    expect(redeemViewSource).toContain(':redeem-history="history"')
    expect(redeemViewSource).toContain('@refresh-redeem-history="fetchHistory"')
    expect(sidebarSource).not.toContain("{ path: '/benefits'")
    expect(routerSource).toMatch(/path: '\/benefits',[\s\S]*?redirect: '\/redeem'/)
  })
})
