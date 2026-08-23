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
const modernConsoleSource = readFileSync(
  resolve(testDirectory, '../../styles/modern-console.css'),
  'utf8'
)

describe('benefit grant history navigation', () => {
  it('merges grant history into redeem and preserves the old URL as a redirect', () => {
    expect(redeemViewSource).toContain('<BenefitGrantHistory')
    expect(redeemViewSource).toContain(':redeem-history="history"')
    expect(redeemViewSource).toContain('@refresh-redeem-history="fetchHistory"')
    expect(sidebarSource).not.toContain("{ path: '/benefits'")
    expect(routerSource).toMatch(/path: '\/benefits',[\s\S]*?redirect: '\/redeem'/)
  })

  it('keeps the classic redeem flow while scoping the two-column layout to modern mode', () => {
    expect(redeemViewSource).toContain(
      'class="redeem-page mx-auto flex max-w-2xl flex-col gap-6"'
    )
    expect(redeemViewSource).toContain('class="redeem-primary-column"')
    expect(redeemViewSource).toContain('class="redeem-secondary-column"')
    expect(redeemViewSource).not.toContain(':global(.modern-console-shell)')
    expect(modernConsoleSource).toContain('.modern-console-shell .redeem-page')
    expect(modernConsoleSource).toContain(
      'grid-template-columns: minmax(0, 0.86fr) minmax(0, 1.14fr)'
    )
  })
})
