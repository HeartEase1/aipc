import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import App from '@/App.vue'

const mocks = vi.hoisted(() => {
  let resolveBenefit: (() => void) | null = null
  const calls: string[] = []
  return {
    calls,
    resolveBenefit: () => resolveBenefit?.(),
    fetchUnread: vi.fn(() => {
      calls.push('benefit:start')
      return new Promise<void>((resolve) => {
        resolveBenefit = () => {
          calls.push('benefit:done')
          resolve()
        }
      })
    }),
    fetchAnnouncements: vi.fn(async () => {
      calls.push('announcement:start')
    }),
    reset() {
      resolveBenefit = null
      calls.length = 0
    },
  }
})

vi.mock('vue-router', () => ({
  RouterView: { template: '<div />' },
  useRouter: () => ({ afterEach: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ fullPath: '/benefits', path: '/benefits', meta: {} }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})
vi.mock('@/api/setup', () => ({ getSetupStatus: vi.fn().mockResolvedValue({ needs_setup: false }) }))
vi.mock('@/router/title', () => ({ resolveRouteDocumentTitle: () => 'IPCAI' }))
vi.mock('@/utils/branding', () => ({ updateFavicon: vi.fn() }))
vi.mock('@/stores', () => ({
  useAppStore: () => ({
    siteLogo: '',
    cachedPublicSettings: null,
    siteName: 'IPCAI',
    fetchPublicSettings: vi.fn().mockResolvedValue(null),
  }),
  useAuthStore: () => ({ isAuthenticated: true, isAdmin: false }),
  useSubscriptionStore: () => ({
    fetchActiveSubscriptions: vi.fn().mockResolvedValue([]),
    startPolling: vi.fn(),
    clear: vi.fn(),
  }),
  useAnnouncementStore: () => ({
    fetchAnnouncements: mocks.fetchAnnouncements,
    reset: vi.fn(),
  }),
  useAdminComplianceStore: () => ({
    fetchStatus: vi.fn().mockResolvedValue({ required: false }),
    reset: vi.fn(),
  }),
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
  useBenefitGrantStore: () => ({
    currentPopup: null,
    fetchUnread: mocks.fetchUnread,
    dismissPopup: vi.fn(),
    reset: vi.fn(),
  }),
}))

describe('App benefit notification ordering', () => {
  beforeEach(() => {
    mocks.fetchUnread.mockClear()
    mocks.fetchAnnouncements.mockClear()
    mocks.reset()
  })

  it('waits for benefit notifications before loading regular announcements', async () => {
    mount(App, {
      global: {
        stubs: {
          NavigationProgress: true,
          RouterView: true,
          Toast: true,
          AnnouncementPopup: true,
          AdminComplianceDialog: true,
        },
      },
    })
    await flushPromises()

    expect(mocks.calls).toEqual(['benefit:start'])
    expect(mocks.fetchAnnouncements).not.toHaveBeenCalled()

    mocks.resolveBenefit()
    await flushPromises()

    expect(mocks.calls).toEqual(['benefit:start', 'benefit:done', 'announcement:start'])
    expect(mocks.fetchAnnouncements).toHaveBeenCalledTimes(1)
  })
})
