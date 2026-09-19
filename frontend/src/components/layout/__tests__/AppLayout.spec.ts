import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppLayout from '../AppLayout.vue'

const { settings } = vi.hoisted(() => ({ settings: { cachedPublicSettings: null as null | { console_ui_mode?: string } } }))
vi.mock('@/stores', () => ({ useAppStore: () => settings }))
vi.mock('../LegacyAppShell.vue', () => ({ default: { template: '<div data-shell="legacy"><slot /></div>' } }))
vi.mock('../ModernAppShell.vue', () => ({ default: { template: '<div data-shell="modern"><slot /></div>' } }))

describe('console mode selection', () => {
  it.each([
    [null, 'legacy'], [{}, 'legacy'],
    [{ console_ui_mode: 'unknown' }, 'legacy'],
    [{ console_ui_mode: 'legacy' }, 'legacy'],
    [{ console_ui_mode: 'modern' }, 'modern']
  ] as const)('uses %j with the shared page slot', (value, mode) => {
    settings.cachedPublicSettings = value
    const wrapper = mount(AppLayout, { slots: { default: '<p>Same business page</p>' } })
    expect(wrapper.get('[data-shell]').attributes('data-shell')).toBe(mode)
    expect(wrapper.text()).toBe('Same business page')
    wrapper.unmount()
  })
})
