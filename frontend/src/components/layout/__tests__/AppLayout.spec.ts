import { createPinia, setActivePinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

import AppLayout from '../AppLayout.vue'
import ModernAppShell from '../ModernAppShell.vue'
import { useAppStore } from '@/stores/app'

vi.mock('@/composables/useOnboardingTour', () => ({
  useOnboardingTour: () => ({ replayTour: vi.fn() }),
}))

const shellStubs = {
  LegacyAppShell: {
    template: '<section data-testid="legacy-shell"><slot /></section>',
  },
  ModernAppShell: {
    template: '<section data-testid="modern-shell"><slot /></section>',
  },
}

describe('AppLayout console mode selection', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it.each([
    ['missing settings', null],
    ['an unknown value', { console_ui_mode: 'future-mode' }],
    ['the explicit legacy value', { console_ui_mode: 'legacy' }],
  ])('keeps the legacy shell for %s', (_label, settings) => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = settings as never

    const wrapper = mount(AppLayout, {
      slots: { default: '<div data-testid="page-content" />' },
      global: { stubs: shellStubs },
    })

    expect(wrapper.find('[data-testid="legacy-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modern-shell"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="page-content"]').exists()).toBe(true)
  })

  it('switches the shared page slot between modern and legacy shells', async () => {
    const appStore = useAppStore()
    appStore.cachedPublicSettings = { console_ui_mode: 'modern' } as never

    const wrapper = mount(AppLayout, {
      slots: { default: '<div data-testid="page-content" />' },
      global: { stubs: shellStubs },
    })

    expect(wrapper.find('[data-testid="modern-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="page-content"]').exists()).toBe(true)

    appStore.cachedPublicSettings = { console_ui_mode: 'legacy' } as never
    await nextTick()

    expect(wrapper.find('[data-testid="legacy-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="modern-shell"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="page-content"]').exists()).toBe(true)
  })
})

describe('ModernAppShell document scope', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    delete document.documentElement.dataset.consoleUiMode
  })

  afterEach(() => {
    delete document.documentElement.dataset.consoleUiMode
  })

  it('scopes teleported surfaces while mounted and cleans up on unmount', () => {
    const wrapper = mount(ModernAppShell, {
      global: {
        stubs: {
          AppSidebar: true,
          AppHeader: true,
        },
      },
    })

    expect(document.documentElement.dataset.consoleUiMode).toBe('modern')
    expect(wrapper.attributes('data-console-shell')).toBe('modern')

    wrapper.unmount()

    expect(document.documentElement.dataset.consoleUiMode).toBeUndefined()
  })
})
