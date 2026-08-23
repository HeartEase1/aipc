import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))

function readSource(relativePath: string): string {
  return readFileSync(resolve(testDirectory, relativePath), 'utf8')
}

describe('console layout contracts', () => {
  it('keeps the mobile sidebar expanded independently of the desktop preference', () => {
    const source = readSource('../AppSidebar.vue')

    expect(source).toContain("useMediaQuery('(min-width: 1024px)')")
    expect(source).toContain('const effectiveSidebarCollapsed = computed(() => isDesktop.value && sidebarCollapsed.value)')
    expect(source).toContain('class="sidebar-link hidden w-full lg:flex"')
  })

  it('derives table pages from shell height and padding tokens', () => {
    const source = readSource('../TablePageLayout.vue')

    expect(source).toContain('var(--console-header-height, 64px)')
    expect(source).toContain('var(--console-page-vertical-padding, 4rem)')
    expect(source).toContain('100dvh')
    expect(source).toContain('layout-section-table')
    expect(source).toContain('grid-template-rows: auto auto minmax(0, 1fr) auto;')
    expect(source).toContain('height: var(--console-viewport-available-height);')
    expect(source).toMatch(
      /\.modern-console-shell \.table-page-layout\.mobile-mode \{[\s\S]*?height: auto;/,
    )
    expect(source).not.toContain(':global(.modern-console-shell)')
  })

  it('gives custom content and the playground a shared available-height contract', () => {
    const customPageSource = readSource('../../../views/user/CustomPageView.vue')
    const playgroundSource = readSource('../../../views/user/OnlinePlaygroundView.vue')

    expect(customPageSource).toContain(
      'var(--console-viewport-available-height, calc(100dvh - 8rem))',
    )
    expect(playgroundSource).toContain('var(--console-viewport-available-height, 100dvh)')
    expect(playgroundSource).toContain('.playground-workspace:fullscreen')
    expect(playgroundSource).toContain('height: 100dvh;')
  })

  it('aligns fixed prompt-audit actions to the effective sidebar width', () => {
    const source = readSource('../../../features/prompt-audit/PromptAuditView.vue')

    expect(source).toContain('prompt-audit-save-bar fixed inset-x-0 bottom-0')
    expect(source).toContain('left: var(--console-sidebar-effective-width, 16rem);')
    expect(source).not.toContain('lg:left-64')
  })

  it('keeps page grids in the shared modern layer instead of scoped global selectors', () => {
    const modernConsoleSource = readSource('../../../styles/modern-console.css')
    const profileSource = readSource('../../../views/user/ProfileView.vue')
    const redeemSource = readSource('../../../views/user/RedeemView.vue')

    expect(modernConsoleSource).toContain('.modern-console-shell .profile-shell')
    expect(modernConsoleSource).toContain('.modern-console-shell .redeem-page')
    expect(profileSource).not.toContain(':global(.modern-console-shell)')
    expect(redeemSource).not.toContain(':global(.modern-console-shell)')
  })

  it('keeps the modern profile hierarchy and jelly motion scoped and accessible', () => {
    const modernConsoleSource = readSource('../../../styles/modern-console.css')
    const profileSource = readSource('../../../views/user/ProfileView.vue')

    expect(profileSource).toContain('class="profile-primary-column min-w-0"')
    expect(profileSource).toContain('class="profile-security-column min-w-0 space-y-6"')
    expect(modernConsoleSource).toContain('@keyframes console-jelly-settle')
    expect(modernConsoleSource).toContain('@keyframes console-sidebar-jelly')
    expect(modernConsoleSource).toContain('--console-ease-jelly:')
    expect(modernConsoleSource).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?animation-duration: 1ms !important;/,
    )
  })

  it('keeps classic teal while modern console and auth surfaces inherit the blue primary palette', () => {
    const tailwindSource = readSource('../../../../tailwind.config.js')
    const globalStyles = readSource('../../../style.css')
    const modernConsoleSource = readSource('../../../styles/modern-console.css')
    const authLayoutSource = readSource('../AuthLayout.vue')

    expect(tailwindSource).toContain("rgb(var(--color-primary-500) / <alpha-value>)")
    expect(globalStyles).toContain('--color-primary-500: 20 184 166;')
    expect(modernConsoleSource).toContain('--color-primary-500: 59 130 246;')
    expect(authLayoutSource).toContain('--color-primary-500: 59 130 246;')
    expect(authLayoutSource).toContain('margin-block: auto;')
    expect(authLayoutSource).not.toContain('align-items: flex-start;')
  })
})
