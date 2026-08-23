import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'

import VersionBadge from '../VersionBadge.vue'

const stores = vi.hoisted(() => ({
  app: {
    versionLoading: false,
    currentVersion: '1.0.43',
    latestVersion: '1.0.44',
    hasUpdate: true,
    releaseInfo: null,
    buildType: 'release',
    fetchVersion: vi.fn(),
    clearVersionCache: vi.fn()
  },
  auth: {
    isAdmin: true
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => stores.app,
  useAuthStore: () => stores.auth
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn(),
  getRollbackVersions: vi.fn(),
  rollback: vi.fn()
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: ref(false),
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

afterEach(() => {
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('VersionBadge', () => {
  it('portals the update panel outside the clipped sidebar and keeps it inside the viewport', async () => {
    const wrapper = mount(VersionBadge, {
      props: { version: '1.0.43' },
      attachTo: document.body,
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    const trigger = wrapper.get('[data-testid="version-badge-trigger"]')
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      x: 220,
      y: 36,
      top: 36,
      left: 220,
      right: 276,
      bottom: 60,
      width: 56,
      height: 24,
      toJSON: () => ({})
    } as DOMRect)

    await trigger.trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('[data-testid="version-badge-dropdown"]')
    expect(dropdown).not.toBeNull()
    expect(dropdown?.parentElement).toBe(document.body)
    expect(dropdown?.classList.contains('fixed')).toBe(true)
    expect(dropdown?.classList.contains('z-[100000020]')).toBe(true)
    expect(dropdown?.style.top).toBe('68px')
    expect(Number.parseFloat(dropdown?.style.left || '0')).toBeGreaterThanOrEqual(8)
    expect(dropdown?.style.maxHeight).toContain('100dvh')

    wrapper.unmount()
  })
})
